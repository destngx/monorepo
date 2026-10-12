import Foundation

/// Kind of posture alert, ordered by severity of the forward-head alerts.
public enum PostureAlertKind: Equatable, Sendable {
    case mildForward
    case severeForward
    case overextension

    /// Maps the engine's sustained states to an alert. Forward-head alerts take precedence over overextension.
    public static func from(warningLevel: Int, isSustainedOverextension: Bool) -> PostureAlertKind? {
        switch warningLevel {
        case 2...: return .severeForward
        case 1: return .mildForward
        default: return isSustainedOverextension ? .overextension : nil
        }
    }
}

/// A notification the escalator decided to deliver.
public struct PostureAlertDelivery: Equatable, Sendable {
    public var kind: PostureAlertKind
    public var playSound: Bool
}

/// Escalation ladder for one drift episode (an episode starts when an alert appears and ends on recovery):
///
/// 1. Ambient: the alert is visible immediately as a menu bar tint and screen-edge glow (`ambient`).
/// 2. Notify: a silent notification once the alert persists (mild `mildNotifyAfterSeconds`, default 30 s; severe 10 s;
///    overextension 15 s),
///    repeated at most once per cooldown. Escalating from mild to severe bypasses the cooldown.
/// 3. Sound: severe drift held `severeSoundAfterSeconds` after a notification in the same episode, once per episode
///    (skipped when `isSoundEnabled` is false).
///
/// While silenced nothing is delivered; persistence keeps counting, so an alert still active when the silence ends
/// is delivered right away.
public struct AlertEscalator: Equatable, Sendable {
    public static let defaultMildNotifyAfterSeconds: TimeInterval = 30
    public static let severeNotifyAfterSeconds: TimeInterval = 10
    public static let overextensionNotifyAfterSeconds: TimeInterval = 15
    public static let severeSoundAfterSeconds: TimeInterval = 60

    public var mildNotifyAfterSeconds: TimeInterval
    public var isSoundEnabled: Bool
    public private(set) var ambient: PostureAlertKind?

    private var forwardSince: TimeInterval?
    private var severeSince: TimeInterval?
    private var overextensionSince: TimeInterval?
    private var lastNotificationAt: TimeInterval?
    private var lastNotifiedKind: PostureAlertKind?
    private var soundPlayedInEpisode = false

    public init(
        mildNotifyAfterSeconds: TimeInterval = AlertEscalator.defaultMildNotifyAfterSeconds,
        isSoundEnabled: Bool = true
    ) {
        self.mildNotifyAfterSeconds = mildNotifyAfterSeconds
        self.isSoundEnabled = isSoundEnabled
    }

    public mutating func update(
        kind: PostureAlertKind?,
        timestamp t: TimeInterval,
        isSilenced: Bool,
        cooldown: TimeInterval
    ) -> PostureAlertDelivery? {
        ambient = kind
        let isForward = kind == .mildForward || kind == .severeForward
        forwardSince = isForward ? (forwardSince ?? t) : nil
        severeSince = kind == .severeForward ? (severeSince ?? t) : nil
        overextensionSince = kind == .overextension ? (overextensionSince ?? t) : nil

        guard let kind else {
            // Episode over
            lastNotifiedKind = nil
            soundPlayedInEpisode = false
            return nil
        }
        guard !isSilenced else { return nil }

        func held(_ since: TimeInterval?, _ duration: TimeInterval) -> Bool {
            guard let since else { return false }
            return t - since >= duration - 1e-9
        }

        if isSoundEnabled,
           kind == .severeForward,
           lastNotifiedKind != nil,
           !soundPlayedInEpisode,
           held(severeSince, Self.severeSoundAfterSeconds) {
            soundPlayedInEpisode = true
            return deliver(kind, playSound: true, at: t)
        }

        let isDue: Bool
        switch kind {
        case .severeForward: isDue = held(severeSince, Self.severeNotifyAfterSeconds)
        case .mildForward: isDue = held(forwardSince, mildNotifyAfterSeconds)
        case .overextension: isDue = held(overextensionSince, Self.overextensionNotifyAfterSeconds)
        }
        guard isDue else { return nil }

        let escalated = kind == .severeForward && lastNotifiedKind != .severeForward
        let cooledDown = lastNotificationAt.map { t - $0 >= cooldown - 1e-9 } ?? true
        guard escalated || cooledDown else { return nil }
        return deliver(kind, playSound: false, at: t)
    }

    private mutating func deliver(_ kind: PostureAlertKind, playSound: Bool, at t: TimeInterval) -> PostureAlertDelivery {
        lastNotificationAt = t
        lastNotifiedKind = kind
        return PostureAlertDelivery(kind: kind, playSound: playSound)
    }
}
