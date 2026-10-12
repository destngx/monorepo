import Foundation
import CoreGraphics
import Vision
import simd

/// Rigid facial constellation for a single frame: centroid, RMS scale and the image-space points.
/// Baseline-relative quantities (scale ratio rho_t, shape residual) are derived by `CalibrationEngine`
/// against its own calibrated baseline so they are always consistent with the calibration state.
///
/// All coordinates are isotropic and width-normalized: x in [0, 1], y in [0, imageAspect]. Vision's normalized
/// coordinates scale x by image width and y by image height, which would distort distances on non-square frames.
public struct FacialConstellation: Equatable, Sendable {
    public let centroid: CGPoint
    public let rmsSpread: CGFloat
    public let points: [CGPoint]
    public let captureQuality: Float
    /// Face length (Vision face bounding-box height) in isotropic units, if measured.
    public let faceLength: CGFloat?
    /// Image height / width of the frame the constellation was measured in.
    public let imageAspect: CGFloat
    
    public init(
        centroid: CGPoint,
        rmsSpread: CGFloat,
        points: [CGPoint] = [],
        captureQuality: Float = 1.0,
        faceLength: CGFloat? = nil,
        imageAspect: CGFloat = 1.0
    ) {
        self.centroid = centroid
        self.rmsSpread = rmsSpread
        self.points = points
        self.captureQuality = captureQuality
        self.faceLength = faceLength
        self.imageAspect = imageAspect
    }
    
    /// Optical center of the frame in isotropic units.
    public var principalPoint: CGPoint {
        CGPoint(x: 0.5, y: 0.5 * imageAspect)
    }
    
    /// Converts a Vision-normalized point from the same frame into this constellation's isotropic coordinates.
    public func isotropic(_ visionPoint: CGPoint) -> CGPoint {
        PostureMathEngine.isotropicPoint(visionPoint, imageAspect: imageAspect)
    }
    
    /// Translation- and scale-invariant shape: (p_i - centroid) / rmsSpread.
    public var normalizedShape: [CGPoint] {
        guard rmsSpread > 0 else { return [] }
        return points.map {
            CGPoint(x: ($0.x - centroid.x) / rmsSpread, y: ($0.y - centroid.y) / rmsSpread)
        }
    }
}

/// Camera-normalized 3D displacement vector q_t = [X_t, Y_t, Z_t]^T
/// in units of the real face length (image face length L_0 at the neutral distance).
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

/// Calibrated head-to-body estimator D_b.
///
/// The head-to-anchor offset is measured in units of the current face RMS spread, which makes it invariant to
/// the whole upper body moving toward or away from the camera. During the forward calibration phase we learn
/// the image-plane direction of head-only forward motion (`forwardAxis`) and a `scale` that maps the projection
/// onto that axis into the same units as the face estimator D_f, so the two can be compared and fused.
public struct BodyAnchorCalibration: Equatable, Sendable {
    public var offset0: SIMD2<Double>      // Neutral (centroid - anchor) / rmsSpread
    public var forwardAxis: SIMD2<Double>  // Unit direction of head-only forward motion in offset space
    public var scale: Double               // Maps offset projection into D_f units
    
    public init(offset0: SIMD2<Double>, forwardAxis: SIMD2<Double>, scale: Double) {
        self.offset0 = offset0
        self.forwardAxis = forwardAxis
        self.scale = scale
    }
    
    public static func offset(centroid: CGPoint, anchor: CGPoint, rmsSpread: CGFloat) -> SIMD2<Double> {
        let spread = Double(max(0.01, rmsSpread))
        return SIMD2<Double>(Double(centroid.x - anchor.x) / spread, Double(centroid.y - anchor.y) / spread)
    }
    
    /// Head-to-body forward displacement D_b in D_f units.
    public func displacement(centroid: CGPoint, anchor: CGPoint, rmsSpread: CGFloat) -> Double {
        let delta = Self.offset(centroid: centroid, anchor: anchor, rmsSpread: rmsSpread) - offset0
        return scale * simd_dot(delta, forwardAxis)
    }
}

