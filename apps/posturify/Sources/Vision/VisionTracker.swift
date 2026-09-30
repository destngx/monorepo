import Foundation
import CoreGraphics
import Vision
import CoreMedia
import QuartzCore
import simd

/// Hardware-accelerated Apple Vision tracker running on Apple Neural Engine (ANE).
/// Concurrently extracts face landmarks (jawline, lips, head pose) and upper/lower body pose joints.
public final class VisionTracker: @unchecked Sendable {
    public static let shared = VisionTracker()
    
    private let faceRequest: VNDetectFaceLandmarksRequest
    private let bodyRequest: VNDetectHumanBodyPoseRequest
    private let segmentationRequest: VNGeneratePersonSegmentationRequest
    private let contourRequest: VNDetectContoursRequest
    private var baselineShoulderSpan: Float = 0.0
    private var isBaselineCalibrated: Bool = false
    private var lastCameraAngle: CameraAngleMode? = nil
    
    public init() {
        self.faceRequest = VNDetectFaceLandmarksRequest()
        self.bodyRequest = VNDetectHumanBodyPoseRequest()
        self.segmentationRequest = VNGeneratePersonSegmentationRequest()
        self.segmentationRequest.qualityLevel = .accurate
        self.contourRequest = VNDetectContoursRequest()
        self.contourRequest.contrastAdjustment = 1.0
        self.contourRequest.detectsDarkOnLight = false
        self.contourRequest.maximumImageDimension = 512
    }
    
    /// Processes a camera frame pixel buffer and resolves the full 3D skeletal kinematic state.
    public func processFrame(
        _ pixelBuffer: CVPixelBuffer,
        cameraAngle: CameraAngleMode = .diagonal45,
        cameraElevation: CameraElevationMode = .eyeLevel,
        timestamp: TimeInterval = CACurrentMediaTime()
    ) -> SkeletalKinematicState {
        let handler = VNImageRequestHandler(cvPixelBuffer: pixelBuffer, orientation: .up, options: [:])
        
        do {
            try handler.perform([faceRequest, bodyRequest, segmentationRequest])
            
            // Prioritize foreground primary user: pick face with largest bounding box area
            let faceObservation = faceRequest.results?.max(by: {
                ($0.boundingBox.width * $0.boundingBox.height) < ($1.boundingBox.width * $1.boundingBox.height)
            })
            let bodyObservation = bodyRequest.results?.first
            
            var contourPoints: [CGPoint] = []
            if let segObs = segmentationRequest.results?.first {
                let maskHandler = VNImageRequestHandler(cvPixelBuffer: segObs.pixelBuffer, orientation: .up, options: [:])
                try? maskHandler.perform([contourRequest])
                if let contourObs = contourRequest.results?.first,
                   let mainContour = contourObs.topLevelContours.max(by: { $0.pointCount < $1.pointCount }) {
                    contourPoints = mainContour.normalizedPoints.map { CGPoint(x: CGFloat($0.x), y: CGFloat($0.y)) }
                }
            }
            
            return resolveKinematicState(
                faceObs: faceObservation,
                bodyObs: bodyObservation,
                contourPoints: contourPoints,
                cameraAngle: cameraAngle,
                cameraElevation: cameraElevation,
                timestamp: timestamp
            )
        } catch {
            return SkeletalKinematicState(timestamp: timestamp, cameraAngle: cameraAngle, cameraElevation: cameraElevation)
        }
    }
    
