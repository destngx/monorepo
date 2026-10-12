import Foundation
import CoreGraphics
import Vision
import QuartzCore
import simd

/// Sustained forward-head alert state machine with hysteresis.
///
/// Each threshold keeps its own "continuously beyond since" timer, which is cleared as soon as ΔCVA moves back
/// across that threshold. Nested thresholds mean oscillating around an inner threshold (e.g. -10°) still keeps
/// the outer (-5°) timer running, while a brief dip never leaves a stale timer behind.
///
/// - Level 2: ΔCVA <= severe held for its hold time, or <= substantial held for its hold time
/// - Level 1: ΔCVA <= mild held for its hold time
/// - Level 0: ΔCVA > recovery held for its hold time (the mild...recovery buffer zone holds the current level)
public struct ForwardHeadAlertHysteresis: Equatable, Sendable {
    public let thresholds: ForwardHeadAlertThresholds

    public private(set) var level: Int = 0
    private var mildSince: TimeInterval?
    private var substantialSince: TimeInterval?
    private var severeSince: TimeInterval?
    private var recoverySince: TimeInterval?

    public init(thresholds: ForwardHeadAlertThresholds = PostureProfile.default.alertThresholds) {
        self.thresholds = thresholds
    }

    public mutating func update(deltaCva: Double, timestamp: TimeInterval) -> Int {
        let t = thresholds
        mildSince = deltaCva <= t.mild ? (mildSince ?? timestamp) : nil
        substantialSince = deltaCva <= t.substantial ? (substantialSince ?? timestamp) : nil
        severeSince = deltaCva <= t.severe ? (severeSince ?? timestamp) : nil
        recoverySince = deltaCva > t.recovery ? (recoverySince ?? timestamp) : nil

        func held(_ since: TimeInterval?, _ duration: TimeInterval) -> Bool {
            guard let since else { return false }
            // Small epsilon absorbs floating-point accumulation in frame timestamps
            return timestamp - since >= duration - 1e-9
        }

        if held(severeSince, t.severeHoldSeconds) || held(substantialSince, t.substantialHoldSeconds) {
            level = 2
        } else if held(mildSince, t.mildHoldSeconds) {
            level = max(level, 1)
        } else if held(recoverySince, t.recoveryHoldSeconds) {
            level = 0
        }
        return level
    }
}

/// Tracks whether a condition has held continuously for `holdSeconds`.
public struct SustainedCondition: Equatable, Sendable {
    public let holdSeconds: TimeInterval
    private var since: TimeInterval?

    public init(holdSeconds: TimeInterval) {
        self.holdSeconds = holdSeconds
    }

    public mutating func update(_ isActive: Bool, timestamp: TimeInterval) -> Bool {
        since = isActive ? (since ?? timestamp) : nil
        guard let since else { return false }
        // Small epsilon absorbs floating-point accumulation in frame timestamps
        return timestamp - since >= holdSeconds - 1e-9
    }
}

/// Calibration state and engine for learning the personalized forward-direction vector g and noise covariance Sigma.
///
/// Flow: (1) hold neutral, (2) move the head forward and hold - collection starts only once the head is clearly
/// displaced and settled, and restarts if the hold breaks, (3) sit tall with the chin tucked and hold - records the
/// best upright displacement (the correction range used by the rehab target), (4) return to neutral - the learned
/// model must map the neutral pose back near D = 0 before it is committed. Each phase times out into `.failed`,
/// except the best-upright phase, which times out into continuing without a correction range.
/// The new model is built in a pending baseline and only replaces `baseline` after validation; evaluation is
/// suspended while calibrating so deliberate calibration movements never raise alerts.
///
/// Thread-safe: frames are recorded/evaluated on the camera queue while the UI starts, resets and reads
/// calibration state from the main thread. All mutable state is guarded by `lock`.
public final class CalibrationEngine: @unchecked Sendable {
    public static let shared = CalibrationEngine()