/// Baseline calibration state capturing neutral pose, noise covariance, and forward direction vector g.
public struct PostureCalibrationBaseline: Equatable, Sendable {
    public var centroid0: CGPoint
    public var rmsSpread0: CGFloat                   // Neutral RMS spread, reference for the scale ratio rho_t
    public var faceLength0: CGFloat                  // Neutral face length L_0, the unit of q_t and D
    public var imageAspect0: CGFloat                 // Image height / width during calibration
    public var faceShape0: [CGPoint]                 // Normalized neutral shape (see FacialConstellation.normalizedShape)
    public var noiseCovarianceDiagonal: SIMD3<Double> // Var(X), Var(Y), Var(Z)
    public var forwardVectorG: SIMD3<Double>          // Unit direction of forward head movement in camera frame
    public var kappa: Double                         // Anatomical ratio h / L_real in face lengths (default ~0.83)
    public var baselinePitch: Float                  // Neutral head pitch in camera frame (degrees)
    public var baselineJawAngle: Float               // Neutral mandibular angle (degrees)
    public var bodyAnchor: BodyAnchorCalibration?   // Nil when no reliable body anchor was seen during calibration
    public var bestUprightDisplacement: Double      // D_f of the best upright (chin tucked) posture, <= 0; 0 when not captured
    public var isCalibrated: Bool
    
    /// Distance from the habitual neutral to the best upright posture, in face lengths.
    public var correctionRange: Double { max(0.0, -bestUprightDisplacement) }
    
    public init(
        centroid0: CGPoint = CGPoint(x: 0.5, y: 0.5),
        rmsSpread0: CGFloat = 0.15,
        faceLength0: CGFloat = 0.15 / PostureMathEngine.nominalSpreadToFaceLength,
        imageAspect0: CGFloat = 1.0,
        faceShape0: [CGPoint] = [],
        noiseCovarianceDiagonal: SIMD3<Double> = SIMD3<Double>(0.001, 0.001, 0.005),
        forwardVectorG: SIMD3<Double> = SIMD3<Double>(0.7071, 0.20, 0.675),
        kappa: Double = 0.83,
        baselinePitch: Float = 0.0,
        baselineJawAngle: Float = 0.0,
        bodyAnchor: BodyAnchorCalibration? = nil,
        bestUprightDisplacement: Double = 0.0,
        isCalibrated: Bool = false
    ) {
        self.centroid0 = centroid0
        self.rmsSpread0 = rmsSpread0
        self.faceLength0 = faceLength0
        self.imageAspect0 = imageAspect0
        self.faceShape0 = faceShape0
        self.noiseCovarianceDiagonal = noiseCovarianceDiagonal
        self.forwardVectorG = forwardVectorG
        self.kappa = kappa
        self.baselinePitch = baselinePitch
        self.baselineJawAngle = baselineJawAngle
        self.bodyAnchor = bodyAnchor
        self.bestUprightDisplacement = bestUprightDisplacement
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
    public var deltaCVA: Double      // ΔCVA_t = CVA_rel,t - CVA_target (degrees); CVA_target = CVA_0 without a rehab target
    public var forwardDisplacementD: Double // Normalized forward displacement D_t
    public var displacementFace: Double    // D_f
    public var displacementBody: Double    // D_b
    public var quality: PostureQualityGate
    public var isSustainedForwardHead: Bool
    public var warningLevel: Int     // 0: Normal, 1: Mild, 2: Severe (see ForwardHeadAlertHysteresis)
    public var deltaPitch: Float     // Head pitch rotation relative to calibrated neutral baseline (degrees)
    public var deltaJawAngle: Float  // Mandibular jawline inclination relative to baseline (degrees)
    public var isOverextended: Bool  // Instantaneous backward hyperextension
    public var isSustainedOverextension: Bool // Hyperextension held long enough to notify
    public var profile: PostureProfile // Profile that produced CVA_0 and the alert thresholds
    public var targetCVA: Double     // CVA of the rehab target posture (degrees)
    public var targetDisplacementD: Double // D of the rehab target posture (<= 0, face lengths)
    
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
        deltaJawAngle: Float = 0.0,
        isOverextended: Bool = false,
        isSustainedOverextension: Bool = false,
        profile: PostureProfile = .default,
        targetCVA: Double? = nil,
        targetDisplacementD: Double = 0.0
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
        self.isOverextended = isOverextended
        self.isSustainedOverextension = isSustainedOverextension
        self.profile = profile
        self.targetCVA = targetCVA ?? profile.baseCvaDegrees
        self.targetDisplacementD = targetDisplacementD
    }
}