    /// Resolves kinematics from Vision observations.
    public func resolveKinematicState(
        faceObs: VNFaceObservation?,
        bodyObs: VNHumanBodyPoseObservation?,
        contourPoints: [CGPoint] = [],
        cameraAngle: CameraAngleMode = .diagonal45,
        cameraElevation: CameraElevationMode = .eyeLevel,
        timestamp: TimeInterval
    ) -> SkeletalKinematicState {
        // 1. Extract Face & Mandibular Jaw Metrics
        var faceMetrics = extractFaceMetrics(from: faceObs)
        
        // 2. Extract Body Landmarks
        let bodyMetrics = extractBodyMetrics(from: bodyObs)
        
        // 3. Resolve Dynamic Bone Visibility (Conditional Culling)
        let visibility = KinematicsCalculator.resolveVisibilityFlags(face: faceMetrics, body: bodyMetrics)
        
        // 4. Clinical Keypoints (T: Tragus, A: Acromion, J: Jugular Notch, C7: Cervical base)
        
        let chinPoint = faceMetrics.chinGnathionPoint
        
        // Dynamically resolve facing direction from facial geometry
        let facingSign: CGFloat = {
            if faceMetrics.jawlinePoints.count >= 17 {
                let p1 = faceMetrics.jawlinePoints[1]
                let p15 = faceMetrics.jawlinePoints[15]
                let headMidX = (p1.x + p15.x) * 0.5
                if abs(chinPoint.x - headMidX) > 0.01 {
                    return chinPoint.x < headMidX ? -1.0 : 1.0
                }
            }
            if abs(faceMetrics.yawDegrees) > 2.0 {
                return faceMetrics.yawDegrees > 0 ? 1.0 : -1.0
            }
            return 1.0
        }()
        
        // Dynamically resolve Tragus (T) from ear or face contour tragion
        let tragusPoint: CGPoint = {
            if let earL = bodyMetrics.leftEar?.point, let earR = bodyMetrics.rightEar?.point {
                let confL = bodyMetrics.leftEar?.confidence ?? 0.0
                let confR = bodyMetrics.rightEar?.confidence ?? 0.0
                if max(confL, confR) > 0.4 {
                    return confL >= confR ? earL : earR
                }
            }
            if let ear = bodyMetrics.leftEar ?? bodyMetrics.rightEar, ear.confidence > 0.4 {
                return ear.point
            }
            if faceMetrics.jawlinePoints.count >= 17 {
                // Jawline point 2 (right tragion) and point 14 (left tragion) mark the anatomical ear tragus
                let p2 = faceMetrics.jawlinePoints[2]
                let p14 = faceMetrics.jawlinePoints[14]
                return (facingSign < 0) ? (p2.x > p14.x ? p2 : p14) : (p2.x < p14.x ? p2 : p14)
            }
            if let tL = faceMetrics.leftTragion, let tR = faceMetrics.rightTragion {
                return (facingSign < 0) ? (tL.x > tR.x ? tL : tR) : (tL.x < tR.x ? tL : tR)
            }
            if let t = faceMetrics.leftTragion ?? faceMetrics.rightTragion {
                return t
            }
            return chinPoint
        }()
        
        // Compute anatomical head yaw relative to torso / screen (compensates for oblique camera angle)
        if faceMetrics.confidence > 0.25 {
            faceMetrics.anatomicalYawDegrees = KinematicsCalculator.computeAnatomicalYaw(
                sensorYawDegrees: faceMetrics.yawDegrees,
                facingSign: Float(facingSign),
                cameraAngle: cameraAngle
            )
        } else {
            faceMetrics.anatomicalYawDegrees = 0.0
        }
        
        var isC7Estimated = false
        let acromionPoint: CGPoint
        let jugularNotchPoint: CGPoint
        let c7Point: CGPoint
        let headHeight = max(0.12, abs(tragusPoint.y - chinPoint.y))
        
        if !contourPoints.isEmpty && chinPoint != .zero && tragusPoint != .zero {
            // Anatomical Spatial ROI Gating:
            // Contours belonging to user's neck and shoulders must fall within a physiological radius of the head.
            // Rejects background objects (high-back chair headrests, wall shelves, curtains, background humans).
            let maxNeckLateralDistance = headHeight * 1.8
            let maxShoulderLateralDistance = headHeight * 2.8
            let maxInferiorDrop = headHeight * 2.5
            
            let anatomicallyGatedPoints = contourPoints.filter { pt in
                let dy = tragusPoint.y - pt.y
                let dx = abs(pt.x - tragusPoint.x)
                // Must be below tragus and within vertical body corridor
                guard dy >= -0.05 && dy <= maxInferiorDrop else { return false }
                // Within lateral span
                return dx <= (dy < headHeight * 0.8 ? maxNeckLateralDistance : maxShoulderLateralDistance)
            }
            
            let activeContourPoints = anatomicallyGatedPoints.isEmpty ? contourPoints : anatomicallyGatedPoints
            let posteriorPoints = activeContourPoints.filter { facingSign < 0 ? ($0.x > tragusPoint.x) : ($0.x < tragusPoint.x) }
            
            // 1. Dynamic C7 (Vertebra Prominens):
            // Sits at the posterior base of the neck below the cervical lordosis concavity
            let neckSlice = posteriorPoints.filter { $0.y <= tragusPoint.y - 0.03 && $0.y >= chinPoint.y - headHeight * 0.5 }
            let lordosisMin = (facingSign < 0)
                ? (neckSlice.min(by: { $0.x < $1.x }) ?? CGPoint(x: tragusPoint.x + 0.05, y: chinPoint.y))
                : (neckSlice.max(by: { $0.x < $1.x }) ?? CGPoint(x: tragusPoint.x - 0.05, y: chinPoint.y))
            
            let c7YTarget = chinPoint.y - headHeight * 0.15
            let c7Candidates = posteriorPoints.filter { $0.y <= lordosisMin.y && $0.y >= chinPoint.y - headHeight * 0.4 }
            let detectedC7 = c7Candidates.min(by: { abs($0.y - c7YTarget) < abs($1.y - c7YTarget) })
                ?? CGPoint(x: lordosisMin.x - facingSign * 0.04, y: c7YTarget)
            c7Point = detectedC7
            isC7Estimated = false
            
            // 2. Dynamic Acromion (A):
            // Lateral shelf of the shoulder shelf before the vertical arm drop
            let shoulderCandidates = posteriorPoints.filter { $0.y <= c7Point.y - 0.02 && $0.y >= c7Point.y - headHeight * 1.0 }
            let maxArmX = (facingSign < 0)
                ? (shoulderCandidates.map(\.x).max() ?? (c7Point.x + 0.25))
                : (shoulderCandidates.map(\.x).min() ?? (c7Point.x - 0.25))
            let targetX = c7Point.x + (maxArmX - c7Point.x) * 0.72
            let shoulderAtTarget = shoulderCandidates.min(by: { abs($0.x - targetX) < abs($1.x - targetX) })
            acromionPoint = shoulderAtTarget ?? CGPoint(x: targetX, y: c7Point.y - headHeight * 0.5)
            
            // 3. Dynamic Jugular Notch (J / Hõm ức):
            // Medial end of the clavicle at the suprasternal fossa between clavicular heads
            let throatX = chinPoint.x + (tragusPoint.x - chinPoint.x) * 0.55
            let throatY = acromionPoint.y - headHeight * 0.15
            jugularNotchPoint = CGPoint(x: throatX, y: throatY)
        } else {
            // Dynamic Acromion (A) fallback
            let shL = bodyMetrics.leftShoulder?.point
            let shR = bodyMetrics.rightShoulder?.point
            if let l = shL, let r = shR {
                acromionPoint = (facingSign > 0) ? (l.x < r.x ? l : r) : (l.x > r.x ? l : r)
            } else if let sh = shL ?? shR {
                acromionPoint = sh
            } else {
                let estShoulderY = min(tragusPoint.y, chinPoint.y) - headHeight * 0.35
                let estShoulderX = tragusPoint.x - facingSign * headHeight * 0.85
                acromionPoint = CGPoint(x: estShoulderX, y: estShoulderY)
            }
            
            // Dynamic C7 fallback: at cervical lordosis level
            let cervicalDrop = headHeight * 0.90
            c7Point = CGPoint(x: tragusPoint.x - facingSign * 0.08, y: tragusPoint.y - cervicalDrop)
            isC7Estimated = true
            
            // Dynamic Jugular Notch (J) fallback: at suprasternal level
            let throatX = chinPoint.x + (tragusPoint.x - chinPoint.x) * 0.55
            let throatY = acromionPoint.y - headHeight * 0.15
            jugularNotchPoint = CGPoint(x: throatX, y: throatY)
        }
        
        let keypoints = PosturalKeypoints2D(
            tragus: tragusPoint,
            acromion: acromionPoint,
            jugularNotch: jugularNotchPoint,
            c7: c7Point,
            isC7Estimated: isC7Estimated
        )
        
        var posture = KinematicsCalculator.computeClinicalPosture(
            tragus: tragusPoint,
            c7: c7Point,
            acromion: acromionPoint,
            jugularNotch: jugularNotchPoint,
            cameraAngle: cameraAngle
        )
        
        // Direct Angle Detection (Frontal / Direct facing camera, |yaw| < 12°)
        // In direct frontal view, sagittal cervical depth is severely foreshortened and CVA cannot be accurately computed
        let isDirectAngle = abs(faceMetrics.yawDegrees) < 12.0 && faceMetrics.confidence > 0.35
        if isDirectAngle {
            posture.isDirectAngleFacing = true
            posture.guidanceCue = "Direct angle detected. Move camera 45° to side for accurate tracking"
        }
        
        // 5. Advanced Relative CVA Proxy (Rigid Constellation + 3D Camera Displacement + Calibrated Forward Vector)
        let rigidPoints = PostureMathEngine.extractRigidConstellation(from: faceObs)
        let constellation = PostureMathEngine.computeConstellation(
            points: rigidPoints,
            baselineSpread: CalibrationEngine.shared.baseline.isCalibrated ? CalibrationEngine.shared.baseline.rmsSpread0 : nil,
            baselinePoints: CalibrationEngine.shared.baseline.isCalibrated ? CalibrationEngine.shared.baseline.facePoints0 : nil,
            captureQuality: faceObs?.faceCaptureQuality ?? faceMetrics.confidence
        )
        
        if CalibrationEngine.shared.currentPhase != .calibrated && CalibrationEngine.shared.currentPhase != .idle {
            CalibrationEngine.shared.recordCalibrationFrame(constellation: constellation)
        }
        
        if CalibrationEngine.shared.baseline.isCalibrated {
            // Compute head-to-body relative displacement if body keypoint is available
            let bodyAnchorY = c7Point != .zero ? Double(c7Point.y) : nil
            let relResult = CalibrationEngine.shared.evaluateFrame(
                constellation: constellation,
                bodyAnchorDisplacement: bodyAnchorY.map { Double(constellation.centroid.y) - $0 },
                timestamp: timestamp
            )
            
            // Map status guidance with relative delta:
            // - Level 2 or acute severe drop (Δ <= -15°): Turtle Neck
            // - Level 1 or noticeable drop (Δ <= -6°): Caution
            let relGuidance: String
            let relStatus: PostureStatus
            if relResult.warningLevel == 2 || relResult.deltaCVA <= -15.0 {
                relStatus = .turtleNeck
                relGuidance = String(format: "Turtle Neck: ΔCVA %.1f° (rel %.1f°). Tuck chin back!", relResult.deltaCVA, relResult.relativeCVA)
            } else if relResult.warningLevel == 1 || relResult.deltaCVA <= -6.0 {
                relStatus = .caution
                relGuidance = String(format: "Mild Forward Drift: ΔCVA %.1f° (rel %.1f°)", relResult.deltaCVA, relResult.relativeCVA)
            } else {
                relStatus = .optimal
                relGuidance = String(format: "Neutral Alignment (ΔCVA %.1f°)", relResult.deltaCVA)
            }
            
            // Dispatch macOS system notification if posture alert is active
            let activeWarningLevel = relStatus == .turtleNeck ? 2 : (relStatus == .caution ? 1 : 0)
            if activeWarningLevel > 0 {
                PostureNotificationManager.shared.checkAndNotify(
                    warningLevel: activeWarningLevel,
                    deltaCVA: relResult.deltaCVA,
                    relativeCVA: relResult.relativeCVA
                )
            }
            
            let finalGuidance = isDirectAngle ? "Direct angle detected. Move camera 45° to side for accurate tracking" : relGuidance
            
            posture = ClinicalPostureMetrics(
                cvaDegrees: relResult.relativeCVA,
                forwardDriftCm: relResult.forwardDisplacementD * 65.0,
                atjDegrees: posture.atjDegrees,
                shjRatio: posture.shjRatio,
                shjDegrees: posture.shjDegrees,
                thoracoCervicalAngle: posture.thoracoCervicalAngle,
                status: relStatus,
                guidanceCue: finalGuidance,
                deltaCvaDegrees: relResult.deltaCVA,
                isRelativeProxy: true,
                forwardDisplacementD: relResult.forwardDisplacementD,
                isDirectAngleFacing: isDirectAngle
            )
        }
        
        // Recalibrate baseline if camera angle mode was switched
        if lastCameraAngle != cameraAngle {
            lastCameraAngle = cameraAngle
            isBaselineCalibrated = false
        }
        
        return SkeletalKinematicState(
            timestamp: timestamp,
            face: faceMetrics,
            body: bodyMetrics,
            keypoints: keypoints,
            visibility: visibility,
            posture: posture,
            cameraAngle: cameraAngle,
            cameraElevation: cameraElevation,
            cervicalVertebrae: [],
            thoracicVertebrae: [],
            lumbarVertebrae: [],
            leftClaviclePosition: SIMD3<Float>(-0.12, 0.56, 0.02),
            rightClaviclePosition: SIMD3<Float>(0.12, 0.56, 0.02),
            sternumPosition: SIMD3<Float>(0.0, 0.48, 0.04),
            dynamicSkullPosition: SIMD3<Float>(0.0, 0.77, 0.0),
            dynamicC7Position: SIMD3<Float>(0.0, 0.60, -0.04),
            dynamicT12Position: SIMD3<Float>(0.0, 0.33, 0.0),
            dynamicLumbarBasePosition: SIMD3<Float>(0.0, 0.18, 0.0),
            torsoRotation: .zero,
            facingSign: Float(facingSign),
            silhouetteContour: contourPoints
        )
    }
    
