import Foundation
import CoreGraphics
import simd

/// Camera viewing angle mode for perspective foreshortening correction
public enum CameraAngleMode: String, CaseIterable, Identifiable, Sendable {
    case oblique30 = "30° Oblique"
    case diagonal45 = "45° Diagonal"
    case oblique60 = "60° Oblique"
    case oblique75 = "75° Oblique"
    case profile90 = "90° Profile"
    
    public var id: String { rawValue }
    
    public var degrees: Double {
        switch self {
        case .oblique30: return 30.0
        case .diagonal45: return 45.0
        case .oblique60: return 60.0
        case .oblique75: return 75.0
        case .profile90: return 90.0
        }
    }
    
    /// sin(alpha) factor for anterior-posterior foreshortening compensation
    public var sinAlpha: Double {
        sin(degrees * .pi / 180.0)
    }
    
    /// Inverse sine projection factor (1 / sin θ) to recover true physical sagittal distance from camera sensor
    public var projectionFactor: Double {
        1.0 / sinAlpha
    }
}

/// Camera vertical elevation / height mode relative to user's head for pitch perspective compensation
public enum CameraElevationMode: String, CaseIterable, Identifiable, Sendable {
    case auto = "Auto Height"
    case highAboveMonitor = "High (Above Screen)"   // ~+18° downward angle (e.g. monitor mount, iMac)
    case eyeLevel = "Eye Level"                     // 0° neutral
    case lowDeskLaptop = "Low (Desk / Laptop)"      // ~-18° upward angle (e.g. MacBook on desk)
    
    public var id: String { rawValue }
    
    /// Nominal elevation pitch angle in degrees (positive = camera looking down; negative = camera looking up)
    public var pitchDegrees: Double {
        switch self {
        case .auto: return 0.0 // Resolved dynamically from tracked head pitch
        case .highAboveMonitor: return 18.0
        case .eyeLevel: return 0.0
        case .lowDeskLaptop: return -18.0
        }
    }
}

/// Framing view mode for camera display: Open View (Aspect Fit full 16:9 uncropped) vs Fill View (Aspect Fill pane-filling cropped)
public enum CameraViewMode: String, CaseIterable, Identifiable, Sendable {
    case openView = "Open View (16:9)"
    case fillView = "Fill Frame"
    
    public var id: String { rawValue }
}

/// Computes the exact rendering rectangle of a video frame within a container size
public func computeVideoRect(in viewSize: CGSize, mode: CameraViewMode, videoAspect: CGFloat = 16.0 / 9.0) -> CGRect {
    guard viewSize.width > 0, viewSize.height > 0 else { return .zero }
    let viewAspect = viewSize.width / viewSize.height
    
    switch mode {
    case .openView:
        // Aspect Fit: completely uncropped, letterbox or pillarbox to fit view
        if viewAspect > videoAspect {
            let height = viewSize.height
            let width = height * videoAspect
            let x = (viewSize.width - width) * 0.5
            return CGRect(x: x, y: 0, width: width, height: height)
        } else {
            let width = viewSize.width
            let height = width / videoAspect
            let y = (viewSize.height - height) * 0.5
            return CGRect(x: 0, y: y, width: width, height: height)
        }
        
    case .fillView:
        // Aspect Fill: fill entire view, cropping overflowing dimensions
        if viewAspect > videoAspect {
            let width = viewSize.width
            let height = width / videoAspect
            let y = (viewSize.height - height) * 0.5
            return CGRect(x: 0, y: y, width: width, height: height)
        } else {
            let height = viewSize.height
            let width = height * videoAspect
            let x = (viewSize.width - width) * 0.5
            return CGRect(x: x, y: 0, width: width, height: height)
        }
    }
}

/// Converts a normalized Vision point [0, 1] x [0, 1] into view coordinate space based on video display rect
public func convertVisionPointToView(_ pt: CGPoint, in videoRect: CGRect) -> CGPoint {
    CGPoint(
        x: videoRect.origin.x + pt.x * videoRect.width,
        y: videoRect.origin.y + (1.0 - pt.y) * videoRect.height
    )
}

