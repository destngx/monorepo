import Foundation
import QuartzCore
import os

/// A message the coach wants delivered as a system notification.
public struct CoachNotification: Equatable, Sendable {
    public var title: String
    public var subtitle: String
    public var body: String
    public var playSound: Bool
}

/// Main-actor coordinator of everything that talks to the user about posture:
/// - A1: escalating alerts (ambient glow, silent notification, sound) with context-aware silence and snooze
/// - A2: guided chin tuck sets
/// - A3: movement-break prompts
/// - A4: stale-calibration prompts
///
/// Frame-driven work runs in `process(_:)` with the frame's monotonic timestamp; time-driven work (system activity,
/// snooze expiry, calibration health, ambient timeout) runs in `tick(now:mediaTime:)` about once per second.
@MainActor
public final class PostureCoach: ObservableObject {
    /// Without a trusted frame for this long (user away, face lost) the ambient alert is cleared.
    public static let ambientTimeoutSeconds: TimeInterval = 10
    public static let activityRefreshSeconds: TimeInterval = 5

    @Published public private(set) var ambientAlert: PostureAlertKind?
    @Published public private(set) var silenceReason: AlertSilenceReason?
    @Published public private(set) var snoozedUntil: Date?
    @Published public private(set) var calibrationHealth: CalibrationHealth = .good
    @Published public private(set) var isBreakDue = false
    @Published public private(set) var seatedSeconds: TimeInterval = 0
    @Published public private(set) var chinTuckSession: ChinTuckSession?
    @Published public private(set) var chinTuckSetsToday = 0

    /// Alert channel preferences (persisted).
    @Published public var settings: AlertSettings {
        didSet {
            guard settings != oldValue else { return }
            saveSettings()
            applySettings()
        }
    }

    public var isExercising: Bool { chinTuckSession.map { !$0.isComplete } ?? false }

    /// How long each alert kind is shown when previewing the glow.
    public static let glowPreviewStepSeconds: TimeInterval = 2

    private enum Keys {
        static let alertSettings = "alertSettings"
        static let chinTuckSetsDay = "chinTuckSetsDay"
        static let chinTuckSetsCount = "chinTuckSetsCount"
    }

    private static let logger = Logger(subsystem: "com.destngx.posturify", category: "Coach")

    private let engine: CalibrationEngine
    private let defaults: UserDefaults
    private let notify: (CoachNotification) -> Void
    private let readActivity: () -> SystemActivity
    private let presentGlow: (PostureAlertKind?, EdgeGlowStyle) -> Void
    private let calendar: Calendar

    private var escalator: AlertEscalator
    private var breakMonitor: BreakMonitor
    private var glowPreviewTask: Task<Void, Never>?
    private var activity = SystemActivity()
    private var lastActivityRefresh: TimeInterval?
    private var lastTrustedFrameAt: TimeInterval?
    private var lastNotifiedHealth: CalibrationHealth = .good

    public init(
        engine: CalibrationEngine,
        defaults: UserDefaults,
        calendar: Calendar = .current,
        notify: @escaping (CoachNotification) -> Void,
        readActivity: @escaping () -> SystemActivity,
        presentGlow: @escaping (PostureAlertKind?, EdgeGlowStyle) -> Void
    ) {
        self.engine = engine
        self.defaults = defaults
        self.calendar = calendar
        self.notify = notify
        self.readActivity = readActivity
        self.presentGlow = presentGlow

        let settings = Self.loadSettings(from: defaults)
        self.settings = settings
        escalator = Self.makeEscalator(settings)
        breakMonitor = BreakMonitor(interval: settings.breakInterval)
        chinTuckSetsToday = storedSetsToday(now: Date())
    }

    private var isSilenced: Bool { silenceReason != nil }

    // MARK: - Frames

    public func process(_ state: SkeletalKinematicState) {
        let t = state.timestamp

        if let session = chinTuckSession, !session.isComplete {
            advanceChinTucks(state, at: t)
            return
        }

        if settings.breakRemindersEnabled {
            if breakMonitor.update(isPresent: state.isFacePresent, timestamp: t, isSilenced: isSilenced) {
                isBreakDue = true
                notify(Self.breakNotification(seatedSeconds: breakMonitor.seatedDuration(at: t)))
            }
            seatedSeconds = breakMonitor.seatedDuration(at: t)
            if breakMonitor.seatedSince == nil { isBreakDue = false }
        }

        guard let relative = state.relative, state.isFacePresent, relative.quality.isPass else { return }
        lastTrustedFrameAt = t
        let kind = PostureAlertKind.from(
            warningLevel: relative.warningLevel,
            isSustainedOverextension: relative.isSustainedOverextension
        )
        let delivery = escalator.update(
            kind: kind,
            timestamp: t,
            isSilenced: isSilenced,
            cooldown: relative.profile.notificationCooldown
        )
        setAmbient(escalator.ambient)
        if let delivery, settings.notificationsEnabled {
            notify(Self.alertNotification(delivery, relative: relative))
        }
    }