    // MARK: - Face Metrics Extraction
    
    private func extractFaceMetrics(from observation: VNFaceObservation?) -> TrackedFaceMetrics {
        guard let obs = observation, let landmarks = obs.landmarks else {
            return TrackedFaceMetrics()
        }
        
        let bbox = obs.boundingBox
        
        // Yaw, pitch, roll
        let pitch = Float(obs.pitch?.doubleValue ?? 0.0) * (180.0 / .pi)
        let yaw = Float(obs.yaw?.doubleValue ?? 0.0) * (180.0 / .pi)
        let roll = Float(obs.roll?.doubleValue ?? 0.0) * (180.0 / .pi)
        
        // Extract lips for mouth aperture calculation
        var upperLipPoint = CGPoint(x: bbox.midX, y: bbox.midY - bbox.height * 0.15)
        var lowerLipPoint = CGPoint(x: bbox.midX, y: bbox.midY - bbox.height * 0.25)
        
        if let innerLips = landmarks.innerLips, innerLips.pointCount >= 6 {
            let pts = innerLips.normalizedPoints
            // Upper center and lower center
            let topIdx = innerLips.pointCount / 4
            let botIdx = (3 * innerLips.pointCount) / 4
            upperLipPoint = convertFacePoint(pts[topIdx], in: bbox)
            lowerLipPoint = convertFacePoint(pts[botIdx], in: bbox)
        } else if let outerLips = landmarks.outerLips, outerLips.pointCount >= 8 {
            let pts = outerLips.normalizedPoints
            let topIdx = outerLips.pointCount / 4
            let botIdx = (3 * outerLips.pointCount) / 4
            upperLipPoint = convertFacePoint(pts[topIdx], in: bbox)
            lowerLipPoint = convertFacePoint(pts[botIdx], in: bbox)
        }
        
        // Chin apex (Gnathion) from face contour
        var chinPoint = CGPoint(x: bbox.midX, y: bbox.minY)
        var jawlinePoints: [CGPoint] = []
        if let faceContour = landmarks.faceContour, faceContour.pointCount > 0 {
            jawlinePoints = faceContour.normalizedPoints.map { convertFacePoint($0, in: bbox) }
            let midIndex = faceContour.pointCount / 2
            chinPoint = jawlinePoints[midIndex]
        }
        
        // Extract key facial landmarks for geometric orientation estimation
        var leftEyePoint: CGPoint? = nil
        if let leftEye = landmarks.leftEye, leftEye.pointCount > 0 {
            let pts = leftEye.normalizedPoints.map { convertFacePoint($0, in: bbox) }
            let sumX = pts.reduce(0) { $0 + $1.x }
            let sumY = pts.reduce(0) { $0 + $1.y }
            leftEyePoint = CGPoint(x: sumX / CGFloat(pts.count), y: sumY / CGFloat(pts.count))
        }
        
        var rightEyePoint: CGPoint? = nil
        if let rightEye = landmarks.rightEye, rightEye.pointCount > 0 {
            let pts = rightEye.normalizedPoints.map { convertFacePoint($0, in: bbox) }
            let sumX = pts.reduce(0) { $0 + $1.x }
            let sumY = pts.reduce(0) { $0 + $1.y }
            rightEyePoint = CGPoint(x: sumX / CGFloat(pts.count), y: sumY / CGFloat(pts.count))
        }
        
        var nosePoint: CGPoint? = nil
        if let nose = landmarks.nose, nose.pointCount > 0 {
            let pts = nose.normalizedPoints.map { convertFacePoint($0, in: bbox) }
            nosePoint = pts.min(by: { $0.y < $1.y })
        } else if let noseCrest = landmarks.noseCrest, noseCrest.pointCount > 0 {
            let pts = noseCrest.normalizedPoints.map { convertFacePoint($0, in: bbox) }
            nosePoint = pts.min(by: { $0.y < $1.y })
        }
        
        // Compute facial angles (pitch, yaw, roll) directly from camera landmark geometry
        let faceAngles = KinematicsCalculator.computeFaceAnglesFromLandmarks(
            leftEye: leftEyePoint,
            rightEye: rightEyePoint,
            nose: nosePoint,
            chin: chinPoint,
            leftCheek: jawlinePoints.first,
            rightCheek: jawlinePoints.last,
            visionPitch: pitch,
            visionYaw: yaw,
            visionRoll: roll
        )
        
        // Calculate mouth aperture fraction & TMJ rotation
        let aperture = KinematicsCalculator.computeMouthAperture(
            upperLip: upperLipPoint,
            lowerLip: lowerLipPoint,
            chin: chinPoint,
            faceBoundingBoxHeight: bbox.height
        )
        let tmjDegrees = KinematicsCalculator.computeTMJRotationDegrees(apertureFraction: aperture)
        
        return TrackedFaceMetrics(
            mouthApertureFraction: aperture,
            tmjRotationDegrees: tmjDegrees,
            pitchDegrees: faceAngles.pitchDegrees,
            yawDegrees: faceAngles.yawDegrees,
            rollDegrees: faceAngles.rollDegrees,
            jawlinePoints: jawlinePoints,
            upperLipPoint: upperLipPoint,
            lowerLipPoint: lowerLipPoint,
            chinGnathionPoint: chinPoint,
            leftTragion: jawlinePoints.count >= 15 ? jawlinePoints[14] : jawlinePoints.last,
            rightTragion: jawlinePoints.count >= 3 ? jawlinePoints[2] : jawlinePoints.first,
            confidence: obs.confidence
        )
    }
    
