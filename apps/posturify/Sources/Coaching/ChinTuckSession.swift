import Foundation

/// Guided chin tuck (deep cervical flexor) set with live feedback from the calibrated forward displacement.
///
/// Depth is the fraction of the personal correction range reached: 0 at the habitual neutral, 1 at the best upright
/// posture captured at calibration. One rep: glide back to `targetDepth`, hold `holdSeconds` with the head level,
/// then release to below `releaseDepth`. Nodding (pitch beyond `maxPitchDeviationDegrees`) or losing depth during a
/// hold restarts the hold.
public struct ChinTuckSession: Equatable, Sendable {
    public static let repsPerSet = 10
    public static let setsPerDay = 3
    public static let holdSeconds: TimeInterval = 5
    public static let targetDepth = 0.6
    /// Depth below which a hold is lost (hysteresis below `targetDepth`).
    public static let holdLossDepth = 0.45
    public static let releaseDepth = 0.25
    public static let maxPitchDeviationDegrees: Float = 10
    /// Light smoothing of the per-frame depth (weight of the newest frame).
    public static let depthSmoothing = 0.5

    public enum Stage: Equatable, Sendable {
        case retract
        case hold(since: TimeInterval)
        case release
        case complete
    }

    public enum Cue: Equatable, Sendable {
        case faceCamera
        case glideBack
        case keepHeadLevel
        case holding(remainingSeconds: Int)
        case release
        case setComplete

        public var text: String {
            switch self {
            case .faceCamera: return "Face the camera"
            case .glideBack: return "Glide your head straight back, chin level"
            case .keepHeadLevel: return "Keep your eyes level - glide back, don't nod"
            case .holding(let remaining): return "Hold... \(remaining)"
            case .release: return "Relax forward slowly"
            case .setComplete: return "Set complete - nice work"
            }
        }
    }

    public let correctionRange: Double
    public private(set) var stage: Stage = .retract
    public private(set) var completedReps = 0
    public private(set) var depth = 0.0
    public private(set) var cue: Cue = .glideBack
    public private(set) var holdProgress = 0.0
    private var isDepthInitialized = false

    /// - Parameter correctionRange: Calibrated distance from habitual neutral to best upright, in face lengths (> 0).
    public init(correctionRange: Double) {
        self.correctionRange = correctionRange
    }

    public var isComplete: Bool { stage == .complete }

    /// - Parameters:
    ///   - displacement: Forward displacement D_f in face lengths (negative is backward), nil when not tracked.
    ///   - deltaPitch: Head pitch relative to the calibrated neutral, in degrees.
    public mutating func update(displacement: Double?, deltaPitch: Float, timestamp t: TimeInterval) {
        guard !isComplete else { return }
        guard let displacement, correctionRange > 0 else {
            cue = .faceCamera
            return
        }

        let rawDepth = -displacement / correctionRange
        depth = isDepthInitialized ? Self.depthSmoothing * rawDepth + (1 - Self.depthSmoothing) * depth : rawDepth
        isDepthInitialized = true
        let isLevel = abs(deltaPitch) <= Self.maxPitchDeviationDegrees

        switch stage {
        case .retract:
            holdProgress = 0
            if !isLevel {
                cue = .keepHeadLevel
            } else if depth >= Self.targetDepth {
                stage = .hold(since: t)
                cue = .holding(remainingSeconds: Int(Self.holdSeconds))
            } else {
                cue = .glideBack
            }

        case .hold(let since):
            guard isLevel, depth >= Self.holdLossDepth else {
                stage = .retract
                holdProgress = 0
                cue = isLevel ? .glideBack : .keepHeadLevel
                return
            }
            let held = t - since
            holdProgress = min(1, held / Self.holdSeconds)
            if held >= Self.holdSeconds - 1e-9 {
                completedReps += 1
                stage = .release
                cue = .release
            } else {
                cue = .holding(remainingSeconds: Int((Self.holdSeconds - held).rounded(.up)))
            }

        case .release:
            holdProgress = 0
            guard depth <= Self.releaseDepth else {
                cue = .release
                return
            }
            if completedReps >= Self.repsPerSet {
                stage = .complete
                cue = .setComplete
            } else {
                stage = .retract
                cue = .glideBack
            }

        case .complete:
            break
        }
    }
}
