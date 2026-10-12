import Foundation

/// ΔCVA alert thresholds (degrees, relative to the calibrated neutral) and how long each must be held.
public struct ForwardHeadAlertThresholds: Equatable, Sendable {
    public var mild: Double
    public var substantial: Double
    public var severe: Double
    /// ΔCVA above this counts as recovered; the band between `recovery` and `mild` holds the current level.
    public var recovery: Double

    public var mildHoldSeconds: TimeInterval
    public var substantialHoldSeconds: TimeInterval
    public var severeHoldSeconds: TimeInterval
    public var recoveryHoldSeconds: TimeInterval

    public init(
        mild: Double,
        substantial: Double,
        severe: Double,
        recovery: Double,
        mildHoldSeconds: TimeInterval,
        substantialHoldSeconds: TimeInterval,
        severeHoldSeconds: TimeInterval,
        recoveryHoldSeconds: TimeInterval
    ) {
        self.mild = mild
        self.substantial = substantial
        self.severe = severe
        self.recovery = recovery
        self.mildHoldSeconds = mildHoldSeconds
        self.substantialHoldSeconds = substantialHoldSeconds
        self.severeHoldSeconds = severeHoldSeconds
        self.recoveryHoldSeconds = recoveryHoldSeconds
    }
}

/// Preset user profile describing the user's existing posture.
///
/// The calibrated neutral of someone with forward head posture is already forward, so the profile sets the
/// assumed neutral CVA (CVA_0) and makes alerts progressively stricter: further drift from an already forward
/// neutral is flagged sooner and at smaller angles, with a longer cooldown to limit alert fatigue.
/// Profiles never change the learned calibration geometry, so switching does not require recalibrating.
public enum PostureProfile: String, CaseIterable, Identifiable, Sendable {
    case goodPosture
    case slightTurtleNeck
    case moreTurtleNeck

    public static let `default` = PostureProfile.goodPosture

    public var id: String { rawValue }

    public var displayName: String {
        switch self {
        case .goodPosture: return "Good Posture"
        case .slightTurtleNeck: return "Slight Turtle Neck"
        case .moreTurtleNeck: return "More Turtle Neck"
        }
    }

    public var summary: String {
        switch self {
        case .goodPosture: return "Upright neutral posture. Standard drift alerts."
        case .slightTurtleNeck: return "Head already slightly forward. Earlier, stricter alerts."
        case .moreTurtleNeck: return "Head clearly forward. Strictest alerts on any further drift."
        }
    }

    /// Assumed CVA of the calibrated neutral posture (degrees).
    public var baseCvaDegrees: Double {
        switch self {
        case .goodPosture: return 55.0
        case .slightTurtleNeck: return 48.0
        case .moreTurtleNeck: return 42.0
        }
    }

    public var alertThresholds: ForwardHeadAlertThresholds {
        switch self {
        case .goodPosture:
            return ForwardHeadAlertThresholds(
                mild: -5.0, substantial: -10.0, severe: -15.0, recovery: -2.5,
                mildHoldSeconds: 3.0, substantialHoldSeconds: 4.0, severeHoldSeconds: 1.5, recoveryHoldSeconds: 3.0
            )
        case .slightTurtleNeck:
            return ForwardHeadAlertThresholds(
                mild: -4.0, substantial: -8.0, severe: -12.0, recovery: -2.0,
                mildHoldSeconds: 2.5, substantialHoldSeconds: 3.5, severeHoldSeconds: 1.5, recoveryHoldSeconds: 3.0
            )
        case .moreTurtleNeck:
            return ForwardHeadAlertThresholds(
                mild: -3.0, substantial: -6.0, severe: -10.0, recovery: -1.5,
                mildHoldSeconds: 2.0, substantialHoldSeconds: 3.0, severeHoldSeconds: 1.5, recoveryHoldSeconds: 3.0
            )
        }
    }

    /// Whether the progressive rehab target applies. A good-posture neutral needs no correction, so its target
    /// stays at the calibrated neutral.
    public var isRehabEligible: Bool {
        self != .goodPosture
    }

    /// Minimum seconds between repeated notifications (escalations bypass it).
    public var notificationCooldown: TimeInterval {
        switch self {
        case .goodPosture: return 30.0
        case .slightTurtleNeck: return 45.0
        case .moreTurtleNeck: return 60.0
        }
    }
}
