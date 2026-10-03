import Foundation
import CoreGraphics
import Vision
import simd

/// Rigid facial constellation tracking providing robust centroid, RMS scale,
/// camera-normalized 3D displacement vector q_t, and shape residual.
public struct FacialConstellation: Equatable, Sendable {
    public let centroid: CGPoint
    public let rmsSpread: CGFloat
    public let scaleFactor: CGFloat // rho_t = rmsSpread / rmsSpread_0
    public let points: [CGPoint]
    public let shapeResidual: Double
    public let captureQuality: Float
    
    public init(
        centroid: CGPoint,
        rmsSpread: CGFloat,
        scaleFactor: CGFloat = 1.0,
        points: [CGPoint] = [],
        shapeResidual: Double = 0.0,
        captureQuality: Float = 1.0
    ) {
        self.centroid = centroid
        self.rmsSpread = rmsSpread
        self.scaleFactor = scaleFactor
        self.points = points
        self.shapeResidual = shapeResidual
        self.captureQuality = captureQuality
    }
}

/// Camera-normalized 3D displacement vector q_t = [X_t, Y_t, Z_t]^T
/// in units of reference face length L_0.
public struct CameraDisplacement3D: Equatable, Sendable {
    public var x: Double // Horizontal camera translation
    public var y: Double // Vertical camera translation (critical for chest-level webcams)
    public var z: Double // Depth-induced scale displacement
    
    public init(x: Double, y: Double, z: Double) {
        self.x = x
        self.y = y
        self.z = z
    }
    
    public var asSIMD3: SIMD3<Double> {
        SIMD3<Double>(x, y, z)
    }
    
    public var magnitude: Double {
        sqrt(x * x + y * y + z * z)
    }
}

/// Baseline calibration state capturing neutral pose, noise covariance, and forward direction vector g.
public struct PostureCalibrationBaseline: Equatable, Sendable {
    public var centroid0: CGPoint
    public var rmsSpread0: CGFloat
    public var facePoints0: [CGPoint]
    public var noiseCovarianceDiagonal: SIMD3<Double> // Var(X), Var(Y), Var(Z)
    public var forwardVectorG: SIMD3<Double>          // Unit direction of forward head movement in camera frame
    public var baseCvaDegrees: Double                // CVA_0 (e.g. 55.0° or 60.0°)
    public var kappa: Double                         // Anatomical ratio h / L_real (default ~0.83)
    public var baselinePitch: Float                  // Neutral head pitch in camera frame (degrees)
    public var baselineJawAngle: Float               // Neutral mandibular angle (degrees)
    public var isCalibrated: Bool
    
    public init(
        centroid0: CGPoint = CGPoint(x: 0.5, y: 0.5),
        rmsSpread0: CGFloat = 0.15,
        facePoints0: [CGPoint] = [],
        noiseCovarianceDiagonal: SIMD3<Double> = SIMD3<Double>(0.001, 0.001, 0.005),
        forwardVectorG: SIMD3<Double> = SIMD3<Double>(0.7071, 0.20, 0.675),
        baseCvaDegrees: Double = 55.0,
        kappa: Double = 0.83,
        baselinePitch: Float = 0.0,
        baselineJawAngle: Float = 0.0,
        isCalibrated: Bool = false
    ) {
        self.centroid0 = centroid0
        self.rmsSpread0 = rmsSpread0
        self.facePoints0 = facePoints0
        self.noiseCovarianceDiagonal = noiseCovarianceDiagonal
        self.forwardVectorG = forwardVectorG
        self.baseCvaDegrees = baseCvaDegrees
        self.kappa = kappa
        self.baselinePitch = baselinePitch
        self.baselineJawAngle = baselineJawAngle
        self.isCalibrated = isCalibrated
    }
}

/// Quality gate metrics for filtering out frames corrupted by yaw/pitch, expressions, or motion blur.
public struct PostureQualityGate: Equatable, Sendable {
    public var qFace: Double       // Capture quality [0, 1]
    public var qAnchor: Double     // Body anchor confidence [0, 1]
    public var qShape: Double      // 1.0 - normalized shape residual
    public var qTemporal: Double   // Temporal consistency
    public var qConsistency: Double // Agreement between Face and Body estimators
    
    public var compositeScore: Double {
        qFace * qAnchor * qShape * qTemporal * qConsistency
    }
    
    public var isPass: Bool {
        compositeScore >= 0.25 && qFace >= 0.20 && qShape >= 0.40
    }
    
    public init(
        qFace: Double = 1.0,
        qAnchor: Double = 1.0,
        qShape: Double = 1.0,
        qTemporal: Double = 1.0,
        qConsistency: Double = 1.0
    ) {
        self.qFace = qFace
        self.qAnchor = qAnchor
        self.qShape = qShape
        self.qTemporal = qTemporal
        self.qConsistency = qConsistency
    }
}

