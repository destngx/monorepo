import Foundation
import CoreGraphics
import Vision
import QuartzCore
import simd

/// Calibration state and engine for learning personalized forward-direction vector g
/// and noise covariance Sigma across 3-phase user movements.
public final class CalibrationEngine: @unchecked Sendable {
    public static let shared = CalibrationEngine()
    
    public enum CalibrationPhase: String, Sendable {
        case idle = "Idle"
        case collectingNeutral = "1/3: Sit Neutral (Keep head straight)"
        case collectingForward = "2/3: Move Head Forward (Hold 2s)"
        case validating = "3/3: Return to Neutral"
        case calibrated = "Calibrated"
    }
    
    public private(set) var currentPhase: CalibrationPhase = .idle
    public private(set) var baseline: PostureCalibrationBaseline = PostureCalibrationBaseline()
    
    private var neutralSamples: [CameraDisplacement3D] = []
    private var forwardSamples: [CameraDisplacement3D] = []
    private var neutralCentroids: [CGPoint] = []
    private var neutralSpreads: [CGFloat] = []
    private var neutralPitches: [Float] = []
    private var neutralJawAngles: [Float] = []
    
    // Temporal EMA filter on displacement D
    private var emaDisplacement: Double = 0.0
    private var isEmaInitialized: Bool = false
    private let emaAlpha: Double = 0.25 // Smooth posture transitions without lag
    
    // Hysteresis alert state tracking
    private var sustainedWarningStartTime: TimeInterval? = nil
    private var sustainedRecoveryStartTime: TimeInterval? = nil
    public private(set) var currentWarningLevel: Int = 0 // 0: Normal, 1: Mild, 2: Severe
    
    public init() {}
    
    /// Starts interactive 3-phase calibration.
    public func startCalibration() {
        currentPhase = .collectingNeutral
        neutralSamples.removeAll()
        forwardSamples.removeAll()
        neutralCentroids.removeAll()
        neutralSpreads.removeAll()
        neutralPitches.removeAll()
        neutralJawAngles.removeAll()
        isEmaInitialized = false
        sustainedWarningStartTime = nil
        sustainedRecoveryStartTime = nil
        currentWarningLevel = 0
    }
    
    /// Feeds live constellation during calibration to learn neutral baseline and forward vector g.
    public func recordCalibrationFrame(
        constellation: FacialConstellation,
        pitch: Float = 0.0,
        jawAngle: Float = 0.0
    ) {
        switch currentPhase {
        case .collectingNeutral:
            neutralCentroids.append(constellation.centroid)
            neutralSpreads.append(constellation.rmsSpread)
            neutralPitches.append(pitch)
            neutralJawAngles.append(jawAngle)
            if neutralCentroids.count >= 20 { // ~0.7s at 30fps
                finalizeNeutralBaseline(constellation: constellation)
                currentPhase = .collectingForward
            }
            
        case .collectingForward:
            let q = PostureMathEngine.computeCameraDisplacement3D(
                currentCentroid: constellation.centroid,
                scaleRho: constellation.scaleFactor,
                baselineCentroid: baseline.centroid0,
                baselineL0: baseline.rmsSpread0
            )
            forwardSamples.append(q)
            if forwardSamples.count >= 25 { // ~0.8s
                finalizeForwardVector()
                currentPhase = .calibrated
            }
            
        case .idle, .validating, .calibrated:
            break
        }
    }
    
