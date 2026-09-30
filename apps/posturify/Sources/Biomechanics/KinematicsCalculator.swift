import Foundation
import CoreGraphics
import simd

/// Biomechanical kinematics calculator that computes mouth aperture, TMJ jaw rotation,
/// spinal spline curves for discrete vertebrae, and dynamic camera-boundary visibility culling.
public enum KinematicsCalculator {
    
    // MARK: - Constants
    
    public static let maxTMJRotationDegrees: Float = 24.0 // Anatomical maximum TMJ opening hinge angle
    public static let restingApertureThreshold: Float = 0.025 // Sensitive resting threshold
    public static let fullApertureThreshold: Float = 0.18     // 18% of face height represents natural wide open mouth
    
    // Standard anatomical rest positions (coordinates in meters)
    public static let defaultSkullPosition = SIMD3<Float>(0.0, 0.77, 0.0)
    public static let defaultC7Position = SIMD3<Float>(0.0, 0.60, -0.04)
    public static let defaultT12Position = SIMD3<Float>(0.0, 0.33, 0.0)
    public static let defaultLumbarBasePosition = SIMD3<Float>(0.0, 0.18, 0.0)
    
    // Default user anthropometric body dimensions:
    // Ngang vai (biacromial shoulder span) = 42 cm (0.42 m)
    // Vòng ngực (chest circumference) = 83 cm (0.83 m)
    public static let defaultShoulderSpanCm: Float = 42.0
    public static let defaultChestCircumferenceCm: Float = 83.0
    
    // MARK: - Clinical Craniovertebral Angle (CVA) & Posture Engine
    
    /// Stricter clinical biomechanical constraints for Craniovertebral Angle (CVA) and forward head drift.
    public enum PostureConstraints {
        /// Healthy neutral cervical spine threshold (CVA > 60.0°)
        public static let optimalCvaMinDegrees: Double = 60.0
        /// Turtle Neck / severe cervical strain threshold (CVA < 55.0°)
        public static let turtleNeckCvaMaxDegrees: Double = 55.0
        
        /// Plumb Line Sagittal Forward Drift: neutral head threshold (drift <= 0.8 cm)
        public static let optimalDriftMaxCm: Double = 0.8
        /// Plumb Line Sagittal Forward Drift: turtle neck threshold (drift > 1.8 cm)
        public static let turtleNeckDriftMinCm: Double = 1.8
        
        /// Thoraco-cervical slouch angle threshold (degrees)
        public static let slouchAngleMinDegrees: Double = 145.0
    }
    
    // MARK: - General Dynamic Posture Angle Engine (30°–90°)
    
    /// General formula to calculate true sagittal angle between any two 2D vectors u and v projected at camera angle alpha (30°–90°).
    ///
    /// Formula:
    ///   k_u = dx_u / dy_u
    ///   k_v = dx_v / dy_v
    ///   theta_true = | arctan(k_u / sin alpha) - arctan(k_v / sin alpha) |
    public static func computeDynamicAngle(
        dxU: Double, dyU: Double,
        dxV: Double, dyV: Double,
        alphaDegrees: Double
    ) -> Double {
        let alphaRad = max(30.0, min(90.0, alphaDegrees)) * .pi / 180.0
        let sinAlpha = sin(alphaRad)
        
        let safeDyU = (abs(dyU) < 0.0001) ? (dyU >= 0 ? 0.0001 : -0.0001) : dyU
        let safeDyV = (abs(dyV) < 0.0001) ? (dyV >= 0 ? 0.0001 : -0.0001) : dyV
        
        let kU = dxU / safeDyU
        let kV = dxV / safeDyV
        
        let phiU = atan(kU / sinAlpha)
        let phiV = atan(kV / sinAlpha)
        
        let angleRad = abs(phiU - phiV)
        return angleRad * 180.0 / .pi
    }
    