/// Relative CVA proxy evaluation results.
public struct RelativeCVAResult: Equatable, Sendable {
    public var relativeCVA: Double   // CVA_rel,t (degrees)
    public var deltaCVA: Double      // ΔCVA_t = CVA_rel,t - CVA_0 (degrees)
    public var forwardDisplacementD: Double // Normalized forward displacement D_t
    public var displacementFace: Double    // D_f
    public var displacementBody: Double    // D_b
    public var quality: PostureQualityGate
    public var isSustainedForwardHead: Bool
    public var warningLevel: Int     // 0: Normal, 1: Mild (-5° for 30s), 2: Severe (-10° for 20s)
    public var deltaPitch: Float     // Head pitch rotation relative to calibrated neutral baseline (degrees)
    public var deltaJawAngle: Float  // Mandibular jawline inclination relative to baseline (degrees)
    
    public init(
        relativeCVA: Double = 55.0,
        deltaCVA: Double = 0.0,
        forwardDisplacementD: Double = 0.0,
        displacementFace: Double = 0.0,
        displacementBody: Double = 0.0,
        quality: PostureQualityGate = PostureQualityGate(),
        isSustainedForwardHead: Bool = false,
        warningLevel: Int = 0,
        deltaPitch: Float = 0.0,
        deltaJawAngle: Float = 0.0
    ) {
        self.relativeCVA = relativeCVA
        self.deltaCVA = deltaCVA
        self.forwardDisplacementD = forwardDisplacementD
        self.displacementFace = displacementFace
        self.displacementBody = displacementBody
        self.quality = quality
        self.isSustainedForwardHead = isSustainedForwardHead
        self.warningLevel = warningLevel
        self.deltaPitch = deltaPitch
        self.deltaJawAngle = deltaJawAngle
    }
}

/// Advanced Posture Math Engine: Implements rigid facial constellation tracking,
/// camera-normalized 3D displacement q_t, forward-direction projection, and relative CVA.
public enum PostureMathEngine {
    
    // MARK: - 1. Rigid Landmark Constellation Extraction
    
    /// Extracts rigid facial points (faceContour, noseCrest, medianLine, eyes) while excluding
    /// non-rigid expressive regions (lips, mouth aperture, eyebrows).
    public static func extractRigidConstellation(from observation: VNFaceObservation?) -> [CGPoint] {
        guard let obs = observation, let landmarks = obs.landmarks else { return [] }
        let bbox = obs.boundingBox
        
        var points: [CGPoint] = []
        
        // Helper to convert normalized landmark point to image space [0, 1]
        func convertPoint(_ pt: CGPoint) -> CGPoint {
            CGPoint(x: bbox.origin.x + pt.x * bbox.width, y: bbox.origin.y + pt.y * bbox.height)
        }
        
        // 1. Face contour (upper jawline/temple and chin apex)
        if let contour = landmarks.faceContour, contour.pointCount > 0 {
            let pts = contour.normalizedPoints
            for p in pts {
                points.append(convertPoint(p))
            }
        }
        
        // 2. Nose Crest (rigid nasal bridge)
        if let crest = landmarks.noseCrest, crest.pointCount > 0 {
            for p in crest.normalizedPoints {
                points.append(convertPoint(p))
            }
        }
        
        // 3. Median Line (sagittal facial midline)
        if let median = landmarks.medianLine, median.pointCount > 0 {
            for p in median.normalizedPoints {
                points.append(convertPoint(p))
            }
        }
        
        // 4. Stable Eye centers
        if let leftEye = landmarks.leftEye, leftEye.pointCount > 0 {
            let pts = leftEye.normalizedPoints.map { convertPoint($0) }
            let avgX = pts.reduce(0) { $0 + $1.x } / CGFloat(pts.count)
            let avgY = pts.reduce(0) { $0 + $1.y } / CGFloat(pts.count)
            points.append(CGPoint(x: avgX, y: avgY))
        }
        if let rightEye = landmarks.rightEye, rightEye.pointCount > 0 {
            let pts = rightEye.normalizedPoints.map { convertPoint($0) }
            let avgX = pts.reduce(0) { $0 + $1.x } / CGFloat(pts.count)
            let avgY = pts.reduce(0) { $0 + $1.y } / CGFloat(pts.count)
            points.append(CGPoint(x: avgX, y: avgY))
        }
        
        return points
    }
    