    // MARK: - Time

    public func tick(now: Date, mediaTime: TimeInterval) {
        if lastActivityRefresh.map({ mediaTime - $0 >= Self.activityRefreshSeconds }) ?? true {
            activity = readActivity()
            lastActivityRefresh = mediaTime
        }
        if let until = snoozedUntil, until <= now {
            snoozedUntil = nil
        }
        let reason = AlertSilence.reason(activity: activity, snoozedUntil: snoozedUntil, isExercising: isExercising, now: now)
        if reason != silenceReason {
            silenceReason = reason
            presentAmbient()
        }

        if let last = lastTrustedFrameAt, mediaTime - last >= Self.ambientTimeoutSeconds, ambientAlert != nil {
            _ = escalator.update(kind: nil, timestamp: mediaTime, isSilenced: true, cooldown: 0)
            setAmbient(nil)
        }

        let health = engine.baseline.isCalibrated ? engine.calibrationHealth : .good
        if health != calibrationHealth {
            calibrationHealth = health
        }
        if case .stale(let reason) = health, lastNotifiedHealth != health {
            lastNotifiedHealth = health
            notify(CoachNotification(
                title: "Calibration May Be Out of Date",
                subtitle: "",
                body: "\(reason.displayText) Recalibrate so posture alerts stay accurate.",
                playSound: false
            ))
        } else if !health.isStale {
            lastNotifiedHealth = .good
        }

        let today = storedSetsToday(now: now)
        if today != chinTuckSetsToday { chinTuckSetsToday = today }
    }

    // MARK: - Snooze

    public func snooze(_ option: SnoozeOption, now: Date = Date()) {
        snoozedUntil = option.endDate(from: now, calendar: calendar)
        refreshSilence(now: now)
    }

    public func cancelSnooze(now: Date = Date()) {
        snoozedUntil = nil
        refreshSilence(now: now)
    }

    // MARK: - Calibration Health

    public func dismissStaleCalibration() {
        engine.dismissStaleCalibration()
        calibrationHealth = .good
        lastNotifiedHealth = .good
    }

    // MARK: - Breaks

    public func acknowledgeBreak() {
        isBreakDue = false
    }

    // MARK: - Chin Tucks

    /// Chin tucks need the correction range captured by the best-upright calibration step.
    public var canStartChinTucks: Bool {
        engine.baseline.isCalibrated && engine.baseline.correctionRange > 0
    }

    public func startChinTucks(now: Date = Date()) {
        guard canStartChinTucks else {
            Self.logger.info("Chin tucks unavailable: no calibrated correction range")
            return
        }
        chinTuckSession = ChinTuckSession(correctionRange: engine.baseline.correctionRange)
        escalator = Self.makeEscalator(settings)
        setAmbient(nil)
        refreshSilence(now: now)
    }

    /// Closes the exercise view, whether the set is complete or abandoned.
    public func endChinTucks(now: Date = Date()) {
        guard chinTuckSession != nil else { return }
        chinTuckSession = nil
        // Deliberate exercise movements must not leave alert timers or smoothing behind
        engine.resetAlertTracking()
        _ = engine.drainAdherence()
        refreshSilence(now: now)
    }

    private func advanceChinTucks(_ state: SkeletalKinematicState, at t: TimeInterval) {
        guard var session = chinTuckSession else { return }
        let displacement = state.isFacePresent ? state.relative?.displacementFace : nil
        session.update(displacement: displacement, deltaPitch: state.relative?.deltaPitch ?? 0, timestamp: t)
        chinTuckSession = session
        guard session.isComplete else { return }

        recordCompletedSet(now: Date())
        breakMonitor.recordBreak(at: t)
        isBreakDue = false
        seatedSeconds = 0
        refreshSilence(now: Date())
    }

    private func recordCompletedSet(now: Date) {
        let count = storedSetsToday(now: now) + 1
        defaults.set(RehabProgram.dayNumber(for: now, calendar: calendar), forKey: Keys.chinTuckSetsDay)
        defaults.set(count, forKey: Keys.chinTuckSetsCount)
        chinTuckSetsToday = count
    }

    private func storedSetsToday(now: Date) -> Int {
        guard defaults.object(forKey: Keys.chinTuckSetsDay) != nil,
              defaults.integer(forKey: Keys.chinTuckSetsDay) == RehabProgram.dayNumber(for: now, calendar: calendar) else {
            return 0
        }
        return defaults.integer(forKey: Keys.chinTuckSetsCount)
    }