/// Clinical posture status classification
public enum PostureStatus: String, Sendable {
    case optimal = "Optimal"
    case caution = "Caution"
    case turtleNeck = "Turtle Neck"
    case slouching = "Slouching"
    
    public var description: String {
        switch self {
        case .optimal: return "Spine and cervical vertebrae in neutral healthy alignment"
        case .caution: return "Mild forward head drift detected, approaching strain threshold"
        case .turtleNeck: return "Excessive forward head translation (high cervical spine load)"
        case .slouching: return "Thoracic kyphosis / spine compressed downward"
        }
    }
}

/// 4 Key Anatomical Landmarks for Clinical Posture Assessment (T, A, J, C7)
public struct PosturalKeypoints2D: Equatable, Sendable {
    public var tragus: CGPoint?        // T: Tragus of the ear (Tragion)
    public var acromion: CGPoint?      // A: Acromion tip of shoulder
    public var jugularNotch: CGPoint?  // J: Suprasternal notch (Hõm ức / đáy cổ)
    public var c7: CGPoint?            // C7 vertebra prominens (or C7_est)
    public var isC7Estimated: Bool     // True if C7 was estimated geometrically from J and A
    
    public init(
        tragus: CGPoint? = nil,
        acromion: CGPoint? = nil,
        jugularNotch: CGPoint? = nil,
        c7: CGPoint? = nil,
        isC7Estimated: Bool = false
    ) {
        self.tragus = tragus
        self.acromion = acromion
        self.jugularNotch = jugularNotch
        self.c7 = c7
        self.isC7Estimated = isC7Estimated
    }
}

/// Clinical biomechanical posture metrics (CVA, ATJ, SHJ, forward head drift, and guidance)
public struct ClinicalPostureMetrics: Equatable, Sendable {
    public var cvaDegrees: Double          // Craniovertebral Angle (Normal: >60°, Caution: 55°–60°, Turtle Neck: <55°)
    public var forwardDriftCm: Double       // Horizontal forward drift of ear in front of shoulder (cm)
    public var atjDegrees: Double          // ATJ Angle (Acromion - Tragus - Jugular Notch)
    public var shjRatio: Double            // SHJ Ratio (d_AT / d_AJ)
    public var shjDegrees: Double          // SHJ Angle (angle between AT and AJ)
    public var thoracoCervicalAngle: Double // Angle between head, C7, and upper chest
    public var status: PostureStatus
    public var guidanceCue: String
    
    // Relative CVA Proxy fields (production relative forward-head displacement model)
    public var deltaCvaDegrees: Double     // ΔCVA_t = CVA_rel - CVA_0 (degrees)
    public var isRelativeProxy: Bool       // True when using calibrated forward-head proxy
    public var forwardDisplacementD: Double // Normalized forward displacement D_t
    public var isDirectAngleFacing: Bool   // True when user faces camera directly (|yaw| < 12°), where sagittal CVA is foreshortened
    
    public init(
        cvaDegrees: Double = 63.0,
        forwardDriftCm: Double = 0.0,
        atjDegrees: Double = 16.0,
        shjRatio: Double = 0.40,
        shjDegrees: Double = 22.0,
        thoracoCervicalAngle: Double = 165.0,
        status: PostureStatus = .optimal,
        guidanceCue: String = "Spine in neutral alignment",
        deltaCvaDegrees: Double = 0.0,
        isRelativeProxy: Bool = false,
        forwardDisplacementD: Double = 0.0,
        isDirectAngleFacing: Bool = false
    ) {
        self.cvaDegrees = cvaDegrees
        self.forwardDriftCm = forwardDriftCm
        self.atjDegrees = atjDegrees
        self.shjRatio = shjRatio
        self.shjDegrees = shjDegrees
        self.thoracoCervicalAngle = thoracoCervicalAngle
        self.status = status
        self.guidanceCue = guidanceCue
        self.deltaCvaDegrees = deltaCvaDegrees
        self.isRelativeProxy = isRelativeProxy
        self.forwardDisplacementD = forwardDisplacementD
        self.isDirectAngleFacing = isDirectAngleFacing
    }
}