    public enum CalibrationPhase: String, Sendable {
        case idle = "Idle"
        case collectingNeutral = "1/4: Sit Neutral (Keep head straight)"
        case collectingForward = "2/4: Move Head Forward and Hold"
        case collectingUpright = "3/4: Sit Tall - Tuck Chin Back and Hold"
        case validating = "4/4: Relax Back to Neutral"
        case calibrated = "Calibrated"
        case failed = "Calibration Failed - Try Again"
    }

    public static let neutralSampleCount = 20
    /// Frames skipped after the forward movement is first detected, so the transition is not learned.
    public static let forwardSettleFrameCount = 5
    public static let forwardSampleCount = 25
    /// Consecutive neutral frames required to validate the learned model.
    public static let validationFrameCount = 10
    /// Maximum duration of any single calibration phase.
    public static let phaseTimeoutSeconds = 20.0
    /// Frames skipped after the upright retraction is first detected, and frames averaged for the best upright pose.
    public static let uprightSettleFrameCount = 5
    public static let uprightSampleCount = 15
    /// The best-upright phase is optional: after this long without a clear retraction, calibration continues
    /// without a correction range instead of failing.
    public static let uprightPhaseTimeoutSeconds = 12.0

    /// Forward movement must exceed both a noise-normalized (Mahalanobis) distance and an absolute distance in
    /// face lengths (~2.5 cm) to count as a deliberate forward hold.
    public static let minForwardMahalanobis = 6.0
    public static let minForwardMagnitude = 0.12
    /// Validation passes while |D_f| stays within this fraction of the calibrated forward excursion.
    public static let validationToleranceFraction = 0.3
    /// Backward retraction must exceed both this distance in face lengths (~1 cm) and this many standard deviations
    /// of the neutral D_f noise to count as a deliberate best-upright hold.
    public static let minUprightMagnitude = 0.04
    public static let minUprightSigmas = 4.0

    /// qShape = 1 - min(1, residual * penalty), with residual in units of face RMS spread.
    public static let shapeResidualPenalty = 5.0
    /// qConsistency = 1 - min(0.9, |D_f - D_b| * penalty), with D in face lengths.
    public static let consistencyPenalty = 3.0
    /// Minimum mean head-to-anchor offset change (in face RMS spreads) for a usable body forward axis.
    public static let minBodyForwardMagnitude = 0.1

    /// Backward hyperextension: ΔPitch or ΔJaw at or above this angle, or ΔCVA at or above `overextensionCvaDegrees`.
    public static let overextensionAngleDegrees: Float = 12.0
    public static let overextensionCvaDegrees = 8.0
    public static let overextensionHoldSeconds = 3.0

    /// Longest gap between trusted frames counted as continuous tracking time for rehab adherence.
    public static let maxAdherenceGapSeconds = 5.0

    private struct ForwardSample {
        let q: CameraDisplacement3D
        let bodyOffset: SIMD2<Double>?
    }

    private let lock = NSLock()

    private var _currentPhase: CalibrationPhase = .idle
    private var _baseline = PostureCalibrationBaseline()
    private var _profile = PostureProfile.default
    private var pending = PostureCalibrationBaseline()
    private var phaseStartedAt: TimeInterval?

    private var neutralCentroids: [CGPoint] = []
    private var neutralSpreads: [CGFloat] = []
    private var neutralFaceLengths: [CGFloat] = []
    private var neutralShapes: [[CGPoint]] = []
    private var neutralBodyOffsets: [SIMD2<Double>] = []
    private var neutralPitches: [Float] = []
    private var neutralJawAngles: [Float] = []
    private var forwardSettleCount = 0
    private var forwardSamples: [ForwardSample] = []
    private var forwardExcursion = 0.0
    private var uprightSettleCount = 0
    private var uprightSamples: [Double] = []
    private var validationCount = 0
    private var _rehabTargetFraction = 0.0

    // Rehab adherence accumulated since the last `drainAdherence()`
    private var adherence = PostureAdherence()
    private var lastTrustedTimestamp: TimeInterval?

    private var healthMonitor = CalibrationHealthMonitor()