    /// Computes the true sagittal Craniovertebral Angle (CVA in degrees) for camera angles 30°–90°.
    ///
    /// Formula:
    ///   CVA_true = arctan((dy_CT * sin alpha) / dx_CT)
    public static func computeDynamicCVA(
        tragus: CGPoint,
        c7: CGPoint,
        alphaDegrees: Double
    ) -> Double {
        let dx = abs(tragus.x - c7.x)
        let dy = abs(tragus.y - c7.y)
        guard dx > 0.0001 else { return 90.0 }
        
        let alphaRad = max(30.0, min(90.0, alphaDegrees)) * .pi / 180.0
        let sinAlpha = sin(alphaRad)
        
        let cvaRad = atan((dy * sinAlpha) / dx)
        return min(90.0, max(0.0, cvaRad * 180.0 / .pi))
    }
    
    /// Computes the true sagittal ATJ Angle (Acromion - Tragus - Jugular Notch in degrees)
    /// measured at the Tragus between vector TA and vector TJ.
    public static func computeDynamicATJ(
        tragus: CGPoint,
        acromion: CGPoint,
        jugularNotch: CGPoint,
        alphaDegrees: Double
    ) -> Double {
        let dxTA = abs(acromion.x - tragus.x)
        let dyTA = abs(acromion.y - tragus.y)
        
        let dxTJ = abs(jugularNotch.x - tragus.x)
        let dyTJ = abs(jugularNotch.y - tragus.y)
        
        return computeDynamicAngle(
            dxU: dxTA, dyU: dyTA,
            dxV: dxTJ, dyV: dyTJ,
            alphaDegrees: alphaDegrees
        )
    }
    
    /// Computes the SHJ (Shoulder - Head - Jugular) index:
    /// Ratio form: SHJ = d_AT / d_AJ (invariant across angles 30°–90° as sin alpha cancels)
    /// Angle form: theta_AT,AJ using the general dynamic angle formula.
    public static func computeDynamicSHJ(
        tragus: CGPoint,
        acromion: CGPoint,
        jugularNotch: CGPoint,
        alphaDegrees: Double
    ) -> (ratio: Double, angleDegrees: Double) {
        let dAT = abs(tragus.x - acromion.x)
        let dAJ = abs(jugularNotch.x - acromion.x)
        let ratio = dAT / max(0.0001, dAJ)
        
        let dyAT = abs(tragus.y - acromion.y)
        let dyAJ = abs(jugularNotch.y - acromion.y)
        
        let angle = computeDynamicAngle(
            dxU: dAT, dyU: dyAT,
            dxV: dAJ, dyV: dyAJ,
            alphaDegrees: alphaDegrees
        )
        
        return (ratio, angle)
    }
    
    /// Estimates C7 vertebra location from Jugular Notch (J) and Acromion (A) when C7 is occluded.
    public static func estimateC7(
        jugularNotch: CGPoint,
        acromion: CGPoint
    ) -> CGPoint {
        let estX = 2.0 * acromion.x - jugularNotch.x
        let estY = max(jugularNotch.y, acromion.y) + 0.02
        return CGPoint(x: estX, y: estY)
    }
    
    /// Primary posture assessment method taking the 4 key landmarks (T, A, J, C7) and camera angle (30°–90°).
    /// If C7 is nil, automatically estimates C7 from jugularNotch and acromion.
    public static func computeClinicalPosture(
        tragus: CGPoint,
        c7: CGPoint? = nil,
        acromion: CGPoint,
        jugularNotch: CGPoint,
        cameraAngle: CameraAngleMode
    ) -> ClinicalPostureMetrics {
        let alpha = cameraAngle.degrees
        let resolvedC7 = c7 ?? estimateC7(jugularNotch: jugularNotch, acromion: acromion)
        let cva = computeDynamicCVA(tragus: tragus, c7: resolvedC7, alphaDegrees: alpha)
        let atj = computeDynamicATJ(tragus: tragus, acromion: acromion, jugularNotch: jugularNotch, alphaDegrees: alpha)
        let shj = computeDynamicSHJ(tragus: tragus, acromion: acromion, jugularNotch: jugularNotch, alphaDegrees: alpha)
        
        // Sagittal forward plumb drift: (d_AT / sin(alpha)) * scaleFactor
        let sinAlpha = max(0.5, sin(alpha * .pi / 180.0))
        let sagittalShift = abs(tragus.x - acromion.x) / sinAlpha
        let forwardDriftCm = Double(sagittalShift * 65.0)
        
        let status: PostureStatus
        let guidance: String
        if cva < 55.0 {
            status = .turtleNeck
            guidance = String(format: "Turtle Neck: CVA %.1f° (<55°). Tuck chin & align ears over shoulders", cva)
        } else if cva <= 60.0 {
            status = .caution
            guidance = String(format: "Caution: CVA %.1f° (55°–60°). Mild forward head tilt detected", cva)
        } else {
            status = .optimal
            guidance = "Optimal Alignment: Spine in neutral posture"
        }
        
        return ClinicalPostureMetrics(
            cvaDegrees: cva,
            forwardDriftCm: forwardDriftCm,
            atjDegrees: atj,
            shjRatio: shj.ratio,
            shjDegrees: shj.angleDegrees,
            thoracoCervicalAngle: 165.0,
            status: status,
            guidanceCue: guidance
        )
    }
    
