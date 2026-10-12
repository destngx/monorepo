import Foundation

/// Whether the committed calibration still describes the current setup.
public enum CalibrationHealth: Equatable, Sendable {
    case good
    case stale(StaleCalibrationReason)

    public var isStale: Bool {
        if case .stale = self { return true }
        return false
    }
}

public enum StaleCalibrationReason: Equatable, Sendable {
    /// The head sits persistently off the calibrated forward axis (camera, chair or seating position moved).
    case positionShifted
    /// The quality gate keeps failing (lighting, framing or face/body disagreement).
    case unreliableTracking
    /// Frames no longer share the calibration's image geometry (camera format changed).
    case cameraGeometryChanged

    public var displayText: String {
        switch self {
        case .positionShifted: return "Your seating or camera position seems to have changed."
        case .unreliableTracking: return "Tracking has been unreliable for a while (lighting or framing changed)."
        case .cameraGeometryChanged: return "The camera format changed since calibration."
        }
    }
}

/// Detects a calibration that no longer matches the setup, from frames with a detected face only (absence is never
/// evidence). Signals are smoothed with a time-based exponential average and must persist before the calibration is
/// declared stale, so leaning aside briefly never triggers it. Stale state is sticky until `reset()`.
public struct CalibrationHealthMonitor: Equatable, Sendable {
    public static let averagingSeconds: TimeInterval = 60
    /// Displacement orthogonal to the forward axis, in face lengths (~6 cm).
    public static let maxOrthogonalOffset = 0.3
    public static let positionShiftHoldSeconds: TimeInterval = 3 * 60
    public static let maxQualityFailureRate = 0.7
    public static let unreliableTrackingHoldSeconds: TimeInterval = 3 * 60
    public static let geometryMismatchHoldSeconds: TimeInterval = 10
    /// Frames further apart than this restart the averages (the user was away or the pipeline was paused).
    public static let maxFrameGapSeconds: TimeInterval = 30

    public private(set) var health: CalibrationHealth = .good

    private var lastTimestamp: TimeInterval?
    private var averageOffset = 0.0
    private var averageFailure = 0.0
    private var offsetExceededSince: TimeInterval?
    private var failureExceededSince: TimeInterval?
    private var geometryMismatchSince: TimeInterval?

    public init() {}

    public mutating func reset() {
        self = CalibrationHealthMonitor()
    }

    /// A comparable frame: `orthogonalOffset` is |q - D_f g| in face lengths.
    @discardableResult
    public mutating func recordFrame(orthogonalOffset: Double, isQualityPass: Bool, timestamp t: TimeInterval) -> CalibrationHealth {
        geometryMismatchSince = nil
        let weight = averagingWeight(at: t)
        averageOffset += weight * (orthogonalOffset - averageOffset)
        averageFailure += weight * ((isQualityPass ? 0.0 : 1.0) - averageFailure)

        offsetExceededSince = averageOffset > Self.maxOrthogonalOffset ? (offsetExceededSince ?? t) : nil
        failureExceededSince = averageFailure > Self.maxQualityFailureRate ? (failureExceededSince ?? t) : nil

        if let since = offsetExceededSince, t - since >= Self.positionShiftHoldSeconds {
            markStale(.positionShifted)
        } else if let since = failureExceededSince, t - since >= Self.unreliableTrackingHoldSeconds {
            markStale(.unreliableTracking)
        }
        return health
    }

    /// A frame with a face whose image geometry does not match the calibration.
    @discardableResult
    public mutating func recordGeometryMismatch(timestamp t: TimeInterval) -> CalibrationHealth {
        let since = geometryMismatchSince ?? t
        geometryMismatchSince = since
        lastTimestamp = t
        if t - since >= Self.geometryMismatchHoldSeconds {
            markStale(.cameraGeometryChanged)
        }
        return health
    }

    private mutating func averagingWeight(at t: TimeInterval) -> Double {
        defer { lastTimestamp = t }
        guard let last = lastTimestamp, t - last <= Self.maxFrameGapSeconds else {
            // First frame or after a long gap: restart the averages and the persistence timers
            averageOffset = 0.0
            averageFailure = 0.0
            offsetExceededSince = nil
            failureExceededSince = nil
            return 1.0
        }
        let dt = max(0.0, t - last)
        return 1.0 - exp(-dt / Self.averagingSeconds)
    }

    private mutating func markStale(_ reason: StaleCalibrationReason) {
        guard !health.isStale else { return }
        health = .stale(reason)
    }
}