    // Temporal EMA filter on displacement D
    private var emaDisplacement: Double = 0.0
    private var isEmaInitialized: Bool = false
    private let emaAlpha: Double = 0.25 // Smooth posture transitions without lag

    private var hysteresis = ForwardHeadAlertHysteresis()
    private var overextension = SustainedCondition(holdSeconds: overextensionHoldSeconds)
    private var isOverextensionSustained = false
    private var lastResult = RelativeCVAResult()

    public init() {}

    public var currentPhase: CalibrationPhase {
        lock.withLock { _currentPhase }
    }

    /// The committed calibration. Uncalibrated while a calibration is in progress.
    public var baseline: PostureCalibrationBaseline {
        lock.withLock { _baseline }
    }

    /// The user's posture profile: assumed neutral CVA and alert thresholds. Independent of calibration.
    public var profile: PostureProfile {
        lock.withLock { _profile }
    }

    /// Switches the posture profile. The learned calibration is kept; alert state restarts under the new thresholds.
    public func setProfile(_ profile: PostureProfile) {
        lock.withLock {
            guard profile != _profile else { return }
            _profile = profile
            hysteresis = ForwardHeadAlertHysteresis(thresholds: profile.alertThresholds)
            lastResult = RelativeCVAResult(relativeCVA: profile.baseCvaDegrees, profile: profile)
        }
    }

    /// Fraction of the calibrated correction range the rehab target asks for (0 = habitual neutral, 1 = best upright).
    public var rehabTargetFraction: Double {
        lock.withLock { _rehabTargetFraction }
    }

    /// Moves the alert target along the correction range. Alert state restarts, since ΔCVA is measured against the target.
    public func setRehabTargetFraction(_ fraction: Double) {
        let clamped = min(1.0, max(0.0, fraction))
        lock.withLock {
            guard clamped != _rehabTargetFraction else { return }
            _rehabTargetFraction = clamped
            hysteresis = ForwardHeadAlertHysteresis(thresholds: _profile.alertThresholds)
            overextension = SustainedCondition(holdSeconds: Self.overextensionHoldSeconds)
            isOverextensionSustained = false
        }
    }

    /// Returns and clears the tracking time accumulated by `evaluateFrame` since the previous call.
    public func drainAdherence() -> PostureAdherence {
        lock.withLock {
            let drained = adherence
            adherence = PostureAdherence()
            return drained
        }
    }

    /// Whether the committed calibration still matches the setup (see `CalibrationHealthMonitor`).
    public var calibrationHealth: CalibrationHealth {
        lock.withLock { healthMonitor.health }
    }

    /// The user chose to keep the current calibration: restart staleness detection from scratch.
    public func dismissStaleCalibration() {
        lock.withLock { healthMonitor.reset() }
    }

    /// Restarts the alert state machines and smoothing, e.g. after a guided exercise whose deliberate movements
    /// must not leave alert timers behind. The calibration is kept.
    public func resetAlertTracking() {
        lock.withLock { resetTracking() }
    }

    public var currentWarningLevel: Int {
        lock.withLock { hysteresis.level }
    }

    /// True while calibration frames should be fed via `recordCalibrationFrame`.
    public var isCollecting: Bool {
        lock.withLock { isCollectingPhase }
    }

    /// Starts interactive 4-phase calibration, discarding the current calibration.
    public func startCalibration() {
        lock.withLock {
            _baseline = PostureCalibrationBaseline()
            healthMonitor.reset()
            enter(.collectingNeutral, at: nil)
            clearSamples()
            resetTracking()
        }
    }

    /// Resets baseline and calibration state.
    public func resetCalibration() {
        lock.withLock {
            _baseline = PostureCalibrationBaseline()
            healthMonitor.reset()
            enter(.idle, at: nil)
            clearSamples()
            resetTracking()
        }
    }