    /// Compatibility overload mapping head/neck/shoulder/chest parameters to T, C7, A, J.
    public static func computeClinicalPosture(
        head: CGPoint,       // Tragion (Ear canal) -> T
        neck: CGPoint,       // C7 Prominens -> C7
        shoulder: CGPoint,   // Acromion (Shoulder tip) -> A
        chest: CGPoint? = nil, // Jugular Notch -> J
        facingSign: CGFloat = 1.0,
        cameraAngle: CameraAngleMode = .diagonal45,
        cameraElevation: CameraElevationMode = .eyeLevel,
        effectiveElevationDegrees: Double? = nil
    ) -> ClinicalPostureMetrics {
        let j = chest ?? CGPoint(x: (shoulder.x + neck.x) * 0.5, y: shoulder.y)
        var metrics = computeClinicalPosture(
            tragus: head,
            c7: neck,
            acromion: shoulder,
            jugularNotch: j,
            cameraAngle: cameraAngle
        )
        let elev = effectiveElevationDegrees ?? cameraElevation.pitchDegrees
        if elev != 0.0 {
            let adjustedCVA = max(10.0, min(89.0, metrics.cvaDegrees + elev * 0.4))
            metrics = ClinicalPostureMetrics(
                cvaDegrees: adjustedCVA,
                forwardDriftCm: metrics.forwardDriftCm,
                atjDegrees: metrics.atjDegrees,
                shjRatio: metrics.shjRatio,
                shjDegrees: metrics.shjDegrees,
                thoracoCervicalAngle: metrics.thoracoCervicalAngle,
                status: metrics.status,
                guidanceCue: metrics.guidanceCue
            )
        }
        return metrics
    }
    
    // MARK: - Anatomical Orientation & Perspective Compensation
    
    public struct FaceLandmarkAngles: Equatable {
        public let pitchDegrees: Float
        public let yawDegrees: Float
        public let rollDegrees: Float
        
        public init(pitchDegrees: Float, yawDegrees: Float, rollDegrees: Float) {
            self.pitchDegrees = pitchDegrees
            self.yawDegrees = yawDegrees
            self.rollDegrees = rollDegrees
        }
    }
    
