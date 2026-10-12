import Foundation

/// Tracks continuous seated time and decides when to prompt a movement break.
///
/// Seated time accumulates while a face is present. Being away for `awayResetSeconds` (no face, or no frames at all,
/// e.g. the camera was stopped) ends the seated session. A prompt is due after `interval` seated, then again every
/// `repeatSeconds` while still seated. Prompts are deferred while silenced.
public struct BreakMonitor: Equatable, Sendable {
    public static let awayResetSeconds: TimeInterval = 120
    public static let repeatSeconds: TimeInterval = 10 * 60
    public static let intervalOptions: [TimeInterval] = [30 * 60, 45 * 60, 60 * 60]
    public static let defaultInterval: TimeInterval = 45 * 60

    public var interval: TimeInterval

    public private(set) var seatedSince: TimeInterval?
    private var lastPresentAt: TimeInterval?
    private var nextPromptAt: TimeInterval?

    public init(interval: TimeInterval = BreakMonitor.defaultInterval) {
        self.interval = interval
    }

    public func seatedDuration(at t: TimeInterval) -> TimeInterval {
        seatedSince.map { max(0, t - $0) } ?? 0
    }

    /// Returns true when a movement break prompt should be delivered now.
    public mutating func update(isPresent: Bool, timestamp t: TimeInterval, isSilenced: Bool) -> Bool {
        if let lastPresentAt, t - lastPresentAt >= Self.awayResetSeconds {
            endSession()
        }
        guard isPresent else { return false }

        lastPresentAt = t
        if seatedSince == nil {
            seatedSince = t
            nextPromptAt = t + interval
        }
        guard !isSilenced, let due = nextPromptAt, t >= due else { return false }
        nextPromptAt = t + Self.repeatSeconds
        return true
    }

    /// The user moved (e.g. completed an exercise set): restart the seated session from now.
    public mutating func recordBreak(at t: TimeInterval) {
        endSession()
        lastPresentAt = t
        seatedSince = t
        nextPromptAt = t + interval
    }

    private mutating func endSession() {
        seatedSince = nil
        nextPromptAt = nil
        lastPresentAt = nil
    }
}