    // MARK: - Settings

    private func applySettings() {
        escalator.mildNotifyAfterSeconds = settings.notifyAfter.seconds
        escalator.isSoundEnabled = settings.severeSoundEnabled
        breakMonitor.interval = settings.breakInterval
        if !settings.breakRemindersEnabled { isBreakDue = false }
        if !settings.edgeGlowEnabled { cancelGlowPreview() }
        presentAmbient()
    }

    private static func makeEscalator(_ settings: AlertSettings) -> AlertEscalator {
        AlertEscalator(mildNotifyAfterSeconds: settings.notifyAfter.seconds, isSoundEnabled: settings.severeSoundEnabled)
    }

    private func saveSettings() {
        do {
            defaults.set(try JSONEncoder().encode(settings), forKey: Keys.alertSettings)
        } catch {
            Self.logger.error("Failed to save alert settings: \(error.localizedDescription, privacy: .public)")
        }
    }

    private static func loadSettings(from defaults: UserDefaults) -> AlertSettings {
        guard let data = defaults.data(forKey: Keys.alertSettings) else { return AlertSettings() }
        do {
            return try JSONDecoder().decode(AlertSettings.self, from: data)
        } catch {
            logger.error("Discarding unreadable alert settings: \(error.localizedDescription, privacy: .public)")
            return AlertSettings()
        }
    }

    // MARK: - Glow Preview

    public var isPreviewingGlow: Bool { glowPreviewTask != nil }

    /// Shows the glow for each alert kind in turn with the current style, then returns to the live state.
    public func previewGlow() {
        cancelGlowPreview()
        let kinds: [PostureAlertKind] = [.mildForward, .severeForward, .overextension]
        glowPreviewTask = Task { @MainActor [weak self] in
            for kind in kinds {
                guard let self, !Task.isCancelled else { return }
                self.presentGlow(kind, self.settings.edgeGlow)
                do {
                    try await Task.sleep(for: .seconds(Self.glowPreviewStepSeconds))
                } catch {
                    return // Cancelled: the canceller restores the live state
                }
            }
            guard let self else { return }
            self.glowPreviewTask = nil
            self.objectWillChange.send()
            self.presentAmbient()
        }
        objectWillChange.send()
    }

    private func cancelGlowPreview() {
        guard let task = glowPreviewTask else { return }
        task.cancel()
        glowPreviewTask = nil
        objectWillChange.send()
        presentAmbient()
    }

    // MARK: - Presentation

    private func refreshSilence(now: Date) {
        let reason = AlertSilence.reason(activity: activity, snoozedUntil: snoozedUntil, isExercising: isExercising, now: now)
        guard reason != silenceReason else { return }
        silenceReason = reason
        presentAmbient()
    }

    private func setAmbient(_ kind: PostureAlertKind?) {
        guard kind != ambientAlert else { return }
        ambientAlert = kind
        presentAmbient()
    }

    /// The menu bar tint always follows `ambientAlert`; the screen-edge glow is visible to others (screen sharing),
    /// so it is hidden while silenced.
    private func presentAmbient() {
        guard glowPreviewTask == nil else { return }
        presentGlow(settings.edgeGlowEnabled && !isSilenced ? ambientAlert : nil, settings.edgeGlow)
    }

    // MARK: - Messages

    static func alertNotification(_ delivery: PostureAlertDelivery, relative: RelativeCVAResult) -> CoachNotification {
        switch delivery.kind {
        case .severeForward:
            return CoachNotification(
                title: "Turtle Neck Posture",
                subtitle: String(format: "Head forward: ΔCVA %.1f°", relative.deltaCVA),
                body: "Your head has drifted well forward. Glide your chin back and stack your ears over your shoulders.",
                playSound: delivery.playSound
            )
        case .mildForward:
            return CoachNotification(
                title: "Posture Reminder",
                subtitle: String(format: "ΔCVA %.1f°", relative.deltaCVA),
                body: "Your head is creeping toward the screen. Sit tall and bring your head back over your shoulders.",
                playSound: delivery.playSound
            )
        case .overextension:
            return CoachNotification(
                title: "Head Tilted Back",
                subtitle: String(format: "ΔPitch %+.1f°", relative.deltaPitch),
                body: "Your head is tilted back. Lower your chin to a level gaze to unload the back of your neck.",
                playSound: delivery.playSound
            )
        }
    }

    static func breakNotification(seatedSeconds: TimeInterval) -> CoachNotification {
        CoachNotification(
            title: "Time to Move",
            subtitle: "Seated for \(Int(seatedSeconds / 60)) min",
            body: "Stand up and walk for a minute, or open Posturify for a set of chin tucks.",
            playSound: false
        )
    }
}