    /// Computes accurate 3D face angles (pitch, yaw, roll in degrees) directly from 2D facial landmark coordinates
    /// and fuses them with Vision sensor estimates for smooth, responsive orientation.
    public static func computeFaceAnglesFromLandmarks(
        leftEye: CGPoint?,
        rightEye: CGPoint?,
        nose: CGPoint?,
        chin: CGPoint? = nil,
        leftCheek: CGPoint? = nil,
        rightCheek: CGPoint? = nil,
        visionPitch: Float? = nil,
        visionYaw: Float? = nil,
        visionRoll: Float? = nil
    ) -> FaceLandmarkAngles {
        // 1. Roll: direct geometric angle of the inter-ocular vector
        let resolvedRoll: Float
        if let eyeL = leftEye, let eyeR = rightEye {
            // Note: in camera image coordinates, subject's left eye has larger X than subject's right eye
            let dx = Float(eyeL.x - eyeR.x)
            let dy = Float(eyeL.y - eyeR.y) // in Vision normalized coordinates, +Y is up
            let landmarkRoll = atan2(dy, max(0.001, abs(dx))) * (180.0 / .pi)
            if let vRoll = visionRoll {
                resolvedRoll = 0.70 * landmarkRoll + 0.30 * vRoll
            } else {
                resolvedRoll = landmarkRoll
            }
        } else {
            resolvedRoll = visionRoll ?? 0.0
        }
        
        // 2. Yaw: horizontal asymmetry of the nose relative to eye midpoint and cheeks
        let resolvedYaw: Float
        if let eyeL = leftEye, let eyeR = rightEye, let n = nose {
            let midX = (eyeL.x + eyeR.x) * 0.5
            let midY = (eyeL.y + eyeR.y) * 0.5
            let eyeSpan = hypot(eyeL.x - eyeR.x, eyeL.y - eyeR.y)
            let halfSpan = max(0.001, eyeSpan * 0.5)
            
            // Unit vector along eye line pointing from right eye to left eye (+X in face coordinates)
            let ux = (eyeL.x - eyeR.x) / eyeSpan
            let uy = (eyeL.y - eyeR.y) / eyeSpan
            
            // Projection of nose offset along inter-ocular axis
            let dNoseX = n.x - midX
            let dNoseY = n.y - midY
            let projNose = (dNoseX * ux + dNoseY * uy)
            let noseRatio = Float(projNose / halfSpan)
            
            // Cheek asymmetry ratio if bilateral jaw/cheek points are available
            var cheekRatio: Float = noseRatio
            if let cL = leftCheek, let cR = rightCheek {
                let distL = hypot(cL.x - n.x, cL.y - n.y)
                let distR = hypot(cR.x - n.x, cR.y - n.y)
                let sum = distL + distR
                if sum > 0.001 {
                    cheekRatio = Float((distR - distL) / sum)
                }
            }
            
            let combinedRatio = max(-0.95, min(0.95, 0.65 * noseRatio + 0.35 * cheekRatio))
            let landmarkYaw = asin(combinedRatio) * (180.0 / .pi)
            
            if let vYaw = visionYaw {
                resolvedYaw = 0.65 * landmarkYaw + 0.35 * vYaw
            } else {
                resolvedYaw = landmarkYaw
            }
        } else {
            resolvedYaw = visionYaw ?? 0.0
        }
        
        // 3. Pitch: vertical facial thirds ratio (eye-to-nose vs nose-to-chin)
        let resolvedPitch: Float
        if let eyeL = leftEye, let eyeR = rightEye, let n = nose {
            let midX = (eyeL.x + eyeR.x) * 0.5
            let midY = (eyeL.y + eyeR.y) * 0.5
            let eyeSpan = hypot(eyeL.x - eyeR.x, eyeL.y - eyeR.y)
            
            // Unit vector perpendicular to eye line pointing upwards (+Y in face coordinates)
            let ux = (eyeL.x - eyeR.x) / eyeSpan
            let uy = (eyeL.y - eyeR.y) / eyeSpan
            let vx = -uy
            let vy = ux
            
            // Distance of nose below eye line: -(vector from eyeMid to nose dot v)
            let hNose = -Float((n.x - midX) * vx + (n.y - midY) * vy)
            
            let landmarkPitch: Float
            if let c = chin {
                let hFace = -Float((c.x - midX) * vx + (c.y - midY) * vy)
                let ratio = hNose / max(0.01, hFace)
                let neutralRatio: Float = 0.45
                // Tilting up (extension) reduces hNose/hFace ratio -> positive pitch
                // Tilting down (flexion) increases ratio -> negative pitch
                landmarkPitch = -120.0 * (ratio - neutralRatio)
            } else {
                let ratio = hNose / max(0.01, Float(eyeSpan))
                let neutralRatio: Float = 0.60
                landmarkPitch = -85.0 * (ratio - neutralRatio)
            }
            
            let clampedLandmarkPitch = max(-55.0, min(55.0, landmarkPitch))
            if let vPitch = visionPitch {
                resolvedPitch = 0.60 * clampedLandmarkPitch + 0.40 * vPitch
            } else {
                resolvedPitch = clampedLandmarkPitch
            }
        } else {
            resolvedPitch = visionPitch ?? 0.0
        }
        
        return FaceLandmarkAngles(
            pitchDegrees: resolvedPitch,
            yawDegrees: resolvedYaw,
            rollDegrees: resolvedRoll
        )
    }
    