    // MARK: - Body Metrics Extraction
    
    private func extractBodyMetrics(from observation: VNHumanBodyPoseObservation?) -> TrackedBodyMetrics {
        guard let obs = observation else {
            return TrackedBodyMetrics()
        }
        
        return TrackedBodyMetrics(
            nose: getPoint(from: obs, joint: .nose),
            leftEye: getPoint(from: obs, joint: .leftEye),
            rightEye: getPoint(from: obs, joint: .rightEye),
            leftEar: getPoint(from: obs, joint: .leftEar),
            rightEar: getPoint(from: obs, joint: .rightEar),
            neckC7: getPoint(from: obs, joint: .neck),
            leftShoulder: getPoint(from: obs, joint: .leftShoulder),
            rightShoulder: getPoint(from: obs, joint: .rightShoulder),
            root: getPoint(from: obs, joint: .root),
            leftHip: getPoint(from: obs, joint: .leftHip),
            rightHip: getPoint(from: obs, joint: .rightHip)
        )
    }
    
    private func getPoint(from obs: VNHumanBodyPoseObservation, joint: VNHumanBodyPoseObservation.JointName) -> TrackedJoint? {
        guard let point = try? obs.recognizedPoint(joint), point.confidence > 0.1 else {
            return nil
        }
        return TrackedJoint(point: point.location, confidence: point.confidence)
    }
    