    private func finalizeNeutralBaseline(constellation: FacialConstellation) {
        let count = CGFloat(neutralCentroids.count)
        guard count > 0 else { return }
        
        let avgX = neutralCentroids.reduce(0) { $0 + $1.x } / count
        let avgY = neutralCentroids.reduce(0) { $0 + $1.y } / count
        let avgSpread = neutralSpreads.reduce(0) { $0 + $1 } / count
        
        baseline.centroid0 = CGPoint(x: avgX, y: avgY)
        baseline.rmsSpread0 = max(0.01, avgSpread)
        baseline.facePoints0 = constellation.points
        
        if !neutralPitches.isEmpty {
            baseline.baselinePitch = neutralPitches.reduce(0, +) / Float(neutralPitches.count)
        }
        if !neutralJawAngles.isEmpty {
            baseline.baselineJawAngle = neutralJawAngles.reduce(0, +) / Float(neutralJawAngles.count)
        }
        
        // Compute diagonal noise variance during neutral hold
        var varX: Double = 0.0005
        var varY: Double = 0.0005
        let varZ: Double = 0.002
        
        for c in neutralCentroids {
            let dx = Double(c.x - avgX) / Double(baseline.rmsSpread0)
            let dy = Double(c.y - avgY) / Double(baseline.rmsSpread0)
            varX += dx * dx
            varY += dy * dy
        }
        varX /= Double(count)
        varY /= Double(count)
        
        baseline.noiseCovarianceDiagonal = SIMD3<Double>(max(0.0001, varX), max(0.0001, varY), max(0.0005, varZ))
    }
    
    private func finalizeForwardVector() {
        guard !forwardSamples.isEmpty else { return }
        
        // Robust mean of forward displacements
        let avgX = forwardSamples.map(\.x).reduce(0, +) / Double(forwardSamples.count)
        let avgY = forwardSamples.map(\.y).reduce(0, +) / Double(forwardSamples.count)
        let avgZ = forwardSamples.map(\.z).reduce(0, +) / Double(forwardSamples.count)
        
        let rawVec = SIMD3<Double>(avgX, avgY, avgZ)
        let mag = simd_length(rawVec)
        
        if mag > 0.01 {
            baseline.forwardVectorG = rawVec / mag
            baseline.isCalibrated = true
        } else {
            // Fallback nominal forward vector for low diagonal webcam
            baseline.forwardVectorG = simd_normalize(SIMD3<Double>(0.7071, 0.20, 0.675))
            baseline.isCalibrated = true
        }
    }
    
    /// Evaluates current frame displacement and relative CVA with sensor fusion, temporal smoothing, and hysteresis.
    public func evaluateFrame(
        constellation: FacialConstellation,
        bodyAnchorDisplacement: Double? = nil,
        bodyAnchorConfidence: Double = 1.0,
        pitch: Float = 0.0,
        jawAngle: Float = 0.0,
        timestamp: TimeInterval = CACurrentMediaTime()
    ) -> RelativeCVAResult {
        guard baseline.isCalibrated else {
            return RelativeCVAResult(
                relativeCVA: baseline.baseCvaDegrees,
                deltaCVA: 0.0,
                forwardDisplacementD: 0.0,
                warningLevel: 0,
                deltaPitch: 0.0,
                deltaJawAngle: 0.0
            )
        }
        
        // 1. Compute 3D camera-normalized displacement q_t
        let q = PostureMathEngine.computeCameraDisplacement3D(
            currentCentroid: constellation.centroid,
            scaleRho: constellation.scaleFactor,
            baselineCentroid: baseline.centroid0,
            baselineL0: baseline.rmsSpread0
        )
        
        // 2. Estimator 1: Face-to-camera weighted projection D_f
        let dFace = PostureMathEngine.computeWeightedForwardDisplacement(
            q: q,
            forwardVectorG: baseline.forwardVectorG,
            noiseCovarianceDiag: baseline.noiseCovarianceDiagonal
        )
        
        // 3. Estimator 2: Head-to-body relative displacement D_b (if body anchor available)
        let dBody = bodyAnchorDisplacement ?? dFace
        
        // 4. Quality Gate & Consistency Check
        let qFace = Double(constellation.captureQuality)
        let qAnchor = bodyAnchorConfidence
        let qShape = max(0.0, 1.0 - min(1.0, constellation.shapeResidual * 5.0))
        
        // Consistency: if |D_f - D_b| is large (e.g. user leaned entire torso forward, whole body shifted),
        // qConsistency drops so the system does NOT trigger a false forward-head neck alarm!
        let diff = abs(dFace - dBody)
        let qConsistency = max(0.1, 1.0 - min(0.9, diff * 1.5))
        
        let qualityGate = PostureQualityGate(
            qFace: qFace,
            qAnchor: qAnchor,
            qShape: qShape,
            qTemporal: 1.0,
            qConsistency: qConsistency
        )
        
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
        
        // 6. Relative CVA Proxy: CVA_rel = atan2(kappa, (kappa / tan(CVA_0)) + D)
        let (cvaRel, deltaCva) = PostureMathEngine.computeRelativeCVA(
            forwardDisplacementD: emaDisplacement,
            baseCvaDegrees: baseline.baseCvaDegrees,
            kappa: baseline.kappa
        )
        
        // 7. Alerting State Machine with Hysteresis:
        // Level 1: ΔCVA <= -5° for 30s
        // Level 2: ΔCVA <= -10° for 20s
        // Recovery: only clear when ΔCVA > -2° for 10s
        let warningLevel = updateHysteresis(
            deltaCva: deltaCva,
            qualityPass: qualityGate.isPass,
            timestamp: timestamp
        )
        
        let dPitch = pitch - baseline.baselinePitch
        let dJaw = jawAngle - baseline.baselineJawAngle
        
        return RelativeCVAResult(
            relativeCVA: cvaRel,
            deltaCVA: deltaCva,
            forwardDisplacementD: emaDisplacement,
            displacementFace: dFace,
            displacementBody: dBody,
            quality: qualityGate,
            isSustainedForwardHead: warningLevel > 0,
            warningLevel: warningLevel,
            deltaPitch: dPitch,
            deltaJawAngle: dJaw
        )
    }
    