    /// Computes anatomical head yaw relative to the user's torso / screen direction (0° = looking directly at screen).
    /// Vision's VNFaceObservation.yaw reports rotation relative to camera sensor optical axis.
    /// This removes the baseline camera viewing angle so the skull and cervical spine align naturally with the chest.
    public static func computeAnatomicalYaw(
        sensorYawDegrees: Float,
        facingSign: Float = 1.0,
        cameraAngle: CameraAngleMode
    ) -> Float {
        let baselineYaw = facingSign * Float(cameraAngle.degrees)
        return sensorYawDegrees - baselineYaw
    }
    
    /// Computes 3D rotation (pitch, yaw, roll in radians) of the torso and shoulders.
    /// Operates independently of head rotation so the head and shoulders can move, tilt, and twist autonomously.
    ///
    /// **Torso Yaw** uses shoulder span compression: when you turn your body, the 2D distance between
    /// left and right shoulders shrinks proportionally to cos(θ). The nose-to-shoulder-midpoint horizontal
    /// offset determines the turn direction. This produces a strong, observable signal from any camera angle.
    public static func computeTorsoRotation(
        body: TrackedBodyMetrics,
        posture: ClinicalPostureMetrics = ClinicalPostureMetrics(),
        facingSign: Float = 1.0,
        baselineShoulderSpan: Float? = nil
    ) -> SIMD3<Float> {
        // 1. Torso Roll (Lateral tilt from bilateral shoulder slope - independent of head)
        let torsoRollRad: Float
        if let shL = body.leftShoulder, let shR = body.rightShoulder,
           shL.confidence > 0.20, shR.confidence > 0.20 {
            let dx = Float(shR.point.x - shL.point.x)
            let dy = Float(shR.point.y - shL.point.y)
            torsoRollRad = atan2(dy, max(0.04, abs(dx)))
        } else {
            torsoRollRad = 0.0
        }
        
        // 2. Torso Pitch (Sagittal forward flexion / slouching)
        let forwardSlouchRatio = max(0.0, Float(posture.forwardDriftCm) - 0.8)
        let torsoPitchRad = min(0.35, forwardSlouchRatio * 0.035) // up to ~15° forward flexion
        
        // 3. Torso Yaw (Axial trunk rotation from shoulder span compression - independent of head)
        //
        // When facing the camera: shoulder span = baseline (maximum).
        // When body rotates θ degrees: span ≈ baseline × cos(θ).
        // So: θ = acos(currentSpan / baselineSpan).
        // Direction comes from nose offset relative to shoulder midpoint.
        let torsoYawRad: Float
        if let shL = body.leftShoulder, let shR = body.rightShoulder,
           shL.confidence > 0.20, shR.confidence > 0.20,
           let baseline = baselineShoulderSpan, baseline > 0.04 {
            let currentSpan = abs(Float(shR.point.x - shL.point.x))
            let spanRatio = min(1.0, currentSpan / baseline)
            
            // Magnitude: acos(spanRatio) gives rotation angle in radians
            // Apply a deadband: ignore span changes < 3% to prevent jitter
            let yawMagnitude: Float
            if spanRatio > 0.97 {
                yawMagnitude = 0.0
            } else {
                yawMagnitude = acos(max(0.34, spanRatio)) // cap at ~70°
            }
            
            // Direction: nose offset from shoulder midpoint
            let shoulderMidX = Float(shL.point.x + shR.point.x) * 0.5
            let directionSign: Float
            if let nose = body.nose, nose.confidence > 0.20 {
                let noseOffset = Float(nose.point.x) - shoulderMidX
                directionSign = noseOffset >= 0 ? 1.0 : -1.0
            } else if let leftEye = body.leftEye, let rightEye = body.rightEye,
                      leftEye.confidence > 0.20, rightEye.confidence > 0.20 {
                let eyeMidX = Float(leftEye.point.x + rightEye.point.x) * 0.5
                let eyeOffset = eyeMidX - shoulderMidX
                directionSign = eyeOffset >= 0 ? 1.0 : -1.0
            } else {
                directionSign = facingSign
            }
            
            torsoYawRad = directionSign * facingSign * min(0.87, yawMagnitude) // cap at ~50°
        } else {
            torsoYawRad = 0.0
        }
        
        return SIMD3<Float>(torsoPitchRad, torsoYawRad, torsoRollRad)
    }
    
