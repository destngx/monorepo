import Foundation

/// Why alerts are currently silenced, in priority order for display.
public enum AlertSilenceReason: Equatable, Sendable {
    case exercising
    case snoozed(until: Date)
    case call
    case videoMeeting
    case fullScreen

    public var displayText: String {
        switch self {
        case .exercising: return "Exercise in progress"
        case .snoozed(let until): return "Snoozed until \(until.formatted(date: .omitted, time: .shortened))"
        case .call: return "Microphone in use"
        case .videoMeeting: return "Camera in use by another app"
        case .fullScreen: return "Full-screen app in front"
        }
    }
}

/// Snapshot of system activity that should silence interruptive alerts.
public struct SystemActivity: Equatable, Sendable {
    public var isMicrophoneInUse: Bool
    public var isCameraInUseByAnotherApp: Bool
    public var isFullScreenAppActive: Bool

    public init(isMicrophoneInUse: Bool = false, isCameraInUseByAnotherApp: Bool = false, isFullScreenAppActive: Bool = false) {
        self.isMicrophoneInUse = isMicrophoneInUse
        self.isCameraInUseByAnotherApp = isCameraInUseByAnotherApp
        self.isFullScreenAppActive = isFullScreenAppActive
    }
}

/// Snooze options offered in the UI.
public enum SnoozeOption: CaseIterable, Identifiable, Sendable {
    case fifteenMinutes
    case oneHour
    case untilTomorrow

    public var id: Self { self }

    public var displayName: String {
        switch self {
        case .fifteenMinutes: return "15 Minutes"
        case .oneHour: return "1 Hour"
        case .untilTomorrow: return "Until Tomorrow"
        }
    }

    public var shortName: String {
        switch self {
        case .fifteenMinutes: return "15m"
        case .oneHour: return "1h"
        case .untilTomorrow: return "Today"
        }
    }

    public func endDate(from now: Date, calendar: Calendar = .current) -> Date {
        switch self {
        case .fifteenMinutes: return now.addingTimeInterval(15 * 60)
        case .oneHour: return now.addingTimeInterval(60 * 60)
        case .untilTomorrow:
            let startOfToday = calendar.startOfDay(for: now)
            return calendar.date(byAdding: .day, value: 1, to: startOfToday) ?? now.addingTimeInterval(24 * 60 * 60)
        }
    }
}

public enum AlertSilence {
    /// The highest-priority reason to silence alerts, or nil when alerts may interrupt.
    public static func reason(
        activity: SystemActivity,
        snoozedUntil: Date?,
        isExercising: Bool,
        now: Date
    ) -> AlertSilenceReason? {
        if isExercising { return .exercising }
        if let snoozedUntil, snoozedUntil > now { return .snoozed(until: snoozedUntil) }
        if activity.isMicrophoneInUse { return .call }
        if activity.isCameraInUseByAnotherApp { return .videoMeeting }
        if activity.isFullScreenAppActive { return .fullScreen }
        return nil
    }
}