    private func convertFacePoint(_ normalizedPt: CGPoint, in bbox: CGRect) -> CGPoint {
        CGPoint(
            x: bbox.origin.x + normalizedPt.x * bbox.width,
            y: bbox.origin.y + normalizedPt.y * bbox.height
        )
    }
    
    // MARK: - Synthetic Simulation Generator
    
    /// Generates a realistic synthetic kinematic cycle for offline testing and developer verification.
    /// Cycles through mouth opening, head rotation, spine slouching, and camera framing adjustments.
    public func generateSyntheticKinematicFrame(time: Double) -> SkeletalKinematicState {
        let cycle = time.truncatingRemainder(dividingBy: 12.0) // 12 second full kinematic demo cycle
        
        // Phase 1 (0s - 4s): Optimal Alignment & Clean Mouth Opening
        // Phase 2 (4s - 8s): Turtle Neck (Head drifts forward +3.4 cm, CVA drops)
        // Phase 3 (8s - 12s): Slouching / Camera Zoom (Lower spine culled)
        
        let mouthPhase: Float
        if cycle < 2.0 {
            mouthPhase = Float(sin((cycle / 2.0) * .pi * 0.5)) // 0.0 to 1.0 opening
        } else if cycle < 4.0 {
            mouthPhase = Float(cos(((cycle - 2.0) / 2.0) * .pi * 0.5)) // 1.0 to 0.0 closing
        } else {
            mouthPhase = 0.0
        }
        
        let aperture = mouthPhase * 0.85
        let tmjDegrees = KinematicsCalculator.computeTMJRotationDegrees(apertureFraction: aperture)
        
        // Turtle Neck forward drift simulation (Phase 2: 4s - 8s)
        let forwardHeadShift: CGFloat
        if cycle >= 4.0 && cycle < 8.0 {
            let t = CGFloat((cycle - 4.0) / 4.0)
            forwardHeadShift = sin(t * .pi) * 0.06 // +0.06 normalized shift ≈ +3.8 cm drift
        } else {
            forwardHeadShift = 0.003
        }
        
        // Head pitch/yaw during nod
        let pitch: Float = (cycle >= 4.0 && cycle < 8.0) ? Float(sin((cycle - 4.0) / 4.0 * .pi) * 12.0) : 0.0
        let yaw: Float = 0.0
        let roll: Float = 0.0
        let lateralTilt: Float = (cycle >= 8.0) ? Float(sin((cycle - 8.0) / 4.0 * .pi) * 0.06) : 0.0
        
        // Camera framing shift: hips out of view during Phase 3 (8s - 12s)
        let isHipInFrame = (cycle < 8.0)
        
        let shoulderX: CGFloat = 0.50
        let earX = shoulderX + forwardHeadShift
        
        let faceMetrics = TrackedFaceMetrics(
            mouthApertureFraction: aperture,
            tmjRotationDegrees: tmjDegrees,
            pitchDegrees: pitch,
            yawDegrees: yaw,
            anatomicalYawDegrees: yaw,
            rollDegrees: roll,
            jawlinePoints: [
                CGPoint(x: earX - 0.10, y: 0.75), CGPoint(x: earX - 0.06, y: 0.70), CGPoint(x: earX - 0.02, y: 0.67),
                CGPoint(x: earX + 0.02, y: 0.65), CGPoint(x: earX + 0.06, y: 0.67), CGPoint(x: earX + 0.10, y: 0.70)
            ],
            upperLipPoint: CGPoint(x: earX + 0.02, y: 0.71),
            lowerLipPoint: CGPoint(x: earX + 0.02, y: 0.71 - CGFloat(aperture * 0.04)),
            chinGnathionPoint: CGPoint(x: earX + 0.04, y: 0.65),
            leftTragion: CGPoint(x: earX, y: 0.76),
            rightTragion: CGPoint(x: earX + 0.12, y: 0.76),
            confidence: 0.95
        )
        
        let hipY: CGFloat = isHipInFrame ? 0.15 : 0.02
        let hipConf: Float = isHipInFrame ? 0.90 : 0.15
        
        let bodyMetrics = TrackedBodyMetrics(
            nose: TrackedJoint(point: CGPoint(x: earX + 0.06, y: 0.75), confidence: 0.95),
            leftEye: TrackedJoint(point: CGPoint(x: earX + 0.04, y: 0.78), confidence: 0.95),
            rightEye: TrackedJoint(point: CGPoint(x: earX + 0.08, y: 0.78), confidence: 0.95),
            leftEar: TrackedJoint(point: CGPoint(x: earX, y: 0.76), confidence: 0.95),
            rightEar: TrackedJoint(point: CGPoint(x: earX + 0.12, y: 0.76), confidence: 0.90),
            neckC7: TrackedJoint(point: CGPoint(x: 0.50, y: 0.62), confidence: 0.95),
            leftShoulder: TrackedJoint(point: CGPoint(x: 0.35, y: 0.55), confidence: 0.95),
            rightShoulder: TrackedJoint(point: CGPoint(x: 0.65, y: 0.55), confidence: 0.95),
            root: TrackedJoint(point: CGPoint(x: 0.50, y: hipY), confidence: hipConf),
            leftHip: TrackedJoint(point: CGPoint(x: 0.42, y: hipY), confidence: hipConf),
            rightHip: TrackedJoint(point: CGPoint(x: 0.58, y: hipY), confidence: hipConf)
        )
        
        let visibility = KinematicsCalculator.resolveVisibilityFlags(face: faceMetrics, body: bodyMetrics)
        
        let keypoints = PosturalKeypoints2D(
            tragus: CGPoint(x: earX, y: 0.76),
            acromion: CGPoint(x: 0.35, y: 0.55),
            jugularNotch: CGPoint(x: 0.55, y: 0.55),
            c7: CGPoint(x: 0.44, y: 0.55),
            isC7Estimated: false
        )
        
        let posture = KinematicsCalculator.computeClinicalPosture(
            tragus: CGPoint(x: earX, y: 0.76),
            c7: CGPoint(x: 0.44, y: 0.55),
            acromion: CGPoint(x: 0.35, y: 0.55),
            jugularNotch: CGPoint(x: 0.55, y: 0.55),
            cameraAngle: .diagonal45
        )
        
        let torsoRot = SIMD3<Float>(0.0, 0.0, lateralTilt)
        let (dynamicSkullPos, dynamicC7Pos, dynamicT12Pos, dynamicLumbarBasePos) = KinematicsCalculator.computeDynamicPositions(
            head: CGPoint(x: earX, y: 0.76),
            neck: CGPoint(x: 0.50, y: 0.62),
            root: CGPoint(x: 0.50, y: hipY),
            posture: posture,
            torsoRoll: lateralTilt
        )
        
        let headRot = SIMD3<Float>(pitch * .pi / 180.0, yaw * .pi / 180.0, roll * .pi / 180.0)
        let cervicalVertebrae = KinematicsCalculator.computeCervicalVertebrae(
            skullPos: dynamicSkullPos,
            c7Pos: dynamicC7Pos,
            headRotation: headRot,
            torsoRotation: torsoRot,
            isVisible: visibility.isCervicalSpineVisible
        )
        
        let thoracicVertebrae = KinematicsCalculator.computeThoracicVertebrae(
            c7Pos: dynamicC7Pos,
            t12Pos: dynamicT12Pos,
            lateralTilt: lateralTilt,
            torsoRotation: torsoRot,
            isVisible: visibility.isThoracicSpineVisible
        )
        
        let lumbarVertebrae = KinematicsCalculator.computeLumbarVertebrae(
            t12Pos: dynamicT12Pos,
            basePos: dynamicLumbarBasePos,
            torsoRotation: torsoRot,
            isVisible: visibility.isLumbarSpineVisible
        )
        
        return SkeletalKinematicState(
            timestamp: time,
            face: faceMetrics,
            body: bodyMetrics,
            keypoints: keypoints,
            visibility: visibility,
            posture: posture,
            cameraAngle: .diagonal45,
            cervicalVertebrae: cervicalVertebrae,
            thoracicVertebrae: thoracicVertebrae,
            lumbarVertebrae: lumbarVertebrae,
            leftClaviclePosition: SIMD3<Float>(-0.12, 0.56, 0.02),
            rightClaviclePosition: SIMD3<Float>(0.12, 0.56, 0.02),
            sternumPosition: SIMD3<Float>(0.0, 0.48, 0.04),
            dynamicSkullPosition: dynamicSkullPos,
            dynamicC7Position: dynamicC7Pos,
            dynamicT12Position: dynamicT12Pos,
            dynamicLumbarBasePosition: dynamicLumbarBasePos,
            torsoRotation: torsoRot
        )
    }
}