    /// Computes dynamic 3D translations for skull, C7 neck, T12 thoracolumbar junction, and lumbar base
    public static func computeDynamicPositions(
        head: CGPoint?,
        neck: CGPoint?,
        root: CGPoint? = nil,
        posture: ClinicalPostureMetrics,
        torsoRoll: Float = 0.0,
        cameraElevation: CameraElevationMode = .eyeLevel
    ) -> (skullPos: SIMD3<Float>, c7Pos: SIMD3<Float>, t12Pos: SIMD3<Float>, basePos: SIMD3<Float>) {
        guard let head = head, neck != nil else {
            return (defaultSkullPosition, defaultC7Position, defaultT12Position, defaultLumbarBasePosition)
        }
        
        // Lateral X translation: map normalized [0, 1] to [-0.25m, +0.25m]
        let lateralX = Float(head.x - 0.5) * 0.5
        
        // Elevation pitch offset compensation for 3D anchor positioning
        let elevOffset = Float(cameraElevation.pitchDegrees * .pi / 180.0) * 0.06
        
        // Vertical Y translation: track elevation changes
        let verticalY = Float(head.y - 0.75) * 0.4 + elevOffset
        
        // Sagittal Z translation: forward head drift pushes skull forward in Z!
        let forwardZ = Float(posture.forwardDriftCm * 0.01) * 1.5 // 1cm drift = 0.015m in 3D
        
        let skullPos = SIMD3<Float>(
            defaultSkullPosition.x + lateralX,
            defaultSkullPosition.y + verticalY,
            defaultSkullPosition.z + forwardZ
        )
        
        let c7Pos = SIMD3<Float>(
            defaultC7Position.x + lateralX * 0.8,
            defaultC7Position.y + verticalY * 0.8,
            defaultC7Position.z + forwardZ * 0.2 // C7 stays relatively anchored
        )
        
        // T12 junction and Lumbar Base follow torso movement and lateral lean
        let hipLateralX: Float
        if let root = root {
            hipLateralX = Float(root.x - 0.5) * 0.4
        } else {
            hipLateralX = lateralX * 0.35 + sin(torsoRoll) * 0.08
        }
        
        let t12Pos = SIMD3<Float>(
            defaultT12Position.x + lateralX * 0.55 + hipLateralX * 0.3,
            c7Pos.y - 0.27,
            defaultT12Position.z + forwardZ * 0.08
        )
        
        let basePos = SIMD3<Float>(
            defaultLumbarBasePosition.x + hipLateralX,
            t12Pos.y - 0.15,
            defaultLumbarBasePosition.z
        )
        
        return (skullPos, c7Pos, t12Pos, basePos)
    }
    
    private static func calculateAngleBetweenVectors(_ v1: CGVector, _ v2: CGVector) -> Double {
        let dot = v1.dx * v2.dx + v1.dy * v2.dy
        let mag1 = sqrt(v1.dx * v1.dx + v1.dy * v1.dy)
        let mag2 = sqrt(v2.dx * v2.dx + v2.dy * v2.dy)
        guard mag1 > 0 && mag2 > 0 else { return 180.0 }
        let cosVal = max(-1.0, min(1.0, dot / (mag1 * mag2)))
        return Double(acos(cosVal) * 180.0 / .pi)
    }
    
    // MARK: - Mouth & TMJ Mandibular Articulation
    
    /// Computes the normalized mouth opening fraction (0.0 = closed, 1.0 = wide open)
    /// using the Euclidean distance between upper and lower lips relative to face scale.
    public static func computeMouthAperture(
        upperLip: CGPoint,
        lowerLip: CGPoint,
        chin: CGPoint,
        faceBoundingBoxHeight: CGFloat
    ) -> Float {
        let lipDistance = hypot(upperLip.x - lowerLip.x, upperLip.y - lowerLip.y)
        let referenceScale = max(0.08, faceBoundingBoxHeight)
        let rawRatio = Float(lipDistance / referenceScale)
        
        let normalized = (rawRatio - restingApertureThreshold) / (fullApertureThreshold - restingApertureThreshold)
        return max(0.0, min(1.0, normalized))
    }
    