    /// Feeds one frame during calibration.
    /// - Parameters:
    ///   - constellation: Rigid face constellation, or nil when no face was detected (still advances the phase timeout).
    ///   - bodyAnchor: Vision-normalized body anchor (Vision neck joint, i.e. shoulder midpoint), if reliably tracked.
    public func recordCalibrationFrame(
        constellation: FacialConstellation?,
        bodyAnchor: CGPoint? = nil,
        pitch: Float = 0.0,
        jawAngle: Float = 0.0,
        timestamp: TimeInterval = CACurrentMediaTime()
    ) {
        lock.withLock {
            guard isCollectingPhase else { return }

            let startedAt = phaseStartedAt ?? timestamp
            phaseStartedAt = startedAt
            if _currentPhase == .collectingUpright {
                guard timestamp - startedAt <= Self.uprightPhaseTimeoutSeconds else {
                    // Optional phase: continue without a correction range
                    pending.bestUprightDisplacement = 0.0
                    enter(.validating, at: timestamp)
                    return
                }
            } else {
                guard timestamp - startedAt <= Self.phaseTimeoutSeconds else {
                    fail()
                    return
                }
            }
            guard let constellation else { return }

            // All samples of one calibration must share the same image geometry
            if _currentPhase != .collectingNeutral || !neutralCentroids.isEmpty {
                guard aspectMatches(constellation, pending) else {
                    fail()
                    return
                }
            }

            let bodyOffset = bodyAnchor.map {
                BodyAnchorCalibration.offset(
                    centroid: constellation.centroid,
                    anchor: constellation.isotropic($0),
                    rmsSpread: constellation.rmsSpread
                )
            }

            switch _currentPhase {
            case .collectingNeutral:
                if neutralCentroids.isEmpty { pending.imageAspect0 = constellation.imageAspect }
                neutralCentroids.append(constellation.centroid)
                neutralSpreads.append(constellation.rmsSpread)
                if let faceLength = constellation.faceLength { neutralFaceLengths.append(faceLength) }
                neutralShapes.append(constellation.normalizedShape)
                if let bodyOffset { neutralBodyOffsets.append(bodyOffset) }
                neutralPitches.append(pitch)
                neutralJawAngles.append(jawAngle)
                if neutralCentroids.count >= Self.neutralSampleCount {
                    finalizeNeutralBaseline()
                    enter(.collectingForward, at: timestamp)
                }

            case .collectingForward:
                let q = displacement(of: constellation, against: pending)
                guard isForwardExcursion(q) else {
                    // Not (or no longer) holding forward: wait, and restart the hold when it resumes
                    forwardSettleCount = 0
                    forwardSamples.removeAll()
                    return
                }
                guard forwardSettleCount >= Self.forwardSettleFrameCount else {
                    forwardSettleCount += 1
                    return
                }
                forwardSamples.append(ForwardSample(q: q, bodyOffset: bodyOffset))
                if forwardSamples.count >= Self.forwardSampleCount {
                    if finalizeForwardVector() {
                        enter(.collectingUpright, at: timestamp)
                    } else {
                        fail()
                    }
                }

            case .collectingUpright:
                let dFace = forwardDisplacement(of: displacement(of: constellation, against: pending), using: pending)
                guard dFace <= -uprightThreshold() else {
                    // Not (or no longer) retracted: wait, and restart the hold when it resumes
                    uprightSettleCount = 0
                    uprightSamples.removeAll()
                    return
                }
                guard uprightSettleCount >= Self.uprightSettleFrameCount else {
                    uprightSettleCount += 1
                    return
                }
                uprightSamples.append(dFace)
                if uprightSamples.count >= Self.uprightSampleCount {
                    pending.bestUprightDisplacement = uprightSamples.reduce(0.0, +) / Double(uprightSamples.count)
                    enter(.validating, at: timestamp)
                }

            case .validating:
                let dFace = forwardDisplacement(of: displacement(of: constellation, against: pending), using: pending)
                let tolerance = Self.validationToleranceFraction * forwardExcursion
                validationCount = abs(dFace) <= tolerance ? validationCount + 1 : 0
                if validationCount >= Self.validationFrameCount {
                    pending.isCalibrated = true
                    _baseline = pending
                    healthMonitor.reset()
                    enter(.calibrated, at: timestamp)
                    clearSamples()
                    resetTracking()
                }

            case .idle, .calibrated, .failed:
                break
            }
        }
    }