    /// Computes centroid and RMS spread of a rigid landmark constellation.
    public static func computeConstellation(
        points: [CGPoint],
        baselineSpread: CGFloat? = nil,
        baselinePoints: [CGPoint]? = nil,
        captureQuality: Float = 1.0
    ) -> FacialConstellation {
        guard !points.isEmpty else {
            return FacialConstellation(centroid: CGPoint(x: 0.5, y: 0.5), rmsSpread: 0.15)
        }
        
        let count = CGFloat(points.count)
        let sumX = points.reduce(0) { $0 + $1.x }
        let sumY = points.reduce(0) { $0 + $1.y }
        let centroid = CGPoint(x: sumX / count, y: sumY / count)
        
        // RMS spread from centroid: sqrt( sum ||p_i - bar{p}||^2 / N )
        var sumDistSq: CGFloat = 0.0
        for p in points {
            let dx = p.x - centroid.x
            let dy = p.y - centroid.y
            sumDistSq += (dx * dx + dy * dy)
        }
        let rmsSpread = max(0.01, sqrt(sumDistSq / count))
        
        // Relative scale factor rho_t = rmsSpread / rmsSpread_0
        let base = baselineSpread ?? rmsSpread
        let scaleFactor = max(0.1, rmsSpread / base)
        
        // Compute shape residual if baseline constellation is available
        var shapeResidual: Double = 0.0
        if let basePts = baselinePoints, basePts.count == points.count, basePts.count > 0 {
            var diffSq: Double = 0.0
            for i in 0..<points.count {
                let dx = Double(points[i].x - centroid.x) - Double(basePts[i].x)
                let dy = Double(points[i].y - centroid.y) - Double(basePts[i].y)
                diffSq += (dx * dx + dy * dy)
            }
            shapeResidual = sqrt(diffSq / Double(points.count))
        }
        
        return FacialConstellation(
            centroid: centroid,
            rmsSpread: rmsSpread,
            scaleFactor: scaleFactor,
            points: points,
            shapeResidual: shapeResidual,
            captureQuality: captureQuality
        )
    }
    
    // MARK: - 2. Camera-Normalized 3D Displacement (q_t)
    
    /// Computes camera-normalized 3D displacement vector q_t = [X_t, Y_t, Z_t]^T
    /// incorporating vertical elevation Y_t and perspective scale compensation r_t.
    public static func computeCameraDisplacement3D(
        currentCentroid: CGPoint,
        scaleRho: CGFloat,
        baselineCentroid: CGPoint,
        baselineL0: CGFloat,
        principalPoint: CGPoint = CGPoint(x: 0.5, y: 0.5),
        focalLengthPixels: Double = 1.2
    ) -> CameraDisplacement3D {
        let rT = 1.0 / Double(max(0.2, scaleRho))
        let L0 = Double(max(0.01, baselineL0))
        
        let cx = Double(principalPoint.x)
        let cy = Double(principalPoint.y)
        
        let ut = Double(currentCentroid.x)
        let vt = Double(currentCentroid.y)
        let u0 = Double(baselineCentroid.x)
        let v0 = Double(baselineCentroid.y)
        
        // X_t = [ r_t * (u_t - c_x) - (u_0 - c_x) ] / L_0
        let x = (rT * (ut - cx) - (u0 - cx)) / L0
        
        // Y_t = [ r_t * (v_t - c_y) - (v_0 - c_y) ] / L_0 (Vertical shift from low elevation camera)
        let y = (rT * (vt - cy) - (v0 - cy)) / L0
        
        // Z_t = (f_x / L_0) * (1.0 - r_t) (Depth movement toward camera)
        let z = (focalLengthPixels / L0) * (1.0 - rT)
        
        return CameraDisplacement3D(x: x, y: y, z: z)
    }
    
    // MARK: - 3. Forward Direction Projection (D_f)
    
    /// Computes generalized least-squares weighted projection D_f = (g^T Sigma^-1 q) / (g^T Sigma^-1 g)
    public static func computeWeightedForwardDisplacement(
        q: CameraDisplacement3D,
        forwardVectorG: SIMD3<Double>,
        noiseCovarianceDiag: SIMD3<Double>
    ) -> Double {
        // Inverse covariance diagonal weights
        let invVarX = 1.0 / max(0.00001, noiseCovarianceDiag.x)
        let invVarY = 1.0 / max(0.00001, noiseCovarianceDiag.y)
        let invVarZ = 1.0 / max(0.00001, noiseCovarianceDiag.z)
        
        let qVec = q.asSIMD3
        let g = forwardVectorG
        
        // g^T Sigma^-1 q
        let numerator = g.x * invVarX * qVec.x + g.y * invVarY * qVec.y + g.z * invVarZ * qVec.z
        
        // g^T Sigma^-1 g
        let denominator = g.x * invVarX * g.x + g.y * invVarY * g.y + g.z * invVarZ * g.z
        
        guard abs(denominator) > 0.00001 else {
            return simd_dot(g, qVec)
        }
        
        return numerator / denominator
    }
    
    // MARK: - 4. Relative CVA Proxy & Warning Evaluation
    
    /// Computes relative CVA from forward displacement D:
    /// CVA_rel = atan2(kappa, (kappa / tan(CVA_0)) + D)
    public static func computeRelativeCVA(
        forwardDisplacementD: Double,
        baseCvaDegrees: Double = 55.0,
        kappa: Double = 0.83
    ) -> (cvaRel: Double, deltaCva: Double) {
        let cva0Rad = baseCvaDegrees * .pi / 180.0
        let tanCva0 = max(0.01, tan(cva0Rad))
        
        let x0 = kappa / tanCva0
        let denominator = x0 + forwardDisplacementD
        
        let cvaRelRad = atan2(kappa, max(0.01, denominator))
        let cvaRelDegrees = cvaRelRad * 180.0 / .pi
        let deltaCva = cvaRelDegrees - baseCvaDegrees
        
        return (cvaRel: cvaRelDegrees, deltaCva: deltaCva)
    }
}