    /// Computes the TMJ jaw rotation angle in degrees ($0^\circ \to 24^\circ$)
    /// based on the mouth aperture fraction.
    public static func computeTMJRotationDegrees(apertureFraction: Float) -> Float {
        let clamped = max(0.0, min(1.0, apertureFraction))
        return clamped * maxTMJRotationDegrees
    }
    
    // MARK: - Dynamic Visibility Culling
    
    /// Determines which discrete bone groups can be analyzed based on camera framing and landmark confidence.
    /// If only the upper torso/face is in the camera view, lower spinal vertebrae and hips are culled.
    public static func resolveVisibilityFlags(
        face: TrackedFaceMetrics,
        body: TrackedBodyMetrics
    ) -> BoneVisibilityFlags {
        var flags = BoneVisibilityFlags()
        
        // 1. Skull & Jaw: visible if face is tracked with sufficient confidence
        if face.confidence > 0.25 {
            flags.isSkullVisible = true
            flags.isJawVisible = true
        }
        
        // 2. Cervical Spine & Neck: visible if either face or neck landmark is tracked
        let hasNeck = (body.neckC7?.confidence ?? 0.0) > 0.25
        let hasShoulder = ((body.leftShoulder?.confidence ?? 0.0) > 0.25) || ((body.rightShoulder?.confidence ?? 0.0) > 0.25)
        
        if (flags.isSkullVisible && hasShoulder) || hasNeck {
            flags.isCervicalSpineVisible = true
        }
        
        // 3. Clavicles & Thoracic spine: visible if shoulder girdle is in view
        // Note: Sternum & ribcage (chest bone) removed from 3D skeleton display per user requirement
        if hasShoulder {
            flags.isClaviclesVisible = true
            flags.isSternumVisible = false
            flags.isRibcageVisible = false
            flags.isThoracicSpineVisible = true
        }
        
        // 4. Lumbar Spine & Pelvis:
        // CULLING RULE: Only display if hips or lower torso are detected and NOT clipped at the bottom of the camera frame
        let leftHipConf = body.leftHip?.confidence ?? 0.0
        let rightHipConf = body.rightHip?.confidence ?? 0.0
        let maxHipConf = max(leftHipConf, rightHipConf)
        
        // In Vision coordinates, y=0.0 is the bottom of the frame.
        // If hip y is below 0.05, the joint is effectively truncated/out-of-frame.
        let leftHipY = body.leftHip?.point.y ?? 0.0
        let rightHipY = body.rightHip?.point.y ?? 0.0
        let isHipInFrame = (leftHipY > 0.06 || rightHipY > 0.06)
        
        if maxHipConf > 0.30 && isHipInFrame {
            flags.isLumbarSpineVisible = true
            flags.isPelvisVisible = true
        } else {
            // Lower spinal bone not displayed when only face and chest are framed
            flags.isLumbarSpineVisible = false
            flags.isPelvisVisible = false
        }
        
        return flags
    }
    
    // MARK: - Discrete Vertebrae Generation
    
    /// Generates the 7 discrete cervical vertebrae (C1 Atlas through C7 Prominens)
    /// distributed along a spline from the skull base to C7, smoothly interpolating head rotation down to torso rotation.
    public static func computeCervicalVertebrae(
        skullPos: SIMD3<Float>,
        c7Pos: SIMD3<Float>,
        headRotation: SIMD3<Float>,
        torsoRotation: SIMD3<Float> = .zero,
        isVisible: Bool
    ) -> [VertebraPose] {
        guard isVisible else {
            return (1...7).map {
                VertebraPose(name: "C\($0)", segmentIndex: $0, isVisible: false)
            }
        }
        
        let cNames = ["C1", "C2", "C3", "C4", "C5", "C6", "C7"]
        var vertebrae: [VertebraPose] = []
        
        for i in 0..<7 {
            let t = Float(i) / 6.0 // 0.0 at C1 (top), 1.0 at C7 (base)
            
            // Linear position interpolation with anatomical cervical lordosis curve (slight forward arch)
            let lordosisSagittalOffset = sin(t * .pi) * 0.015 // Forward arch in Z
            let x = skullPos.x + (c7Pos.x - skullPos.x) * t
            let y = skullPos.y + (c7Pos.y - skullPos.y) * t
            let z = skullPos.z + (c7Pos.z - skullPos.z) * t + lordosisSagittalOffset
            
            // Proportionally distribute head rotation down to torso rotation
            let rot = headRotation * (1.0 - t * 0.85) + torsoRotation * (t * 0.85)
            
            vertebrae.append(
                VertebraPose(
                    name: cNames[i],
                    segmentIndex: i + 1,
                    position: SIMD3<Float>(x, y, z),
                    rotation: rot,
                    isVisible: true
                )
            )
        }
        
        return vertebrae
    }
    