    /// Camera-normalized displacement q_t of a constellation relative to the committed baseline, in face lengths.
    public func displacement(of constellation: FacialConstellation) -> CameraDisplacement3D {
        lock.withLock { displacement(of: constellation, against: _baseline) }
    }

    /// Evaluates current frame displacement and relative CVA with sensor fusion, temporal smoothing, and hysteresis.
    ///
    /// Pass `nil` when no face was detected: the previous result is returned with a failing quality gate and no
    /// filter or alert state is advanced, so dropouts never inject placeholder measurements.
    /// - Parameter bodyAnchor: Vision-normalized body anchor from the same frame as `constellation`.
    public func evaluateFrame(
        constellation: FacialConstellation?,
        bodyAnchor: CGPoint? = nil,
        bodyAnchorConfidence: Double = 1.0,
        pitch: Float = 0.0,
        jawAngle: Float = 0.0,
        timestamp: TimeInterval = CACurrentMediaTime()
    ) -> RelativeCVAResult {
        lock.withLock {
            guard _baseline.isCalibrated else {
                return RelativeCVAResult(relativeCVA: _profile.baseCvaDegrees, profile: _profile)
            }

            // No face, or a frame whose geometry is not comparable with the calibration (e.g. camera switched)
            if let constellation, !aspectMatches(constellation, _baseline) {
                healthMonitor.recordGeometryMismatch(timestamp: timestamp)
            }
            guard let constellation, aspectMatches(constellation, _baseline) else {
                var held = lastResult
                held.quality = PostureQualityGate(qFace: 0.0)
                return held
            }

            // 1. Compute 3D camera-normalized displacement q_t
            let q = displacement(of: constellation, against: _baseline)

            // 2. Estimator 1: Face-to-camera weighted projection D_f
            let dFace = forwardDisplacement(of: q, using: _baseline)

            // 3. Estimator 2: Head-to-body relative displacement D_b. Without a calibrated anchor we run face-only.
            let dBody: Double
            let qAnchor: Double
            if let anchor = bodyAnchor, let bodyCalibration = _baseline.bodyAnchor {
                dBody = bodyCalibration.displacement(
                    centroid: constellation.centroid,
                    anchor: constellation.isotropic(anchor),
                    rmsSpread: constellation.rmsSpread
                )
                qAnchor = max(0.0, min(1.0, bodyAnchorConfidence))
            } else {
                dBody = dFace
                qAnchor = 1.0
            }

            // 4. Quality Gate & Consistency Check
            let qFace = Double(constellation.captureQuality)
            let qShape = shapeQuality(of: constellation)

            // Consistency: if |D_f - D_b| is large (e.g. user leaned entire torso forward, whole body shifted),
            // qConsistency drops so the system does NOT trigger a false forward-head neck alarm!
            let diff = abs(dFace - dBody)
            let qConsistency = max(0.1, 1.0 - min(0.9, diff * Self.consistencyPenalty))

            let qualityGate = PostureQualityGate(
                qFace: qFace,
                qAnchor: qAnchor,
                qShape: qShape,
                qTemporal: 1.0,
                qConsistency: qConsistency
            )

            // Staleness: displacement off the forward axis cannot be forward head posture
            let orthogonalOffset = simd_length(q.asSIMD3 - dFace * _baseline.forwardVectorG)
            healthMonitor.recordFrame(orthogonalOffset: orthogonalOffset, isQualityPass: qualityGate.isPass, timestamp: timestamp)

            // Sensor Fusion: weight face vs body
            let wFace = qFace * 0.4
            let wBody = qAnchor * qConsistency * 0.6
            let totalW = max(0.001, wFace + wBody)
            let rawD = (wFace * dFace + wBody * dBody) / totalW

            // 5. Temporal Filtering on D (EMA filter before nonlinear atan2)
            if !isEmaInitialized {
                emaDisplacement = rawD
                isEmaInitialized = true
            } else {
                emaDisplacement = emaAlpha * rawD + (1.0 - emaAlpha) * emaDisplacement
            }

            // 6. Relative CVA Proxy: CVA_rel = atan2(kappa, (kappa / tan(CVA_0)) + D), with CVA_0 at the habitual neutral.
            // ΔCVA is measured against the rehab target, a point on the correction range between the habitual
            // neutral (D = 0) and the best upright posture (D < 0).
            let (cvaRel, _) = PostureMathEngine.computeRelativeCVA(
                forwardDisplacementD: emaDisplacement,
                baseCvaDegrees: _profile.baseCvaDegrees,
                kappa: _baseline.kappa
            )
            let targetD = targetDisplacement()
            let (targetCva, _) = PostureMathEngine.computeRelativeCVA(
                forwardDisplacementD: targetD,
                baseCvaDegrees: _profile.baseCvaDegrees,
                kappa: _baseline.kappa
            )
            let deltaCva = cvaRel - targetCva

            // 7. Alerting state machine: only trusted frames advance the sustained-alert timers
            let warningLevel = qualityGate.isPass
                ? hysteresis.update(deltaCva: deltaCva, timestamp: timestamp)
                : hysteresis.level

            let deltaPitch = pitch - _baseline.baselinePitch
            let deltaJawAngle = jawAngle - _baseline.baselineJawAngle
            let isOverextended = deltaPitch >= Self.overextensionAngleDegrees
                || deltaJawAngle >= Self.overextensionAngleDegrees
                || deltaCva >= Self.overextensionCvaDegrees
            if qualityGate.isPass {
                isOverextensionSustained = overextension.update(isOverextended, timestamp: timestamp)
                recordAdherence(isOnTarget: warningLevel == 0, timestamp: timestamp)
            }

            let result = RelativeCVAResult(
                relativeCVA: cvaRel,
                deltaCVA: deltaCva,
                forwardDisplacementD: emaDisplacement,
                displacementFace: dFace,
                displacementBody: dBody,
                quality: qualityGate,
                isSustainedForwardHead: warningLevel > 0,
                warningLevel: warningLevel,
                deltaPitch: deltaPitch,
                deltaJawAngle: deltaJawAngle,
                isOverextended: isOverextended,
                isSustainedOverextension: isOverextensionSustained,
                profile: _profile,
                targetCVA: targetCva,
                targetDisplacementD: targetD
            )
            lastResult = result
            return result
        }
    }