    private func updateHysteresis(
        deltaCva: Double,
        qualityPass: Bool,
        timestamp: TimeInterval
    ) -> Int {
        guard qualityPass else { return currentWarningLevel }
        
        if deltaCva <= -15.0 {
            // Acute severe posture breakdown (e.g. Δ-15° to Δ-25° turtle neck):
            // Fast confirmation (1.5s) to avoid lag when user is visibly hunched
            if let start = sustainedWarningStartTime {
                if timestamp - start >= 1.5 {
                    currentWarningLevel = 2
                }
            } else {
                sustainedWarningStartTime = timestamp
            }
            sustainedRecoveryStartTime = nil
        } else if deltaCva <= -10.0 {
            // Substantial forward head drift (Δ-10°): trigger Level 2 within 4.0s
            if let start = sustainedWarningStartTime {
                if timestamp - start >= 4.0 {
                    currentWarningLevel = 2
                }
            } else {
                sustainedWarningStartTime = timestamp
            }
            sustainedRecoveryStartTime = nil
        } else if deltaCva <= -5.0 {
            // Mild posture drift (Δ-5°): trigger Level 1 reminder within 3.0s
            if let start = sustainedWarningStartTime {
                if timestamp - start >= 3.0 {
                    currentWarningLevel = max(1, currentWarningLevel)
                }
            } else {
                sustainedWarningStartTime = timestamp
            }
            sustainedRecoveryStartTime = nil
        } else if deltaCva > -2.5 {
            // Fast recovery when user sits upright
            if let recStart = sustainedRecoveryStartTime {
                if timestamp - recStart >= 3.0 {
                    currentWarningLevel = 0
                    sustainedWarningStartTime = nil
                }
            } else {
                sustainedRecoveryStartTime = timestamp
            }
        } else {
            // Hysteresis buffer zone (-5.0° to -2.5°): maintain current status
            sustainedRecoveryStartTime = nil
        }
        
        return currentWarningLevel
    }
    
    /// Resets baseline and calibration state.
    public func resetCalibration() {
        baseline = PostureCalibrationBaseline()
        currentPhase = .idle
        neutralPitches.removeAll()
        neutralJawAngles.removeAll()
        isEmaInitialized = false
        sustainedWarningStartTime = nil
        sustainedRecoveryStartTime = nil
        currentWarningLevel = 0
    }
}