    /// Generates the 12 discrete thoracic vertebrae (T1 through T12)
    /// spanning from C7 down to the thoracolumbar junction, with kyphotic posterior curvature
    /// and 3D rotational response to torso twist, forward pitch slouch, and lateral tilt.
    public static func computeThoracicVertebrae(
        c7Pos: SIMD3<Float>,
        t12Pos: SIMD3<Float>,
        lateralTilt: Float = 0.0,
        torsoRotation: SIMD3<Float> = .zero,
        isVisible: Bool
    ) -> [VertebraPose] {
        guard isVisible else {
            return (1...12).map {
                VertebraPose(name: "T\($0)", segmentIndex: $0, isVisible: false)
            }
        }
        
        var vertebrae: [VertebraPose] = []
        let effectiveRoll = (torsoRotation.z != 0.0) ? torsoRotation.z : lateralTilt
        
        for i in 0..<12 {
            let t = Float(i) / 11.0
            
            // Anatomical thoracic kyphosis curve (posterior curve in Z)
            let kyphosisOffset = -sin(t * .pi) * 0.022
            let x = c7Pos.x + (t12Pos.x - c7Pos.x) * t + effectiveRoll * sin(t * .pi) * 0.03
            let y = c7Pos.y + (t12Pos.y - c7Pos.y) * t
            let z = c7Pos.z + (t12Pos.z - c7Pos.z) * t + kyphosisOffset
            
            let rot = SIMD3<Float>(
                torsoRotation.x * sin(t * .pi) * 1.1,
                torsoRotation.y * (1.0 - t * 0.35),
                effectiveRoll * (1.0 - t * 0.5)
            )
            
            vertebrae.append(
                VertebraPose(
                    name: "T\(i + 1)",
                    segmentIndex: i + 1,
                    position: SIMD3<Float>(x, y, z),
                    rotation: rot,
                    isVisible: true
                )
            )
        }
        
        return vertebrae
    }
    
    /// Generates the 5 discrete lumbar vertebrae (L1 through L5)
    /// extending from T12 to the sacral base, with lumbar lordosis and rotational twist/tilt.
    public static func computeLumbarVertebrae(
        t12Pos: SIMD3<Float>,
        basePos: SIMD3<Float>,
        torsoRotation: SIMD3<Float> = .zero,
        isVisible: Bool
    ) -> [VertebraPose] {
        guard isVisible else {
            return (1...5).map {
                VertebraPose(name: "L\($0)", segmentIndex: $0, isVisible: false)
            }
        }
        
        var vertebrae: [VertebraPose] = []
        
        for i in 0..<5 {
            let t = Float(i) / 4.0
            
            // Lumbar lordosis anterior curve
            let lordosisOffset = sin(t * .pi) * 0.016
            let x = t12Pos.x + (basePos.x - t12Pos.x) * t
            let y = t12Pos.y + (basePos.y - t12Pos.y) * t
            let z = t12Pos.z + (basePos.z - t12Pos.z) * t + lordosisOffset
            
            let rot = SIMD3<Float>(
                torsoRotation.x * 0.45 * (1.0 - t * 0.5),
                torsoRotation.y * 0.65 * (1.0 - t * 0.6),
                torsoRotation.z * 0.55 * (1.0 - t * 0.5)
            )
            
            vertebrae.append(
                VertebraPose(
                    name: "L\(i + 1)",
                    segmentIndex: i + 1,
                    position: SIMD3<Float>(x, y, z),
                    rotation: rot,
                    isVisible: true
                )
            )
        }
        
        return vertebrae
    }
}