    // MARK: - Private (callers must hold `lock`)

    private var isCollectingPhase: Bool {
        switch _currentPhase {
        case .collectingNeutral, .collectingForward, .collectingUpright, .validating: return true
        case .idle, .calibrated, .failed: return false
        }
    }

    private func enter(_ phase: CalibrationPhase, at timestamp: TimeInterval?) {
        _currentPhase = phase
        phaseStartedAt = timestamp
    }

    private func fail() {
        enter(.failed, at: nil)
        clearSamples()
    }

    private func clearSamples() {
        pending = PostureCalibrationBaseline()
        neutralCentroids.removeAll()
        neutralSpreads.removeAll()
        neutralFaceLengths.removeAll()
        neutralShapes.removeAll()
        neutralBodyOffsets.removeAll()
        neutralPitches.removeAll()
        neutralJawAngles.removeAll()
        forwardSettleCount = 0
        forwardSamples.removeAll()
        forwardExcursion = 0.0
        uprightSettleCount = 0
        uprightSamples.removeAll()
        validationCount = 0
    }

    private func resetTracking() {
        isEmaInitialized = false
        emaDisplacement = 0.0
        hysteresis = ForwardHeadAlertHysteresis(thresholds: _profile.alertThresholds)
        overextension = SustainedCondition(holdSeconds: Self.overextensionHoldSeconds)
        isOverextensionSustained = false
        lastTrustedTimestamp = nil
        lastResult = RelativeCVAResult(relativeCVA: _profile.baseCvaDegrees, profile: _profile)
    }