/// Visibility state of each discrete anatomical bone grouping.
/// Lower spinal and pelvic bones are dynamically culled if not detected in the camera frame.
public struct BoneVisibilityFlags: Equatable, Sendable {
    public var isSkullVisible: Bool
    public var isJawVisible: Bool
    public var isCervicalSpineVisible: Bool
    public var isClaviclesVisible: Bool
    public var isSternumVisible: Bool
    public var isRibcageVisible: Bool
    public var isThoracicSpineVisible: Bool
    public var isLumbarSpineVisible: Bool
    public var isPelvisVisible: Bool
    
    public init(
        isSkullVisible: Bool = false,
        isJawVisible: Bool = false,
        isCervicalSpineVisible: Bool = false,
        isClaviclesVisible: Bool = false,
        isSternumVisible: Bool = false,
        isRibcageVisible: Bool = false,
        isThoracicSpineVisible: Bool = false,
        isLumbarSpineVisible: Bool = false,
        isPelvisVisible: Bool = false
    ) {
        self.isSkullVisible = isSkullVisible
        self.isJawVisible = isJawVisible
        self.isCervicalSpineVisible = isCervicalSpineVisible
        self.isClaviclesVisible = isClaviclesVisible
        self.isSternumVisible = isSternumVisible
        self.isRibcageVisible = isRibcageVisible
        self.isThoracicSpineVisible = isThoracicSpineVisible
        self.isLumbarSpineVisible = isLumbarSpineVisible
        self.isPelvisVisible = isPelvisVisible
    }
    
    /// Returns the count of currently visible bone categories
    public var visibleCount: Int {
        var count = 0
        if isSkullVisible { count += 1 }
        if isJawVisible { count += 1 }
        if isCervicalSpineVisible { count += 7 } // 7 vertebrae
        if isClaviclesVisible { count += 2 }
        if isSternumVisible { count += 1 }
        if isRibcageVisible { count += 20 } // 10 pairs
        if isThoracicSpineVisible { count += 12 } // 12 vertebrae
        if isLumbarSpineVisible { count += 5 } // 5 vertebrae
        if isPelvisVisible { count += 1 }
        return count
    }
}

/// Tracked facial kinematics including mouth aperture and 3D head rotation.
public struct TrackedFaceMetrics: Equatable, Sendable {
    public var mouthApertureFraction: Float // 0.0 (closed) to 1.0 (wide open)
    public var tmjRotationDegrees: Float    // 0.0° to 24.0° mandibular rotation around TMJ
    public var pitchDegrees: Float          // Head nod up/down
    public var yawDegrees: Float            // Sensor / camera-relative head turn left/right
    public var anatomicalYawDegrees: Float  // Torso / screen-relative anatomical head turn left/right
    public var rollDegrees: Float           // Head tilt ear to shoulder
    public var jawlinePoints: [CGPoint]     // 2D normalized jawline contour
    public var upperLipPoint: CGPoint
    public var lowerLipPoint: CGPoint
    public var chinGnathionPoint: CGPoint
    public var leftTragion: CGPoint?
    public var rightTragion: CGPoint?
    public var confidence: Float
    