/// Advanced Posture Math Engine: Implements rigid facial constellation tracking,
/// camera-normalized 3D displacement q_t, forward-direction projection, and relative CVA.
public enum PostureMathEngine {
    
    /// Typical rigid-constellation RMS spread / face bounding-box height, measured on sample webcam frames
    /// (0.34 to 0.50 across poses). Used to express D in face lengths when no face height is available.
    public static let nominalSpreadToFaceLength: CGFloat = 0.4
    
    /// Nominal focal length in width-normalized units: f = 1 / (2 tan(HFOV / 2)) for a ~70° HFOV webcam.
    /// macOS exposes no field-of-view API for capture devices, so a typical built-in camera is assumed.
    public static let nominalFocalLength = 0.7
    
    /// Maps a Vision-normalized point (x by width, y by height) to isotropic width-normalized coordinates.
    public static func isotropicPoint(_ p: CGPoint, imageAspect: CGFloat) -> CGPoint {
        CGPoint(x: p.x, y: p.y * imageAspect)
    }
    
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
    /// Returns nil when no landmarks are available (no face detected) so callers never mistake a placeholder for a measurement.
    /// - Parameters:
    ///   - points: Vision-normalized image points.
    ///   - imageAspect: Image height / width, used to convert the points to isotropic coordinates.
    ///   - faceHeight: Vision-normalized face bounding-box height, if available.
    public static func computeConstellation(
        points visionPoints: [CGPoint],
        imageAspect: CGFloat = 1.0,
        faceHeight: CGFloat? = nil,
        captureQuality: Float = 1.0
    ) -> FacialConstellation? {
        guard !visionPoints.isEmpty else { return nil }
        
        let points = visionPoints.map { isotropicPoint($0, imageAspect: imageAspect) }
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
        
        return FacialConstellation(
            centroid: centroid,
            rmsSpread: rmsSpread,
            points: points,
            captureQuality: captureQuality,
            faceLength: faceHeight.flatMap { $0 > 0 ? $0 * imageAspect : nil },
            imageAspect: imageAspect
        )
    }
    
    /// RMS distance between two normalized shapes, in units of face RMS spread.
    /// Returns nil when the shapes are not comparable (different landmark counts).
    public static func shapeResidual(_ shape: [CGPoint], baseline: [CGPoint]) -> Double? {
        guard !shape.isEmpty, shape.count == baseline.count else { return nil }
        var diffSq: Double = 0.0
        for (p, b) in zip(shape, baseline) {
            let dx = Double(p.x - b.x)
            let dy = Double(p.y - b.y)
            diffSq += dx * dx + dy * dy
        }
        return sqrt(diffSq / Double(shape.count))
    }
    
    /// Element-wise mean of equally sized shapes. Returns nil if the shapes are empty or have mismatched counts.
    public static func meanShape(_ shapes: [[CGPoint]]) -> [CGPoint]? {
        guard let first = shapes.first, !first.isEmpty,
              shapes.allSatisfy({ $0.count == first.count }) else { return nil }
        let n = CGFloat(shapes.count)
        return (0..<first.count).map { i in
            CGPoint(
                x: shapes.reduce(0) { $0 + $1[i].x } / n,
                y: shapes.reduce(0) { $0 + $1[i].y } / n
            )
        }
    }
    
    // MARK: - 2. Camera-Normalized 3D Displacement (q_t)
    
    /// Computes camera-normalized 3D displacement vector q_t = [X_t, Y_t, Z_t]^T
    /// incorporating vertical elevation Y_t and perspective scale compensation r_t.
    ///
    /// Centroids, principal point, `baselineL0` (neutral image face length) and `focalLength` must all be in the
    /// same isotropic image units; q_t is then expressed in real face lengths.
    public static func computeCameraDisplacement3D(
        currentCentroid: CGPoint,
        scaleRho: CGFloat,
        baselineCentroid: CGPoint,
        baselineL0: CGFloat,
        principalPoint: CGPoint = CGPoint(x: 0.5, y: 0.5),
        focalLength: Double = nominalFocalLength
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
        let z = (focalLength / L0) * (1.0 - rT)
        
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
        baseCvaDegrees: Double = PostureProfile.default.baseCvaDegrees,
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