    /// D of the rehab target posture, in face lengths (0 without a calibrated correction range).
    private func targetDisplacement() -> Double {
        _rehabTargetFraction * _baseline.bestUprightDisplacement
    }

    /// Counts the time since the previous trusted frame (gaps longer than `maxAdherenceGapSeconds` are not tracked).
    private func recordAdherence(isOnTarget: Bool, timestamp: TimeInterval) {
        defer { lastTrustedTimestamp = timestamp }
        guard let last = lastTrustedTimestamp else { return }
        let dt = timestamp - last
        guard dt > 0, dt <= Self.maxAdherenceGapSeconds else { return }
        adherence.trackedSeconds += dt
        if isOnTarget { adherence.onTargetSeconds += dt }
    }

    /// Minimum backward D_f for the best-upright hold: an absolute floor and a multiple of the neutral noise of D_f,
    /// whose GLS variance is 1 / (g^T Sigma^-1 g).
    private func uprightThreshold() -> Double {
        let g = pending.forwardVectorG
        let information = simd_reduce_add(g * g / pending.noiseCovarianceDiagonal)
        let sigma = information > 0 ? 1.0 / sqrt(information) : 0.0
        return max(Self.minUprightMagnitude, Self.minUprightSigmas * sigma)
    }

    private func aspectMatches(_ constellation: FacialConstellation, _ baseline: PostureCalibrationBaseline) -> Bool {
        abs(constellation.imageAspect - baseline.imageAspect0) < 1e-3
    }

    /// q_t relative to a neutral baseline, with scale ratio rho_t computed against its rmsSpread_0.
    private func displacement(of constellation: FacialConstellation, against baseline: PostureCalibrationBaseline) -> CameraDisplacement3D {
        PostureMathEngine.computeCameraDisplacement3D(
            currentCentroid: constellation.centroid,
            scaleRho: constellation.rmsSpread / baseline.rmsSpread0,
            baselineCentroid: baseline.centroid0,
            baselineL0: baseline.faceLength0,
            principalPoint: constellation.principalPoint,
            focalLength: PostureMathEngine.nominalFocalLength
        )
    }

    private func forwardDisplacement(of q: CameraDisplacement3D, using baseline: PostureCalibrationBaseline) -> Double {
        PostureMathEngine.computeWeightedForwardDisplacement(
            q: q,
            forwardVectorG: baseline.forwardVectorG,
            noiseCovarianceDiag: baseline.noiseCovarianceDiagonal
        )
    }

    private func isForwardExcursion(_ q: CameraDisplacement3D) -> Bool {
        let v = q.asSIMD3
        let mahalanobis = sqrt(simd_reduce_add(v * v / pending.noiseCovarianceDiagonal))
        return mahalanobis >= Self.minForwardMahalanobis && q.magnitude >= Self.minForwardMagnitude
    }

    private func shapeQuality(of constellation: FacialConstellation) -> Double {
        // No baseline shape captured: shape gating unavailable, do not penalize.
        guard !_baseline.faceShape0.isEmpty else { return 1.0 }
        // Landmark set changed versus baseline: shapes are not comparable, treat as untrusted.
        guard let residual = PostureMathEngine.shapeResidual(constellation.normalizedShape, baseline: _baseline.faceShape0) else {
            return 0.0
        }
        return max(0.0, 1.0 - min(1.0, residual * Self.shapeResidualPenalty))
    }