    public init(
        mouthApertureFraction: Float = 0.0,
        tmjRotationDegrees: Float = 0.0,
        pitchDegrees: Float = 0.0,
        yawDegrees: Float = 0.0,
        anatomicalYawDegrees: Float = 0.0,
        rollDegrees: Float = 0.0,
        jawlinePoints: [CGPoint] = [],
        upperLipPoint: CGPoint = .zero,
        lowerLipPoint: CGPoint = .zero,
        chinGnathionPoint: CGPoint = .zero,
        leftTragion: CGPoint? = nil,
        rightTragion: CGPoint? = nil,
        confidence: Float = 0.0
    ) {
        self.mouthApertureFraction = mouthApertureFraction
        self.tmjRotationDegrees = tmjRotationDegrees
        self.pitchDegrees = pitchDegrees
        self.yawDegrees = yawDegrees
        self.anatomicalYawDegrees = anatomicalYawDegrees
        self.rollDegrees = rollDegrees
        self.jawlinePoints = jawlinePoints
        self.upperLipPoint = upperLipPoint
        self.lowerLipPoint = lowerLipPoint
        self.chinGnathionPoint = chinGnathionPoint
        self.leftTragion = leftTragion
        self.rightTragion = rightTragion
        self.confidence = confidence
    }
}

/// 2D normalized body landmark with confidence score.
public struct TrackedJoint: Equatable, Sendable {
    public var point: CGPoint
    public var confidence: Float
    
    public init(point: CGPoint, confidence: Float) {
        self.point = point
        self.confidence = confidence
    }
}

/// Tracked body kinematics and landmarks.
public struct TrackedBodyMetrics: Equatable, Sendable {
    public var nose: TrackedJoint?
    public var leftEye: TrackedJoint?
    public var rightEye: TrackedJoint?
    public var leftEar: TrackedJoint?
    public var rightEar: TrackedJoint?
    public var neckC7: TrackedJoint?
    public var leftShoulder: TrackedJoint?
    public var rightShoulder: TrackedJoint?
    public var root: TrackedJoint?
    public var leftHip: TrackedJoint?
    public var rightHip: TrackedJoint?
    
    public init(
        nose: TrackedJoint? = nil,
        leftEye: TrackedJoint? = nil,
        rightEye: TrackedJoint? = nil,
        leftEar: TrackedJoint? = nil,
        rightEar: TrackedJoint? = nil,
        neckC7: TrackedJoint? = nil,
        leftShoulder: TrackedJoint? = nil,
        rightShoulder: TrackedJoint? = nil,
        root: TrackedJoint? = nil,
        leftHip: TrackedJoint? = nil,
        rightHip: TrackedJoint? = nil
    ) {
        self.nose = nose
        self.leftEye = leftEye
        self.rightEye = rightEye
        self.leftEar = leftEar
        self.rightEar = rightEar
        self.neckC7 = neckC7
        self.leftShoulder = leftShoulder
        self.rightShoulder = rightShoulder
        self.root = root
        self.leftHip = leftHip
        self.rightHip = rightHip
    }
}

/// A discrete vertebra in the 3D spinal column.
public struct VertebraPose: Identifiable, Equatable, Sendable {
    public var id: String { name }
    public let name: String
    public let segmentIndex: Int
    public var position: SIMD3<Float>
    public var rotation: SIMD3<Float> // Euler angles (pitch, yaw, roll in radians)
    public var isVisible: Bool
    
    public init(
        name: String,
        segmentIndex: Int,
        position: SIMD3<Float> = .zero,
        rotation: SIMD3<Float> = .zero,
        isVisible: Bool = true
    ) {
        self.name = name
        self.segmentIndex = segmentIndex
        self.position = position
        self.rotation = rotation
        self.isVisible = isVisible
    }
}

/// Aggregated kinematics state of the skeleton at a single time instant.
public struct SkeletalKinematicState: Equatable, Sendable {
    public var timestamp: TimeInterval
    public var face: TrackedFaceMetrics
    public var body: TrackedBodyMetrics
    public var visibility: BoneVisibilityFlags
    
    // Cervical vertebrae C1–C7
    public var cervicalVertebrae: [VertebraPose]
    
    // Thoracic vertebrae T1–T12
    public var thoracicVertebrae: [VertebraPose]
    
    // Lumbar vertebrae L1–L5
    public var lumbarVertebrae: [VertebraPose]
    
