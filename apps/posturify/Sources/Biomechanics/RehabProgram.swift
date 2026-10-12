import Foundation

/// Time spent tracked with a trusted measurement, and the part of it spent at or better than the alert target.
public struct PostureAdherence: Codable, Equatable, Sendable {
    public var trackedSeconds: TimeInterval
    public var onTargetSeconds: TimeInterval

    public init(trackedSeconds: TimeInterval = 0.0, onTargetSeconds: TimeInterval = 0.0) {
        self.trackedSeconds = trackedSeconds
        self.onTargetSeconds = onTargetSeconds
    }

    public var onTargetFraction: Double {
        trackedSeconds > 0 ? onTargetSeconds / trackedSeconds : 0.0
    }

    public static func + (lhs: PostureAdherence, rhs: PostureAdherence) -> PostureAdherence {
        PostureAdherence(
            trackedSeconds: lhs.trackedSeconds + rhs.trackedSeconds,
            onTargetSeconds: lhs.onTargetSeconds + rhs.onTargetSeconds
        )
    }
}

/// Progressive rehab target for users with forward head posture.
///
/// The alert target starts at the habitual neutral (stage 0) and moves toward the best upright posture captured at
/// calibration in steps of `stepFraction` of the correction range. Asking for the full correction at once causes
/// alert fatigue and neck strain, so the target changes at most once per `evaluationWindowDays`, based on daily
/// adherence over the preceding window:
/// - advance when at least `advanceDayRatio` of the active days were good days (mostly on target)
/// - step back when at least `stepBackDayRatio` of the active days were poor days (target too ambitious)
/// Days are calendar day numbers (see `dayNumber(for:calendar:)`), so the program is independent of time zones
/// changing between launches.
public struct RehabProgram: Codable, Equatable, Sendable {
    public static let stepFraction = 0.15
    /// Holding the full chin tuck all day is not sustainable; the program tops out at this fraction of the range.
    public static let maxTargetFraction = 0.75
    public static let evaluationWindowDays = 7
    /// A day counts toward the evaluation only with at least this much trusted tracking time.
    public static let minActiveSecondsPerDay: TimeInterval = 30 * 60
    public static let minActiveDays = 4
    /// Good day: on target for at least this fraction of tracked time. Poor day: below `poorDayOnTargetFraction`.
    public static let goodDayOnTargetFraction = 0.8
    public static let poorDayOnTargetFraction = 0.5
    public static let advanceDayRatio = 0.75
    public static let stepBackDayRatio = 0.5
    /// Daily history kept for evaluation and display.
    public static let historyDays = 28

    public static var maxStage: Int { Int((maxTargetFraction / stepFraction).rounded()) }

    public enum Change: Equatable, Sendable {
        case advanced
        case steppedBack
    }

    public private(set) var stage: Int
    /// Day the current stage started; nil until the program first sees a day.
    public private(set) var stageStartDay: Int?
    public private(set) var dailyAdherence: [Int: PostureAdherence]

    public init(stage: Int = 0, stageStartDay: Int? = nil, dailyAdherence: [Int: PostureAdherence] = [:]) {
        self.stage = min(Self.maxStage, max(0, stage))
        self.stageStartDay = stageStartDay
        self.dailyAdherence = dailyAdherence
    }

    private enum CodingKeys: String, CodingKey {
        case stage, stageStartDay, dailyAdherence
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        self.init(
            stage: try container.decode(Int.self, forKey: .stage),
            stageStartDay: try container.decodeIfPresent(Int.self, forKey: .stageStartDay),
            dailyAdherence: try container.decode([Int: PostureAdherence].self, forKey: .dailyAdherence)
        )
    }

    /// Fraction of the correction range the current target asks for.
    public var targetFraction: Double {
        min(Self.maxTargetFraction, Double(stage) * Self.stepFraction)
    }

    /// Consecutive number of the local calendar day containing `date`: the local year/month/day mapped onto UTC days,
    /// so daylight-saving transitions never merge or skip days.
    public static func dayNumber(for date: Date, calendar: Calendar = .current) -> Int {
        let local = calendar.dateComponents([.year, .month, .day], from: date)
        var utc = Calendar(identifier: .gregorian)
        utc.timeZone = TimeZone(identifier: "UTC")!
        let midnight = utc.date(from: DateComponents(year: local.year, month: local.month, day: local.day))!
        return Int((midnight.timeIntervalSinceReferenceDate / 86_400).rounded(.down))
    }

    public mutating func record(_ adherence: PostureAdherence, day: Int) {
        if stageStartDay == nil { stageStartDay = day }
        dailyAdherence[day, default: PostureAdherence()] = dailyAdherence[day, default: PostureAdherence()] + adherence
        dailyAdherence = dailyAdherence.filter { $0.key > day - Self.historyDays }
    }

    /// Evaluates the completed days of the last window and moves the target at most one stage.
    /// Call on each day (repeated calls are harmless: a change restarts the window).
    @discardableResult
    public mutating func evaluate(today: Int) -> Change? {
        guard let start = stageStartDay else {
            stageStartDay = today
            return nil
        }
        guard today - start >= Self.evaluationWindowDays else { return nil }

        let window = (today - Self.evaluationWindowDays)..<today
        let activeDays = window.compactMap { dailyAdherence[$0] }.filter { $0.trackedSeconds >= Self.minActiveSecondsPerDay }
        guard activeDays.count >= Self.minActiveDays else { return nil }

        let active = Double(activeDays.count)
        let goodDays = Double(activeDays.filter { $0.onTargetFraction >= Self.goodDayOnTargetFraction }.count)
        let poorDays = Double(activeDays.filter { $0.onTargetFraction < Self.poorDayOnTargetFraction }.count)

        if goodDays >= Self.advanceDayRatio * active, stage < Self.maxStage {
            stage += 1
            stageStartDay = today
            return .advanced
        }
        if poorDays >= Self.stepBackDayRatio * active, stage > 0 {
            stage -= 1
            stageStartDay = today
            return .steppedBack
        }
        return nil
    }

    /// Restarts the program at the habitual neutral, keeping the adherence history.
    public mutating func reset(today: Int) {
        stage = 0
        stageStartDay = today
    }
}