    private func finalizeNeutralBaseline() {
        let count = CGFloat(neutralCentroids.count)
        guard count > 0 else { return }

        let avgX = neutralCentroids.reduce(0) { $0 + $1.x } / count
        let avgY = neutralCentroids.reduce(0) { $0 + $1.y } / count
        let avgSpread = neutralSpreads.reduce(0) { $0 + $1 } / count

        pending.centroid0 = CGPoint(x: avgX, y: avgY)
        pending.rmsSpread0 = max(0.01, avgSpread)
        // L_0 from the measured face height when mostly available, otherwise from the nominal spread ratio
        if neutralFaceLengths.count * 2 >= neutralCentroids.count {
            pending.faceLength0 = max(0.01, neutralFaceLengths.reduce(0, +) / CGFloat(neutralFaceLengths.count))
        } else {
            pending.faceLength0 = pending.rmsSpread0 / PostureMathEngine.nominalSpreadToFaceLength
        }
        pending.faceShape0 = PostureMathEngine.meanShape(neutralShapes) ?? []
        pending.baselinePitch = neutralPitches.reduce(0, +) / Float(neutralPitches.count)
        pending.baselineJawAngle = neutralJawAngles.reduce(0, +) / Float(neutralJawAngles.count)

        // Diagonal noise variance of q during the neutral hold (X/Y from centroid jitter, Z from scale jitter)
        let l0 = Double(pending.faceLength0)
        let spread0 = Double(pending.rmsSpread0)
        let f = PostureMathEngine.nominalFocalLength
        var varX = 0.0
        var varY = 0.0
        var varZ = 0.0
        for (c, s) in zip(neutralCentroids, neutralSpreads) {
            let dx = Double(c.x - avgX) / l0
            let dy = Double(c.y - avgY) / l0
            let dz = (f / l0) * (1.0 - spread0 / Double(max(0.01, s)))
            varX += dx * dx
            varY += dy * dy
            varZ += dz * dz
        }
        let n = Double(count)
        pending.noiseCovarianceDiagonal = SIMD3<Double>(
            max(0.0001, varX / n),
            max(0.0001, varY / n),
            max(0.0005, varZ / n)
        )
    }

    /// Learns g (and the body-anchor estimator) from the forward hold. Returns false if the samples do not
    /// define a usable forward direction.
    private func finalizeForwardVector() -> Bool {
        guard !forwardSamples.isEmpty else { return false }

        let n = Double(forwardSamples.count)
        let meanQ = forwardSamples.reduce(SIMD3<Double>.zero) { $0 + $1.q.asSIMD3 } / n
        let mag = simd_length(meanQ)
        // Every sample exceeded the threshold, so only an inconsistent hold can average below it
        guard mag >= Self.minForwardMagnitude else { return false }

        pending.forwardVectorG = meanQ / mag
        forwardExcursion = forwardSamples.reduce(0.0) { $0 + forwardDisplacement(of: $1.q, using: pending) } / n
        pending.bodyAnchor = calibrateBodyAnchor()
        return true
    }

    /// Learns D_b's forward axis and scale from samples where the body anchor was tracked in both phases.
    private func calibrateBodyAnchor() -> BodyAnchorCalibration? {
        let minNeutral = Self.neutralSampleCount / 2
        let minForward = Self.forwardSampleCount / 2
        guard neutralBodyOffsets.count >= minNeutral else { return nil }

        let offset0 = neutralBodyOffsets.reduce(SIMD2<Double>.zero, +) / Double(neutralBodyOffsets.count)
        let paired = forwardSamples.compactMap { sample in sample.bodyOffset.map { (q: sample.q, delta: $0 - offset0) } }
        guard paired.count >= minForward else { return nil }

        let meanDelta = paired.reduce(SIMD2<Double>.zero) { $0 + $1.delta } / Double(paired.count)
        let deltaMagnitude = simd_length(meanDelta)
        guard deltaMagnitude >= Self.minBodyForwardMagnitude else { return nil }

        let axis = meanDelta / deltaMagnitude
        let meanFace = paired.reduce(0.0) { $0 + forwardDisplacement(of: $1.q, using: pending) } / Double(paired.count)
        // mean(delta · axis) == |mean(delta)| by construction
        let scale = meanFace / deltaMagnitude
        guard scale.isFinite, scale > 0 else { return nil }

        return BodyAnchorCalibration(offset0: offset0, forwardAxis: axis, scale: scale)
    }
}