    // Shoulder and sternum 3D points
    public var leftClaviclePosition: SIMD3<Float>
    public var rightClaviclePosition: SIMD3<Float>
    public var sternumPosition: SIMD3<Float>
    
    public var keypoints: PosturalKeypoints2D
    public var posture: ClinicalPostureMetrics
    public var cameraAngle: CameraAngleMode
    public var cameraElevation: CameraElevationMode
    public var dynamicSkullPosition: SIMD3<Float>
    public var dynamicC7Position: SIMD3<Float>
    public var dynamicT12Position: SIMD3<Float>
    public var dynamicLumbarBasePosition: SIMD3<Float>
    
    // Torso / Shoulder 3D rotation (Euler angles: pitch, yaw, roll in radians)
    public var torsoRotation: SIMD3<Float>
    
    public var facingSign: Float
    
    /// Normalized 2D contour points of the detected body silhouette outline.
    public var silhouetteContour: [CGPoint]
    
    public var torsoPitchDegrees: Float { torsoRotation.x * 180.0 / .pi }
    public var torsoYawDegrees: Float { torsoRotation.y * 180.0 / .pi }
    public var torsoRollDegrees: Float { torsoRotation.z * 180.0 / .pi }
    
    /// Perspective-corrected 2D C7 neck point.
    public var perspectiveNeckPoint: CGPoint {
        body.neckC7?.point ?? CGPoint(x: 0.5, y: 0.6)
    }
    
    public init(
        timestamp: TimeInterval = 0.0,
        face: TrackedFaceMetrics = TrackedFaceMetrics(),
        body: TrackedBodyMetrics = TrackedBodyMetrics(),
        keypoints: PosturalKeypoints2D = PosturalKeypoints2D(),
        visibility: BoneVisibilityFlags = BoneVisibilityFlags(),
        posture: ClinicalPostureMetrics = ClinicalPostureMetrics(),
        cameraAngle: CameraAngleMode = .diagonal45,
        cameraElevation: CameraElevationMode = .eyeLevel,
        cervicalVertebrae: [VertebraPose] = [],
        thoracicVertebrae: [VertebraPose] = [],
        lumbarVertebrae: [VertebraPose] = [],
        leftClaviclePosition: SIMD3<Float> = .zero,
        rightClaviclePosition: SIMD3<Float> = .zero,
        sternumPosition: SIMD3<Float> = .zero,
        dynamicSkullPosition: SIMD3<Float> = SIMD3<Float>(0.0, 0.77, 0.0),
        dynamicC7Position: SIMD3<Float> = SIMD3<Float>(0.0, 0.60, -0.04),
        dynamicT12Position: SIMD3<Float> = SIMD3<Float>(0.0, 0.33, 0.0),
        dynamicLumbarBasePosition: SIMD3<Float> = SIMD3<Float>(0.0, 0.18, 0.0),
        torsoRotation: SIMD3<Float> = .zero,
        facingSign: Float = 1.0,
        silhouetteContour: [CGPoint] = []
    ) {
        self.timestamp = timestamp
        self.face = face
        self.body = body
        self.keypoints = keypoints
        self.visibility = visibility
        self.posture = posture
        self.cameraAngle = cameraAngle
        self.cameraElevation = cameraElevation
        self.cervicalVertebrae = cervicalVertebrae
        self.thoracicVertebrae = thoracicVertebrae
        self.lumbarVertebrae = lumbarVertebrae
        self.leftClaviclePosition = leftClaviclePosition
        self.rightClaviclePosition = rightClaviclePosition
        self.sternumPosition = sternumPosition
        self.dynamicSkullPosition = dynamicSkullPosition
        self.dynamicC7Position = dynamicC7Position
        self.dynamicT12Position = dynamicT12Position
        self.dynamicLumbarBasePosition = dynamicLumbarBasePosition
        self.torsoRotation = torsoRotation
        self.facingSign = facingSign
        self.silhouetteContour = silhouetteContour
    }
}
