import XCTest
import CoreGraphics
import simd
@testable import Posturify

final class KinematicsTests: XCTestCase {
    
    func testMouthApertureAndTMJRotation() {
        // 1. Closed mouth
        let closedUpper = CGPoint(x: 0.5, y: 0.500)
        let closedLower = CGPoint(x: 0.5, y: 0.495) // Touching resting lips (gap = 0.005)
        let chin = CGPoint(x: 0.5, y: 0.40)
        
        let closedAperture = KinematicsCalculator.computeMouthAperture(
            upperLip: closedUpper,
            lowerLip: closedLower,
            chin: chin,
            faceBoundingBoxHeight: 0.3
        )
        let closedTMJ = KinematicsCalculator.computeTMJRotationDegrees(apertureFraction: closedAperture)
        XCTAssertEqual(closedAperture, 0.0, accuracy: 0.01)
        XCTAssertEqual(closedTMJ, 0.0, accuracy: 0.1)
        
        // 2. Wide open mouth
        let openUpper = CGPoint(x: 0.5, y: 0.55)
        let openLower = CGPoint(x: 0.5, y: 0.42) // Large gap
        let openAperture = KinematicsCalculator.computeMouthAperture(
            upperLip: openUpper,
            lowerLip: openLower,
            chin: chin,
            faceBoundingBoxHeight: 0.3
        )
        let openTMJ = KinematicsCalculator.computeTMJRotationDegrees(apertureFraction: openAperture)
        XCTAssertGreaterThan(openAperture, 0.8)
        XCTAssertGreaterThan(openTMJ, 20.0)
        XCTAssertLessThanOrEqual(openTMJ, 24.0)
    }
    
    func testDynamicVisibilityCullingWhenLowerBodyOutOfView() {
        // Face and upper chest visible, but hips are NOT detected
        let face = TrackedFaceMetrics(
            mouthApertureFraction: 0.2,
            confidence: 0.95
        )
        let bodyNoHips = TrackedBodyMetrics(
            neckC7: TrackedJoint(point: CGPoint(x: 0.5, y: 0.6), confidence: 0.9),
            leftShoulder: TrackedJoint(point: CGPoint(x: 0.35, y: 0.55), confidence: 0.9),
            rightShoulder: TrackedJoint(point: CGPoint(x: 0.65, y: 0.55), confidence: 0.9),
            leftHip: nil,
            rightHip: nil
        )
        
        let flags = KinematicsCalculator.resolveVisibilityFlags(face: face, body: bodyNoHips)
        
        // Upper bones must be visible
        XCTAssertTrue(flags.isSkullVisible, "Skull should be visible when face is tracked")
        XCTAssertTrue(flags.isJawVisible, "Jaw should be visible when face is tracked")
        XCTAssertTrue(flags.isCervicalSpineVisible, "Cervical spine should be visible")
        XCTAssertTrue(flags.isThoracicSpineVisible, "Thoracic spine should be visible")
        XCTAssertTrue(flags.isClaviclesVisible, "Clavicles should be visible")
        XCTAssertFalse(flags.isRibcageVisible, "Ribcage (chest bone) should be hidden from 3D skeleton display")
        XCTAssertFalse(flags.isSternumVisible, "Sternum (chest bone) should be hidden from 3D skeleton display")
        
        // Lower spine must be CULLED
        XCTAssertFalse(flags.isLumbarSpineVisible, "Lower part of spinal bone not need to be displayed when only face and chest are framed")
        XCTAssertFalse(flags.isPelvisVisible, "Pelvis should be culled")
    }
    
    func testDynamicVisibilityRevealsLowerSpineWhenHipsInFrame() {
        let face = TrackedFaceMetrics(confidence: 0.95)
        let bodyWithHips = TrackedBodyMetrics(
            neckC7: TrackedJoint(point: CGPoint(x: 0.5, y: 0.6), confidence: 0.9),
            leftShoulder: TrackedJoint(point: CGPoint(x: 0.35, y: 0.55), confidence: 0.9),
            rightShoulder: TrackedJoint(point: CGPoint(x: 0.65, y: 0.55), confidence: 0.9),
            leftHip: TrackedJoint(point: CGPoint(x: 0.42, y: 0.18), confidence: 0.85),
            rightHip: TrackedJoint(point: CGPoint(x: 0.58, y: 0.18), confidence: 0.85)
        )
        
        let flags = KinematicsCalculator.resolveVisibilityFlags(face: face, body: bodyWithHips)
        XCTAssertTrue(flags.isLumbarSpineVisible, "Lumbar spine should be displayed when hips are in frame")
        XCTAssertTrue(flags.isPelvisVisible, "Pelvis should be displayed when hips are in frame")
    }
    
    func testDiscreteVertebraeCountsAndSegmentIdentities() {
        // Cervical spine has exactly 7 vertebrae: C1 to C7
        let cervical = KinematicsCalculator.computeCervicalVertebrae(
            skullPos: SIMD3<Float>(0, 0.77, 0),
            c7Pos: SIMD3<Float>(0, 0.60, 0),
            headRotation: .zero,
            isVisible: true
        )
        XCTAssertEqual(cervical.count, 7)
        XCTAssertEqual(cervical.first?.name, "C1")
        XCTAssertEqual(cervical.last?.name, "C7")
        XCTAssertTrue(cervical.allSatisfy { $0.isVisible })
        
        // Thoracic spine has exactly 12 vertebrae: T1 to T12
        let thoracic = KinematicsCalculator.computeThoracicVertebrae(
            c7Pos: SIMD3<Float>(0, 0.60, 0),
            t12Pos: SIMD3<Float>(0, 0.33, 0),
            lateralTilt: 0.0,
            isVisible: true
        )
        XCTAssertEqual(thoracic.count, 12)
        XCTAssertEqual(thoracic.first?.name, "T1")
        XCTAssertEqual(thoracic.last?.name, "T12")
        XCTAssertTrue(thoracic.allSatisfy { $0.isVisible })
        
        // Lumbar spine has exactly 5 vertebrae: L1 to L5
        let lumbarVisible = KinematicsCalculator.computeLumbarVertebrae(
            t12Pos: SIMD3<Float>(0, 0.33, 0),
            basePos: SIMD3<Float>(0, 0.18, 0),
            isVisible: true
        )
        XCTAssertEqual(lumbarVisible.count, 5)
        XCTAssertEqual(lumbarVisible.first?.name, "L1")
        XCTAssertEqual(lumbarVisible.last?.name, "L5")
        XCTAssertTrue(lumbarVisible.allSatisfy { $0.isVisible })
        
        // Culled lumbar spine marks all 5 vertebrae as not visible
        let lumbarCulled = KinematicsCalculator.computeLumbarVertebrae(
            t12Pos: SIMD3<Float>(0, 0.33, 0),
            basePos: SIMD3<Float>(0, 0.18, 0),
            isVisible: false
        )
        XCTAssertEqual(lumbarCulled.count, 5)
        XCTAssertTrue(lumbarCulled.allSatisfy { !$0.isVisible })
    }
    
    func testSyntheticKinematicCycleGeneratesConsistentData() {
        let tracker = VisionTracker()
        
        // Frame at t=1.0s (mouth opening)
        let frame1 = tracker.generateSyntheticKinematicFrame(time: 1.0)
        XCTAssertGreaterThan(frame1.face.mouthApertureFraction, 0.0)
        XCTAssertGreaterThan(frame1.face.tmjRotationDegrees, 0.0)
        XCTAssertTrue(frame1.visibility.isSkullVisible)
        XCTAssertTrue(frame1.visibility.isJawVisible)
        XCTAssertTrue(frame1.visibility.isLumbarSpineVisible)
        
        // Frame at t=8.5s (zoomed in camera view, hips out of frame)
        let frame8 = tracker.generateSyntheticKinematicFrame(time: 8.5)
        XCTAssertFalse(frame8.visibility.isLumbarSpineVisible, "Lumbar spine should be culled in zoomed-in frame")
    }
    
    func testClinicalCVACalculationAndPerspectiveCorrection() {
        // 1. Optimal neutral posture at 90° profile
        let neutralHead = CGPoint(x: 0.50, y: 0.75)
        let neutralNeck = CGPoint(x: 0.50, y: 0.62)
        let neutralShoulder = CGPoint(x: 0.50, y: 0.55)
        
        let optimalMetrics = KinematicsCalculator.computeClinicalPosture(
            head: neutralHead,
            neck: neutralNeck,
            shoulder: neutralShoulder,
            cameraAngle: .profile90
        )
        XCTAssertEqual(optimalMetrics.cvaDegrees, 90.0, accuracy: 1.0)
        XCTAssertEqual(optimalMetrics.forwardDriftCm, 0.0, accuracy: 0.1)
        XCTAssertEqual(optimalMetrics.status, .optimal)
        
        // 2. Turtle Neck at 45° diagonal angle (foreshortened drift on camera sensor)
        // Head translates forward to x=0.565 and lowers to y=0.69 relative to neck at (0.50, 0.62)
        // DeltaY = 0.07, TrueSagittalDeltaX = 0.065 * 1.4142 ≈ 0.0919 -> CVA = atan(0.07 / 0.0919) ≈ 37.3° < 52°!
        let turtleHead = CGPoint(x: 0.565, y: 0.69)
        let turtleNeck = CGPoint(x: 0.50, y: 0.62)
        let turtleShoulder = CGPoint(x: 0.50, y: 0.55)
        
        let turtleMetrics = KinematicsCalculator.computeClinicalPosture(
            head: turtleHead,
            neck: turtleNeck,
            shoulder: turtleShoulder,
            cameraAngle: .diagonal45
        )
        XCTAssertLessThan(turtleMetrics.cvaDegrees, KinematicsCalculator.PostureConstraints.turtleNeckCvaMaxDegrees, "CVA should drop below strict 55° threshold during turtle neck")
        XCTAssertGreaterThan(turtleMetrics.forwardDriftCm, KinematicsCalculator.PostureConstraints.turtleNeckDriftMinCm, "Forward drift should exceed strict 1.8 cm threshold")
        XCTAssertEqual(turtleMetrics.status, .turtleNeck)
        XCTAssertTrue(turtleMetrics.guidanceCue.contains("Turtle Neck"))
        
        // 3. Strict Caution boundary: CVA between 55° and 60° (57.5°), mild drift +1.47 cm
        // DeltaY = 0.08, DeltaX = 0.0360 * 1.4142 ≈ 0.0509 -> CVA = atan(0.08 / 0.0509) ≈ 57.5° (< 60.0°, >= 55.0°)
        let mildDriftHead = CGPoint(x: 0.5360, y: 0.70)
        let mildDriftNeck = CGPoint(x: 0.50, y: 0.62)
        let mildDriftShoulder = CGPoint(x: 0.52, y: 0.55)
        
        let cautionMetrics = KinematicsCalculator.computeClinicalPosture(
            head: mildDriftHead,
            neck: mildDriftNeck,
            shoulder: mildDriftShoulder,
            cameraAngle: .diagonal45
        )
        XCTAssertLessThanOrEqual(cautionMetrics.cvaDegrees, KinematicsCalculator.PostureConstraints.optimalCvaMinDegrees)
        XCTAssertGreaterThanOrEqual(cautionMetrics.cvaDegrees, KinematicsCalculator.PostureConstraints.turtleNeckCvaMaxDegrees)
        XCTAssertEqual(cautionMetrics.status, .caution, "Strict constraint should trigger Caution when CVA is below 60°")
        XCTAssertTrue(cautionMetrics.guidanceCue.contains("Caution"))
        
        // 4. Strict Turtle Neck boundary below 55°: (e.g. 53.5°)
        let below55Head = CGPoint(x: 0.5366, y: 0.69)
        let below55Metrics = KinematicsCalculator.computeClinicalPosture(
            head: below55Head,
            neck: mildDriftNeck,
            shoulder: mildDriftShoulder,
            cameraAngle: .diagonal45
        )
        XCTAssertLessThan(below55Metrics.cvaDegrees, KinematicsCalculator.PostureConstraints.turtleNeckCvaMaxDegrees)
        XCTAssertEqual(below55Metrics.status, .turtleNeck, "CVA < 55° must trigger Turtle Neck under strict criteria")
    }
    
    func testDynamicPositionTranslationWithPosture() {
        let head = CGPoint(x: 0.55, y: 0.76)
        let neck = CGPoint(x: 0.50, y: 0.62)
        let posture = ClinicalPostureMetrics(
            cvaDegrees: 44.0,
            forwardDriftCm: 3.5,
            status: .turtleNeck
        )
        
        let (skullPos, _, _, _) = KinematicsCalculator.computeDynamicPositions(
            head: head,
            neck: neck,
            posture: posture
        )
        
        // Skull must translate forward in Z to visualize forward head posture
        XCTAssertGreaterThan(skullPos.z, KinematicsCalculator.defaultSkullPosition.z)
        // Skull must translate laterally in X matching head.x offset
        XCTAssertGreaterThan(skullPos.x, KinematicsCalculator.defaultSkullPosition.x)
    }
    
    func testCameraViewModeAndCoordinateMapping() {
        let viewSize = CGSize(width: 600, height: 800)
        
        // 1. Open View (16:9 Aspect Fit - uncropped)
        let openRect = computeVideoRect(in: viewSize, mode: .openView, videoAspect: 16.0 / 9.0)
        XCTAssertEqual(openRect.width, 600.0, accuracy: 0.01)
        XCTAssertEqual(openRect.height, 337.5, accuracy: 0.01)
        XCTAssertEqual(openRect.origin.x, 0.0, accuracy: 0.01)
        XCTAssertEqual(openRect.origin.y, 231.25, accuracy: 0.01)
        
        // 2. Fill View (Aspect Fill - cropped)
        let fillRect = computeVideoRect(in: viewSize, mode: .fillView, videoAspect: 16.0 / 9.0)
        XCTAssertEqual(fillRect.height, 800.0, accuracy: 0.01)
        XCTAssertEqual(fillRect.width, 800.0 * (16.0 / 9.0), accuracy: 0.1)
        XCTAssertEqual(fillRect.origin.y, 0.0, accuracy: 0.01)
        XCTAssertLessThan(fillRect.origin.x, 0.0) // overflows horizontally to fill tall frame
        
        // 3. Vision Coordinate Mapping [0, 1] to View Space
        let centerVision = CGPoint(x: 0.5, y: 0.5)
        let mappedOpenCenter = convertVisionPointToView(centerVision, in: openRect)
        XCTAssertEqual(mappedOpenCenter.x, openRect.midX, accuracy: 0.01)
        XCTAssertEqual(mappedOpenCenter.y, openRect.midY, accuracy: 0.01)
        
        let bottomLeftVision = CGPoint(x: 0.0, y: 0.0)
        let mappedBottomLeft = convertVisionPointToView(bottomLeftVision, in: openRect)
        XCTAssertEqual(mappedBottomLeft.x, openRect.minX, accuracy: 0.01)
        XCTAssertEqual(mappedBottomLeft.y, openRect.maxY, accuracy: 0.01)
        
        let topRightVision = CGPoint(x: 1.0, y: 1.0)
        let mappedTopRight = convertVisionPointToView(topRightVision, in: openRect)
        XCTAssertEqual(mappedTopRight.x, openRect.maxX, accuracy: 0.01)
        XCTAssertEqual(mappedTopRight.y, openRect.minY, accuracy: 0.01)
    }
    
    func testCameraElevationCompensationHighAndLow() {
        let head = CGPoint(x: 0.54, y: 0.70)
        let neck = CGPoint(x: 0.50, y: 0.62)
        let shoulder = CGPoint(x: 0.50, y: 0.55)
        
        // 1. High camera looking down (+18° downward pitch)
        let highMetrics = KinematicsCalculator.computeClinicalPosture(
            head: head,
            neck: neck,
            shoulder: shoulder,
            cameraAngle: .diagonal45,
            cameraElevation: .highAboveMonitor
        )
        
        // 2. Eye level camera (0° neutral pitch)
        let eyeMetrics = KinematicsCalculator.computeClinicalPosture(
            head: head,
            neck: neck,
            shoulder: shoulder,
            cameraAngle: .diagonal45,
            cameraElevation: .eyeLevel
        )
        
        // 3. Low camera looking up (-18° upward pitch)
        let lowMetrics = KinematicsCalculator.computeClinicalPosture(
            head: head,
            neck: neck,
            shoulder: shoulder,
            cameraAngle: .diagonal45,
            cameraElevation: .lowDeskLaptop
        )
        
        // When camera is high looking down, downward projection shrinks deltaY on sensor.
        // Elevation compensation restores the true anatomical height, making compensated CVA higher than raw sensor CVA!
        XCTAssertGreaterThan(highMetrics.cvaDegrees, eyeMetrics.cvaDegrees)
        
        // When camera is low looking up, upward projection artificially inflates deltaY on sensor.
        // Elevation compensation subtracts the upward artifact, preventing false-healthy CVA!
        XCTAssertLessThan(lowMetrics.cvaDegrees, eyeMetrics.cvaDegrees)
        
        // Verify 3D dynamic position offset adjusts for camera height
        let (highSkull, _, _, _) = KinematicsCalculator.computeDynamicPositions(
            head: head,
            neck: neck,
            posture: highMetrics,
            cameraElevation: .highAboveMonitor
        )
        let (lowSkull, _, _, _) = KinematicsCalculator.computeDynamicPositions(
            head: head,
            neck: neck,
            posture: lowMetrics,
            cameraElevation: .lowDeskLaptop
        )
        XCTAssertGreaterThan(highSkull.y, lowSkull.y, "High camera compensation should adjust 3D skull height above low camera")
    }
    
    func testAnatomicalYawCalculationAcrossCameraAngles() {
        // 1. User facing screen (neutral posture) with camera at 45° diagonal (left side: facingSign = +1.0)
        let yawDiagLeft = KinematicsCalculator.computeAnatomicalYaw(
            sensorYawDegrees: 45.0,
            facingSign: 1.0,
            cameraAngle: .diagonal45
        )
        XCTAssertEqual(yawDiagLeft, 0.0, accuracy: 0.001, "When user looks at screen, anatomical head yaw relative to torso must be 0°")
        
        // 2. User facing screen with camera at 45° diagonal (right side: facingSign = -1.0)
        let yawDiagRight = KinematicsCalculator.computeAnatomicalYaw(
            sensorYawDegrees: -45.0,
            facingSign: -1.0,
            cameraAngle: .diagonal45
        )
        XCTAssertEqual(yawDiagRight, 0.0, accuracy: 0.001, "When camera is on right and user looks at screen, anatomical yaw must be 0°")
        
        // 3. User facing screen with camera at 60° oblique
        let yawOblique = KinematicsCalculator.computeAnatomicalYaw(
            sensorYawDegrees: 60.0,
            facingSign: 1.0,
            cameraAngle: .oblique60
        )
        XCTAssertEqual(yawOblique, 0.0, accuracy: 0.001)
        
        // 4. User facing screen with camera at 90° profile
        let yawProfile = KinematicsCalculator.computeAnatomicalYaw(
            sensorYawDegrees: 90.0,
            facingSign: 1.0,
            cameraAngle: .profile90
        )
        XCTAssertEqual(yawProfile, 0.0, accuracy: 0.001)
        // 5. Oblique camera (30°)
        let yaw30 = KinematicsCalculator.computeAnatomicalYaw(
            sensorYawDegrees: 30.0,
            facingSign: 1.0,
            cameraAngle: .oblique30
        )
        XCTAssertEqual(yaw30, 0.0, accuracy: 0.001)
        
        // 6. User turning head towards the 45° camera (sensor sees face looking straight into lens: yaw = 0°)
        let yawLookingAtCamera = KinematicsCalculator.computeAnatomicalYaw(
            sensorYawDegrees: 0.0,
            facingSign: 1.0,
            cameraAngle: .diagonal45
        )
        XCTAssertEqual(yawLookingAtCamera, -45.0, accuracy: 0.001, "Turning head to face camera rotates anatomical yaw -45° towards camera")
    }
    
    func testCervicalSpineAndSkullAlignmentWithTorsoAt45DegreeCamera() {
        // User looking at screen with camera at 45° diagonal
        var face = TrackedFaceMetrics(
            pitchDegrees: 0.0,
            yawDegrees: 45.0, // Vision sensor sees face at 45° relative to camera axis
            rollDegrees: 0.0,
            confidence: 0.95
        )
        face.anatomicalYawDegrees = KinematicsCalculator.computeAnatomicalYaw(
            sensorYawDegrees: face.yawDegrees,
            facingSign: 1.0,
            cameraAngle: .diagonal45
        )
        XCTAssertEqual(face.anatomicalYawDegrees, 0.0, accuracy: 0.01)
        
        let headRot = SIMD3<Float>(
            face.pitchDegrees * .pi / 180.0,
            face.anatomicalYawDegrees * .pi / 180.0,
            face.rollDegrees * .pi / 180.0
        )
        
        let cervical = KinematicsCalculator.computeCervicalVertebrae(
            skullPos: SIMD3<Float>(0.0, 0.77, 0.0),
            c7Pos: SIMD3<Float>(0.0, 0.60, -0.04),
            headRotation: headRot,
            isVisible: true
        )
        
        // Cervical vertebrae yaw must be 0.0 (aligned with thoracic spine and chest, not twisted 45°!)
        for v in cervical {
            XCTAssertEqual(v.rotation.y, 0.0, accuracy: 0.001, "Vertebra \(v.name) must not be twisted sideways when user faces screen")
        }
    }
    
    func testTorsoRotationFromShoulderTiltAndTwist() {
        let baselineSpan: Float = 0.30 // 0.35..0.65 = 0.30 normalized
        
        // 1. Shoulder roll tilt: right shoulder higher than left shoulder
        let bodyTilted = TrackedBodyMetrics(
            neckC7: TrackedJoint(point: CGPoint(x: 0.50, y: 0.60), confidence: 0.95),
            leftShoulder: TrackedJoint(point: CGPoint(x: 0.35, y: 0.52), confidence: 0.90),
            rightShoulder: TrackedJoint(point: CGPoint(x: 0.65, y: 0.58), confidence: 0.90)
        )
        let rotTilted = KinematicsCalculator.computeTorsoRotation(
            body: bodyTilted,
            posture: ClinicalPostureMetrics(),
            facingSign: 1.0,
            baselineShoulderSpan: baselineSpan
        )
        XCTAssertGreaterThan(rotTilted.z, 0.10, "Right shoulder higher must produce positive torso roll")
        
        // 2. Slouching posture produces forward thoracic pitch
        let slouchPosture = ClinicalPostureMetrics(
            cvaDegrees: 48.0,
            forwardDriftCm: 3.2,
            status: .slouching
        )
        let rotSlouch = KinematicsCalculator.computeTorsoRotation(
            body: bodyTilted,
            posture: slouchPosture,
            facingSign: 1.0,
            baselineShoulderSpan: baselineSpan
        )
        XCTAssertGreaterThan(rotSlouch.x, 0.05, "Slouching posture must produce forward thoracic pitch")
        
        // 3. Torso twist produces yaw rotation from shoulder span compression
        // When body turns ~25° left: span shrinks from 0.30 to ~0.27, nose shifts right of shoulder midpoint
        let bodyTwisted = TrackedBodyMetrics(
            nose: TrackedJoint(point: CGPoint(x: 0.53, y: 0.72), confidence: 0.95), // Nose right of mid
            neckC7: TrackedJoint(point: CGPoint(x: 0.50, y: 0.60), confidence: 0.95),
            leftShoulder: TrackedJoint(point: CGPoint(x: 0.37, y: 0.55), confidence: 0.90),
            rightShoulder: TrackedJoint(point: CGPoint(x: 0.63, y: 0.55), confidence: 0.90) // span = 0.26
        )
        let rotTwist = KinematicsCalculator.computeTorsoRotation(
            body: bodyTwisted,
            posture: ClinicalPostureMetrics(),
            facingSign: 1.0,
            baselineShoulderSpan: baselineSpan
        )
        XCTAssertGreaterThan(rotTwist.y, 0.25, "Shoulder span compression must produce positive torso yaw even when head faces forward")
    }
    
    func testTorsoAndSpineRotationWhenTurningBodyLeftAndRight() {
        let baselineSpan: Float = 0.30 // neutral: left=0.35, right=0.65
        
        let neutralBody = TrackedBodyMetrics(
            nose: TrackedJoint(point: CGPoint(x: 0.50, y: 0.72), confidence: 0.95),
            neckC7: TrackedJoint(point: CGPoint(x: 0.50, y: 0.60), confidence: 0.95),
            leftShoulder: TrackedJoint(point: CGPoint(x: 0.35, y: 0.55), confidence: 0.90),
            rightShoulder: TrackedJoint(point: CGPoint(x: 0.65, y: 0.55), confidence: 0.90)
        )
        
        // Turned left ~30°: span shrinks to 0.30*cos(30°) ≈ 0.26, nose shifts right of shoulder midpoint
        let bodyTurnedLeft = TrackedBodyMetrics(
            nose: TrackedJoint(point: CGPoint(x: 0.53, y: 0.72), confidence: 0.95),
            neckC7: TrackedJoint(point: CGPoint(x: 0.49, y: 0.60), confidence: 0.95),
            leftShoulder: TrackedJoint(point: CGPoint(x: 0.37, y: 0.55), confidence: 0.90),
            rightShoulder: TrackedJoint(point: CGPoint(x: 0.63, y: 0.55), confidence: 0.90) // span=0.26
        )
        
        // Turned right ~30°: span shrinks to 0.26, nose shifts left of shoulder midpoint
        let bodyTurnedRight = TrackedBodyMetrics(
            nose: TrackedJoint(point: CGPoint(x: 0.47, y: 0.72), confidence: 0.95),
            neckC7: TrackedJoint(point: CGPoint(x: 0.51, y: 0.60), confidence: 0.95),
            leftShoulder: TrackedJoint(point: CGPoint(x: 0.37, y: 0.55), confidence: 0.90),
            rightShoulder: TrackedJoint(point: CGPoint(x: 0.63, y: 0.55), confidence: 0.90) // span=0.26
        )
        
        // 1. Neutral posture: Torso and shoulders must be straight
        let rotScreen = KinematicsCalculator.computeTorsoRotation(
            body: neutralBody,
            facingSign: 1.0,
            baselineShoulderSpan: baselineSpan
        )
        XCTAssertEqual(rotScreen.y, 0.0, accuracy: 0.001, "Torso yaw must be 0° when facing screen")
        
        // 2. Head only turns (+35°), shoulders stay: Torso must NOT rotate
        let rotHeadOnly = KinematicsCalculator.computeTorsoRotation(
            body: neutralBody,
            facingSign: 1.0,
            baselineShoulderSpan: baselineSpan
        )
        XCTAssertEqual(rotHeadOnly.y, 0.0, accuracy: 0.001, "Torso must NOT rotate when user turns only head")
        
        // 3. Shoulders turn left: Torso MUST rotate left
        let rotLeft = KinematicsCalculator.computeTorsoRotation(
            body: bodyTurnedLeft,
            facingSign: 1.0,
            baselineShoulderSpan: baselineSpan
        )
        XCTAssertGreaterThan(rotLeft.y, 0.25, "Turning shoulders left must produce positive torso yaw")
        
        // 4. Shoulders turn right: Torso MUST rotate right
        let rotRight = KinematicsCalculator.computeTorsoRotation(
            body: bodyTurnedRight,
            facingSign: 1.0,
            baselineShoulderSpan: baselineSpan
        )
        XCTAssertLessThan(rotRight.y, -0.25, "Turning shoulders right must produce negative torso yaw")
        XCTAssertEqual(rotLeft.y, -rotRight.y, accuracy: 0.001, "Trunk rotation must be symmetric")
        
        // 5. Verify thoracic and lumbar spine vertebrae rotate with the torso turns
        let thoracicLeft = KinematicsCalculator.computeThoracicVertebrae(
            c7Pos: SIMD3<Float>(0, 0.6, 0),
            t12Pos: SIMD3<Float>(0, 0.33, 0),
            torsoRotation: rotLeft,
            isVisible: true
        )
        let thoracicRight = KinematicsCalculator.computeThoracicVertebrae(
            c7Pos: SIMD3<Float>(0, 0.6, 0),
            t12Pos: SIMD3<Float>(0, 0.33, 0),
            torsoRotation: rotRight,
            isVisible: true
        )
        XCTAssertGreaterThan(thoracicLeft.first!.rotation.y, 0.25, "T1 must rotate left with torso")
        XCTAssertLessThan(thoracicRight.first!.rotation.y, -0.25, "T1 must rotate right with torso")
        
        let lumbarLeft = KinematicsCalculator.computeLumbarVertebrae(
            t12Pos: SIMD3<Float>(0, 0.33, 0),
            basePos: SIMD3<Float>(0, 0.18, 0),
            torsoRotation: rotLeft,
            isVisible: true
        )
        let lumbarRight = KinematicsCalculator.computeLumbarVertebrae(
            t12Pos: SIMD3<Float>(0, 0.33, 0),
            basePos: SIMD3<Float>(0, 0.18, 0),
            torsoRotation: rotRight,
            isVisible: true
        )
        XCTAssertGreaterThan(lumbarLeft.first!.rotation.y, 0.10, "L1 must rotate left with torso")
        XCTAssertLessThan(lumbarRight.first!.rotation.y, -0.10, "L1 must rotate right with torso")
    }
    
    func testIndependentHeadAndShoulderPostureAndRotation() {
        let baselineSpan: Float = 0.30
        
        let neutralBody = TrackedBodyMetrics(
            nose: TrackedJoint(point: CGPoint(x: 0.50, y: 0.72), confidence: 0.95),
            neckC7: TrackedJoint(point: CGPoint(x: 0.50, y: 0.60), confidence: 0.95),
            leftShoulder: TrackedJoint(point: CGPoint(x: 0.35, y: 0.55), confidence: 0.90),
            rightShoulder: TrackedJoint(point: CGPoint(x: 0.65, y: 0.55), confidence: 0.90)
        )
        
        // Body turned left ~30°: span shrinks, nose shifts right
        let shoulderTurnedBody = TrackedBodyMetrics(
            nose: TrackedJoint(point: CGPoint(x: 0.53, y: 0.72), confidence: 0.95),
            neckC7: TrackedJoint(point: CGPoint(x: 0.49, y: 0.60), confidence: 0.95),
            leftShoulder: TrackedJoint(point: CGPoint(x: 0.37, y: 0.55), confidence: 0.90),
            rightShoulder: TrackedJoint(point: CGPoint(x: 0.63, y: 0.55), confidence: 0.90) // span=0.26
        )
        
        let shoulderTiltedBody = TrackedBodyMetrics(
            nose: TrackedJoint(point: CGPoint(x: 0.50, y: 0.72), confidence: 0.95),
            neckC7: TrackedJoint(point: CGPoint(x: 0.50, y: 0.60), confidence: 0.95),
            leftShoulder: TrackedJoint(point: CGPoint(x: 0.35, y: 0.52), confidence: 0.90),
            rightShoulder: TrackedJoint(point: CGPoint(x: 0.65, y: 0.58), confidence: 0.90) // Right shoulder +0.06 higher
        )
        
        // 1. Head turns +40° in yaw, but shoulders stay facing forward (no span change):
        let rotTorsoWhenHeadTurns = KinematicsCalculator.computeTorsoRotation(
            body: neutralBody,
            facingSign: 1.0,
            baselineShoulderSpan: baselineSpan
        )
        // Torso must NOT turn (span is still 0.30 = baseline)
        XCTAssertEqual(rotTorsoWhenHeadTurns.y, 0.0, accuracy: 0.001)
        
        // Cervical spine articulates between skull (+40°) and C7/torso (0°)
        let headRot40 = SIMD3<Float>(0.0, 40.0 * .pi / 180.0, 0.0)
        let cervical = KinematicsCalculator.computeCervicalVertebrae(
            skullPos: SIMD3<Float>(0, 0.77, 0),
            c7Pos: SIMD3<Float>(0, 0.60, 0),
            headRotation: headRot40,
            torsoRotation: rotTorsoWhenHeadTurns,
            isVisible: true
        )
        XCTAssertEqual(cervical.first!.rotation.y, headRot40.y, accuracy: 0.001, "C1 must follow skull rotation")
        XCTAssertLessThan(cervical.last!.rotation.y, 0.15, "C7 must align with torso (near 0°)")
        
        // 2. Shoulders turn in yaw (span shrinks), but head stays looking at screen (0°):
        let rotTorsoWhenShouldersTurn = KinematicsCalculator.computeTorsoRotation(
            body: shoulderTurnedBody,
            facingSign: 1.0,
            baselineShoulderSpan: baselineSpan
        )
        XCTAssertGreaterThan(rotTorsoWhenShouldersTurn.y, 0.25, "Torso must rotate when shoulders turn, even if head faces screen")
        
        // Cervical spine articulates from skull (0°) to C7/torso (turned)
        let headRot0 = SIMD3<Float>.zero
        let cervical2 = KinematicsCalculator.computeCervicalVertebrae(
            skullPos: SIMD3<Float>(0, 0.77, 0),
            c7Pos: SIMD3<Float>(0, 0.60, 0),
            headRotation: headRot0,
            torsoRotation: rotTorsoWhenShouldersTurn,
            isVisible: true
        )
        XCTAssertEqual(cervical2.first!.rotation.y, 0.0, accuracy: 0.001, "C1 must stay at 0° with skull")
        XCTAssertGreaterThan(cervical2.last!.rotation.y, 0.20, "C7 must rotate with torso")
        
        // 3. Head tilts in roll, but shoulders stay level (no span change):
        XCTAssertEqual(rotTorsoWhenHeadTurns.z, 0.0, accuracy: 0.001, "Torso roll must be 0° when shoulders are level")
        
        // 4. Shoulders tilt in roll (right shoulder raised), but head stays level:
        let rotTorsoWhenShouldersTilt = KinematicsCalculator.computeTorsoRotation(
            body: shoulderTiltedBody,
            facingSign: 1.0,
            baselineShoulderSpan: baselineSpan
        )
        XCTAssertGreaterThan(rotTorsoWhenShouldersTilt.z, 0.15, "Torso must tilt in roll when right shoulder is raised")
        XCTAssertEqual(rotTorsoWhenShouldersTilt.y, 0.0, accuracy: 0.001, "Raising shoulders must not create fake yaw rotation")
    }
    
    func testSpinalRotationDuringPostureChanges() {
        let torsoRot = SIMD3<Float>(0.12, 0.25, -0.15) // pitch, yaw, roll
        
        // 1. Thoracic spine vertebrae must rotate in all 3 axes
        let thoracic = KinematicsCalculator.computeThoracicVertebrae(
            c7Pos: SIMD3<Float>(0.0, 0.60, -0.04),
            t12Pos: SIMD3<Float>(0.0, 0.33, 0.0),
            lateralTilt: -0.15,
            torsoRotation: torsoRot,
            isVisible: true
        )
        XCTAssertEqual(thoracic.count, 12)
        XCTAssertGreaterThan(thoracic[1].rotation.x, 0.0, "Thoracic vertebrae must have pitch rotation during posture change")
        XCTAssertGreaterThan(thoracic[1].rotation.y, 0.10, "Thoracic vertebrae must have yaw twist during posture change")
        XCTAssertLessThan(thoracic[1].rotation.z, -0.05, "Thoracic vertebrae must have roll tilt during posture change")
        
        // 2. Lumbar spine vertebrae must rotate with torso
        let lumbar = KinematicsCalculator.computeLumbarVertebrae(
            t12Pos: SIMD3<Float>(0.0, 0.33, 0.0),
            basePos: SIMD3<Float>(0.0, 0.18, 0.0),
            torsoRotation: torsoRot,
            isVisible: true
        )
        XCTAssertEqual(lumbar.count, 5)
        XCTAssertGreaterThan(lumbar[0].rotation.y, 0.05, "Lumbar vertebrae must rotate in yaw with torso")
        XCTAssertLessThan(lumbar[0].rotation.z, -0.02, "Lumbar vertebrae must rotate in roll with torso")
    }
    
    func testFaceAnglesFromLandmarks() {
        // 1. Neutral face looking directly at camera
        // Subject's left eye on image right (x = 0.55), right eye on image left (x = 0.45)
        let eyeL = CGPoint(x: 0.55, y: 0.60)
        let eyeR = CGPoint(x: 0.45, y: 0.60)
        let noseNeutral = CGPoint(x: 0.50, y: 0.528) // hNose = 0.072
        let chin = CGPoint(x: 0.50, y: 0.44) // hFace = 0.16 -> ratio = 0.072/0.16 = 0.45
        let cheekL = CGPoint(x: 0.62, y: 0.53)
        let cheekR = CGPoint(x: 0.38, y: 0.53)
        
        let neutralAngles = KinematicsCalculator.computeFaceAnglesFromLandmarks(
            leftEye: eyeL,
            rightEye: eyeR,
            nose: noseNeutral,
            chin: chin,
            leftCheek: cheekL,
            rightCheek: cheekR
        )
        XCTAssertEqual(neutralAngles.rollDegrees, 0.0, accuracy: 0.1)
        XCTAssertEqual(neutralAngles.yawDegrees, 0.0, accuracy: 0.5)
        XCTAssertEqual(neutralAngles.pitchDegrees, 0.0, accuracy: 1.0)
        
        // 2. Head tilted to left (subject's left ear down towards left shoulder)
        // Subject's left eye drops in image Y (+Y is up in Vision coordinates)
        let eyeLTilted = CGPoint(x: 0.55, y: 0.58)
        let eyeRTilted = CGPoint(x: 0.45, y: 0.62)
        let tiltedAngles = KinematicsCalculator.computeFaceAnglesFromLandmarks(
            leftEye: eyeLTilted,
            rightEye: eyeRTilted,
            nose: CGPoint(x: 0.50, y: 0.528),
            chin: chin
        )
        XCTAssertLessThan(tiltedAngles.rollDegrees, -10.0, "Tilting head left must produce negative roll angle")
        
        // 3. Head turned to left (subject looks to their left)
        // Nose moves towards subject's left eye (larger X in image)
        let noseTurnedLeft = CGPoint(x: 0.53, y: 0.528)
        let turnedAngles = KinematicsCalculator.computeFaceAnglesFromLandmarks(
            leftEye: eyeL,
            rightEye: eyeR,
            nose: noseTurnedLeft,
            chin: chin
        )
        XCTAssertGreaterThan(turnedAngles.yawDegrees, 15.0, "Turning head left must produce positive yaw angle")
        
        // 4. Head tilted up (extension)
        // Nose moves closer to eye line (ratio decreases)
        let noseTiltedUp = CGPoint(x: 0.50, y: 0.55) // hNose decreases
        let pitchUpAngles = KinematicsCalculator.computeFaceAnglesFromLandmarks(
            leftEye: eyeL,
            rightEye: eyeR,
            nose: noseTiltedUp,
            chin: chin
        )
        XCTAssertGreaterThan(pitchUpAngles.pitchDegrees, 5.0, "Looking up must produce positive pitch angle")
        
        // 5. Head tilted down (flexion)
        // Nose moves closer to chin (ratio increases)
        let noseTiltedDown = CGPoint(x: 0.50, y: 0.49) // hNose increases
        let pitchDownAngles = KinematicsCalculator.computeFaceAnglesFromLandmarks(
            leftEye: eyeL,
            rightEye: eyeR,
            nose: noseTiltedDown,
            chin: chin
        )
        XCTAssertLessThan(pitchDownAngles.pitchDegrees, -5.0, "Looking down must produce negative pitch angle")
    }
    
    // MARK: - Dynamic Sagittal Angle Rules (30°–90°) Tests
    
    func testGeneralDynamicAngleFormulaAcrossAngles() {
        // Test vectors U and V
        // At 90°: ku = 0.5 (atan = 26.565°), kv = -0.5 (atan = -26.565°) -> angle = 53.13°
        let angle90 = KinematicsCalculator.computeDynamicAngle(
            dxU: 0.5, dyU: 1.0,
            dxV: -0.5, dyV: 1.0,
            alphaDegrees: 90.0
        )
        XCTAssertEqual(angle90, 53.13, accuracy: 0.1)
        
        // At 30° (sin 30° = 0.5):
        // Image dx is foreshortened by factor of 0.5 -> dxU_img = 0.25, dxV_img = -0.25
        let angle30 = KinematicsCalculator.computeDynamicAngle(
            dxU: 0.25, dyU: 1.0,
            dxV: -0.25, dyV: 1.0,
            alphaDegrees: 30.0
        )
        // With foreshortening compensation (dx_img / sin α = 0.25 / 0.5 = 0.5), recovered angle must equal angle90!
        XCTAssertEqual(angle30, angle90, accuracy: 0.1, "Dynamic compensation at 30° must recover the true 90° sagittal angle")
        
        // At 45° (sin 45° = 1/√2 ≈ 0.7071)
        let sin45 = sin(45.0 * .pi / 180.0)
        let angle45 = KinematicsCalculator.computeDynamicAngle(
            dxU: 0.5 * sin45, dyU: 1.0,
            dxV: -0.5 * sin45, dyV: 1.0,
            alphaDegrees: 45.0
        )
        XCTAssertEqual(angle45, angle90, accuracy: 0.1, "Dynamic compensation at 45° must recover the true 90° sagittal angle")
    }
    
    func testDynamicCVAInvarianceAcrossViewingAngles() {
        // True anatomical sagittal coordinates:
        // C7 at (0.50, 0.60), Tragus at (0.58, 0.72) -> Δx_true = 0.08, Δy = 0.12
        // True sagittal CVA = atan(0.12 / 0.08) * 180 / π = 56.31°
        let c7True = CGPoint(x: 0.50, y: 0.60)
        let tragusTrue = CGPoint(x: 0.58, y: 0.72)
        let trueCVA = KinematicsCalculator.computeDynamicCVA(tragus: tragusTrue, c7: c7True, alphaDegrees: 90.0)
        XCTAssertEqual(trueCVA, 56.31, accuracy: 0.1)
        
        let testAngles: [Double] = [30.0, 45.0, 60.0, 75.0, 90.0]
        for alpha in testAngles {
            let sinAlpha = sin(alpha * .pi / 180.0)
            // Foreshortened dx on camera sensor
            let dxSensor = 0.08 * sinAlpha
            let tragusSensor = CGPoint(x: c7True.x + dxSensor, y: tragusTrue.y)
            
            let recoveredCVA = KinematicsCalculator.computeDynamicCVA(
                tragus: tragusSensor,
                c7: c7True,
                alphaDegrees: alpha
            )
            XCTAssertEqual(
                recoveredCVA,
                trueCVA,
                accuracy: 0.1,
                "Dynamic CVA at \(alpha)° must recover true sagittal CVA \(trueCVA)°"
            )
        }
    }
    
    func testDynamicATJAndSHJCalculations() {
        let tragus = CGPoint(x: 0.56, y: 0.72)
        let acromion = CGPoint(x: 0.50, y: 0.54)
        let jugular = CGPoint(x: 0.52, y: 0.58)
        
        // 1. ATJ at 45°
        let atj45 = KinematicsCalculator.computeDynamicATJ(
            tragus: tragus,
            acromion: acromion,
            jugularNotch: jugular,
            alphaDegrees: 45.0
        )
        XCTAssertGreaterThan(atj45, 0.0)
        XCTAssertLessThan(atj45, 180.0)
        
        // 2. SHJ Ratio and Angle at 45°
        let (shjRatio45, shjDeg45) = KinematicsCalculator.computeDynamicSHJ(
            tragus: tragus,
            acromion: acromion,
            jugularNotch: jugular,
            alphaDegrees: 45.0
        )
        XCTAssertGreaterThan(shjRatio45, 0.0)
        XCTAssertGreaterThan(shjDeg45, 0.0)
        
        // SHJ Ratio is invariant under foreshortening because sin(α) cancels out
        let (shjRatio90, _) = KinematicsCalculator.computeDynamicSHJ(
            tragus: tragus,
            acromion: acromion,
            jugularNotch: jugular,
            alphaDegrees: 90.0
        )
        XCTAssertEqual(shjRatio45, shjRatio90, accuracy: 0.001, "SHJ ratio d_AT / d_AJ is mathematically invariant to camera angle")
    }
    
    func testC7EstimationFromJugularNotchAndAcromion() {
        // Person facing left: J is anterior at x = 0.50, A is posterior at x = 0.56
        // C7_est is further posterior (x = 2 * 0.56 - 0.50 = 0.62 > 0.50)
        let jugular = CGPoint(x: 0.50, y: 0.58)
        let acromion = CGPoint(x: 0.56, y: 0.54)
        
        let c7Est = KinematicsCalculator.estimateC7(jugularNotch: jugular, acromion: acromion)
        
        // C7 should be posterior to J (greater X when facing left) and superior to J (+Y)
        XCTAssertGreaterThan(c7Est.x, jugular.x, "C7_est should be posterior to jugular notch")
        XCTAssertGreaterThan(c7Est.y, jugular.y, "C7_est should be superior to jugular notch")
        XCTAssertEqual(c7Est.x, 0.62, accuracy: 0.001)
        XCTAssertEqual(c7Est.y, 0.60, accuracy: 0.001)
    }
    
    func testFullDynamicClinicalPostureEvaluation() {
        let tragus = CGPoint(x: 0.54, y: 0.72)
        let acromion = CGPoint(x: 0.50, y: 0.54)
        let jugular = CGPoint(x: 0.51, y: 0.58)
        
        let posture = KinematicsCalculator.computeClinicalPosture(
            tragus: tragus,
            c7: nil, // Trigger C7 estimation
            acromion: acromion,
            jugularNotch: jugular,
            cameraAngle: .diagonal45
        )
        
        XCTAssertGreaterThan(posture.cvaDegrees, 0.0)
        XCTAssertGreaterThan(posture.atjDegrees, 0.0)
        XCTAssertGreaterThan(posture.shjRatio, 0.0)
        XCTAssertFalse(posture.guidanceCue.isEmpty)
    }
    
    func testDynamicVisionLandmarkExtractionInObliqueView() {
        let tracker = VisionTracker()
        
        // Simulate real oblique profile view matching user screenshot:
        // Head facing right (yaw = +45°), near shoulder on left (x = 0.25), far shoulder on chest (x = 0.68)
        let frame = tracker.generateSyntheticKinematicFrame(time: 1.0)
        
        XCTAssertNotNil(frame.keypoints.tragus)
        XCTAssertNotNil(frame.keypoints.acromion)
        XCTAssertNotNil(frame.keypoints.jugularNotch)
        XCTAssertNotNil(frame.keypoints.c7)
        
        // Ensure CVA is not artificially locked at 89°-90° vertical
        XCTAssertLessThan(frame.posture.cvaDegrees, 85.0, "CVA should reflect true anatomical angle, not an artificial vertical line")
        XCTAssertGreaterThan(frame.posture.cvaDegrees, 40.0)
    }
    
    func testSilhouetteContourDynamicLandmarkExtraction() {
        let tracker = VisionTracker()
        
        // Synthetic contour of a subject facing left (chin.x < tragus.x):
        // Tragus at (0.51, 0.68), Chin at (0.35, 0.42)
        // Throat anterior inflection (Jugular Notch) at (0.41, 0.38)
        // Shoulder shelf with lateral apex (Acromion) at (0.87, 0.30)
        // Posterior cervical lordosis base (C7) at (0.60, 0.46)
        var syntheticContour: [CGPoint] = []
        // Head / Face
        syntheticContour.append(CGPoint(x: 0.50, y: 0.90))
        syntheticContour.append(CGPoint(x: 0.35, y: 0.50))
        // Anterior throat into Jugular Notch
        syntheticContour.append(CGPoint(x: 0.32, y: 0.44))
        syntheticContour.append(CGPoint(x: 0.41, y: 0.38)) // J inflection
        syntheticContour.append(CGPoint(x: 0.35, y: 0.30)) // chest anterior
        // Lower torso
        syntheticContour.append(CGPoint(x: 0.30, y: 0.05))
        syntheticContour.append(CGPoint(x: 0.92, y: 0.05))
        // Arm up into Acromion apex
        syntheticContour.append(CGPoint(x: 0.92, y: 0.20))
        syntheticContour.append(CGPoint(x: 0.87, y: 0.30)) // A apex
        syntheticContour.append(CGPoint(x: 0.75, y: 0.38)) // trapezius slope
        // Posterior neck / C7
        syntheticContour.append(CGPoint(x: 0.60, y: 0.46)) // C7
        syntheticContour.append(CGPoint(x: 0.59, y: 0.55))
        syntheticContour.append(CGPoint(x: 0.65, y: 0.70))
        
        let state = tracker.resolveKinematicState(
            faceObs: nil,
            bodyObs: nil,
            contourPoints: syntheticContour,
            cameraAngle: .diagonal45,
            cameraElevation: .eyeLevel,
            timestamp: 1.0
        )
        
        XCTAssertEqual(state.silhouetteContour.count, syntheticContour.count)
        XCTAssertNotNil(state.keypoints.acromion)
        XCTAssertNotNil(state.keypoints.jugularNotch)
        XCTAssertNotNil(state.keypoints.c7)
    }
    
    func testLiveVisionTrackerOnUserImage() {
        let imagePath = "/Users/destnguyxn/.gemini/antigravity/brain/848cf3b5-67f2-44fd-aecf-f2aee65e6e4e/.user_uploaded/media_1790704033641.jpg"
        guard FileManager.default.fileExists(atPath: imagePath),
              let image = NSImage(contentsOfFile: imagePath),
              let tiffData = image.tiffRepresentation,
              let bitmapImage = NSBitmapImageRep(data: tiffData),
              let cgImage = bitmapImage.cgImage else {
            return // Skip if running on machine without user test image
        }
        
        let width = cgImage.width
        let height = cgImage.height
        var pixelBuffer: CVPixelBuffer?
        let status = CVPixelBufferCreate(
            kCFAllocatorDefault,
            width,
            height,
            kCVPixelFormatType_32ARGB,
            [
                kCVPixelBufferCGImageCompatibilityKey as String: true,
                kCVPixelBufferCGBitmapContextCompatibilityKey as String: true
            ] as CFDictionary,
            &pixelBuffer
        )
        guard status == kCVReturnSuccess, let buffer = pixelBuffer else { return }
        
        CVPixelBufferLockBaseAddress(buffer, [])
        let pxData = CVPixelBufferGetBaseAddress(buffer)
        let rgbColorSpace = CGColorSpaceCreateDeviceRGB()
        if let context = CGContext(
            data: pxData,
            width: width,
            height: height,
            bitsPerComponent: 8,
            bytesPerRow: CVPixelBufferGetBytesPerRow(buffer),
            space: rgbColorSpace,
            bitmapInfo: CGImageAlphaInfo.noneSkipFirst.rawValue
        ) {
            context.draw(cgImage, in: CGRect(x: 0, y: 0, width: width, height: height))
        }
        CVPixelBufferUnlockBaseAddress(buffer, [])
        
        let tracker = VisionTracker()
        let state = tracker.processFrame(buffer, cameraAngle: .diagonal45, cameraElevation: .eyeLevel)
        
        XCTAssertGreaterThan(state.silhouetteContour.count, 50, "Silhouette contour should be extracted from person segmentation")
        XCTAssertNotNil(state.keypoints.tragus)
        XCTAssertNotNil(state.keypoints.acromion)
        XCTAssertNotNil(state.keypoints.jugularNotch)
        XCTAssertNotNil(state.keypoints.c7)
        
        // Assert clinical metrics are anatomically sound
        XCTAssertGreaterThan(state.posture.cvaDegrees, 35.0)
        XCTAssertLessThan(state.posture.cvaDegrees, 80.0)
        XCTAssertGreaterThan(state.posture.atjDegrees, 20.0)
        XCTAssertLessThan(state.posture.atjDegrees, 85.0)
        XCTAssertGreaterThan(state.posture.shjRatio, 0.5)
        XCTAssertLessThan(state.posture.shjRatio, 2.0)
    }
    
    func testLiveVisionTrackerOnNewBareCameraImage() {
        let imagePath = "/Users/destnguyxn/.gemini/antigravity/brain/848cf3b5-67f2-44fd-aecf-f2aee65e6e4e/.user_uploaded/media_1790751934640.jpg"
        guard FileManager.default.fileExists(atPath: imagePath),
              let image = NSImage(contentsOfFile: imagePath),
              let tiffData = image.tiffRepresentation,
              let bitmapImage = NSBitmapImageRep(data: tiffData),
              let cgImage = bitmapImage.cgImage else {
            return
        }
        
        let width = cgImage.width
        let height = cgImage.height
        var pixelBuffer: CVPixelBuffer?
        let status = CVPixelBufferCreate(
            kCFAllocatorDefault,
            width,
            height,
            kCVPixelFormatType_32ARGB,
            [
                kCVPixelBufferCGImageCompatibilityKey as String: true,
                kCVPixelBufferCGBitmapContextCompatibilityKey as String: true
            ] as CFDictionary,
            &pixelBuffer
        )
        guard status == kCVReturnSuccess, let buffer = pixelBuffer else { return }
        
        CVPixelBufferLockBaseAddress(buffer, [])
        let pxData = CVPixelBufferGetBaseAddress(buffer)
        let rgbColorSpace = CGColorSpaceCreateDeviceRGB()
        if let context = CGContext(
            data: pxData,
            width: width,
            height: height,
            bitsPerComponent: 8,
            bytesPerRow: CVPixelBufferGetBytesPerRow(buffer),
            space: rgbColorSpace,
            bitmapInfo: CGImageAlphaInfo.noneSkipFirst.rawValue
        ) {
            context.draw(cgImage, in: CGRect(x: 0, y: 0, width: width, height: height))
        }
        CVPixelBufferUnlockBaseAddress(buffer, [])
        
        let tracker = VisionTracker()
        let state = tracker.processFrame(buffer, cameraAngle: .diagonal45, cameraElevation: .eyeLevel)
        
        print("\n================ NEW BARE IMAGE TRACKER RESULTS ================")
        print("Tragus: \(state.keypoints.tragus ?? .zero)")
        print("C7: \(state.keypoints.c7 ?? .zero)")
        print("Acromion: \(state.keypoints.acromion ?? .zero)")
        print("Jugular Notch: \(state.keypoints.jugularNotch ?? .zero)")
        print("CVA: \(state.posture.cvaDegrees)°")
        print("ATJ: \(state.posture.atjDegrees)°")
        print("SHJ: \(state.posture.shjRatio)")
        print("FacingSign: \(state.facingSign)")
        print("Silhouette count: \(state.silhouetteContour.count)")
        print("=================================================================\n")
    }
    
    func testLiveVisionTrackerOnUserCollarImage() {
        let imagePath = "/Users/destnguyxn/.gemini/antigravity/brain/848cf3b5-67f2-44fd-aecf-f2aee65e6e4e/.user_uploaded/media_1790826792852.jpg"
        guard FileManager.default.fileExists(atPath: imagePath),
              let image = NSImage(contentsOfFile: imagePath),
              let tiffData = image.tiffRepresentation,
              let bitmapImage = NSBitmapImageRep(data: tiffData),
              let cgImage = bitmapImage.cgImage else {
            return
        }
        
        let width = cgImage.width
        let height = cgImage.height
        var pixelBuffer: CVPixelBuffer?
        let status = CVPixelBufferCreate(
            kCFAllocatorDefault,
            width,
            height,
            kCVPixelFormatType_32ARGB,
            [
                kCVPixelBufferCGImageCompatibilityKey as String: true,
                kCVPixelBufferCGBitmapContextCompatibilityKey as String: true
            ] as CFDictionary,
            &pixelBuffer
        )
        guard status == kCVReturnSuccess, let buffer = pixelBuffer else { return }
        
        CVPixelBufferLockBaseAddress(buffer, [])
        let pxData = CVPixelBufferGetBaseAddress(buffer)
        let rgbColorSpace = CGColorSpaceCreateDeviceRGB()
        if let context = CGContext(
            data: pxData,
            width: width,
            height: height,
            bitsPerComponent: 8,
            bytesPerRow: CVPixelBufferGetBytesPerRow(buffer),
            space: rgbColorSpace,
            bitmapInfo: CGImageAlphaInfo.noneSkipFirst.rawValue
        ) {
            context.draw(cgImage, in: CGRect(x: 0, y: 0, width: width, height: height))
        }
        CVPixelBufferUnlockBaseAddress(buffer, [])
        
        let tracker = VisionTracker()
        tracker.c7DepthRatio = 0.15 // Keep C7 at correct anatomical neck location
        for jRatio in [0.15, 0.30, 0.45, 0.60] {
            tracker.jDepthRatio = jRatio
            let state = tracker.processFrame(buffer, cameraAngle: .diagonal45, cameraElevation: .eyeLevel)
            print("\n================ J RATIO \(jRatio) (C7=0.15) ================")
            print("C7: \(state.keypoints.c7 ?? .zero)")
            print("Acromion: \(state.keypoints.acromion ?? .zero)")
            print("Jugular Notch (J): \(state.keypoints.jugularNotch ?? .zero)")
            print("CVA: \(state.posture.cvaDegrees)°")
        }
        print("===================================================================\n")
    }
    
    // MARK: - Advanced Relative CVA & Calibration Engine Tests
    
    func testRigidFacialConstellationCentroidAndRMS() {
        // Synthetic facial constellation points
        let p1 = CGPoint(x: 0.40, y: 0.50)
        let p2 = CGPoint(x: 0.60, y: 0.50)
        let p3 = CGPoint(x: 0.50, y: 0.70)
        let p4 = CGPoint(x: 0.50, y: 0.30)
        
        let constellation = PostureMathEngine.computeConstellation(points: [p1, p2, p3, p4])
        
        XCTAssertEqual(constellation.centroid.x, 0.50, accuracy: 0.001)
        XCTAssertEqual(constellation.centroid.y, 0.50, accuracy: 0.001)
        XCTAssertGreaterThan(constellation.rmsSpread, 0.05)
        XCTAssertEqual(constellation.scaleFactor, 1.0, accuracy: 0.001)
    }
    
    func testCameraNormalized3DDisplacementVectorIncludesVerticalShift() {
        // At low elevation laptop camera, forward movement shifts centroid down (Y) and scales up (Z)
        let baselineCentroid = CGPoint(x: 0.50, y: 0.50)
        let baselineL0: CGFloat = 0.15
        
        // Forward head movement: head shifts down (v=0.46) and slightly left (u=0.48), scale increases by 10% (rho=1.10)
        let currentCentroid = CGPoint(x: 0.48, y: 0.46)
        let currentRho: CGFloat = 1.10
        
        let q = PostureMathEngine.computeCameraDisplacement3D(
            currentCentroid: currentCentroid,
            scaleRho: currentRho,
            baselineCentroid: baselineCentroid,
            baselineL0: baselineL0,
            principalPoint: CGPoint(x: 0.5, y: 0.5),
            focalLengthPixels: 1.2
        )
        
        // Must capture negative vertical shift Y (downward in camera frame)
        XCTAssertLessThan(q.y, 0.0, "Vertical displacement Y must be non-zero for low elevation MacBook camera")
        // Must capture positive depth translation Z
        XCTAssertGreaterThan(q.z, 0.0, "Depth displacement Z must be positive when leaning closer")
    }
    
    func testWeightedForwardDisplacementProjectionAcrossViewingAngles() {
        let forwardVector45 = simd_normalize(SIMD3<Double>(0.7071, 0.20, 0.675))
        let noiseDiag = SIMD3<Double>(0.001, 0.001, 0.004) // Depth Z has higher variance
        
        let forwardMotion = CameraDisplacement3D(x: 0.10, y: 0.03, z: 0.12)
        let d = PostureMathEngine.computeWeightedForwardDisplacement(
            q: forwardMotion,
            forwardVectorG: forwardVector45,
            noiseCovarianceDiag: noiseDiag
        )
        
        XCTAssertGreaterThan(d, 0.08, "Weighted projection must detect clear forward displacement")
    }
    
    func testRelativeCVAComputationAndDelta() {
        // Test user table values:
        // CVA_0 = 55°, kappa = 0.83
        // D = 0.00 -> 55.0°
        // D = 0.15 -> ~48.7° (ΔCVA ~ -6.3°)
        // D = 0.30 -> ~43.5° (ΔCVA ~ -11.5°)
        
        let (cva0, delta0) = PostureMathEngine.computeRelativeCVA(forwardDisplacementD: 0.0, baseCvaDegrees: 55.0, kappa: 0.83)
        XCTAssertEqual(cva0, 55.0, accuracy: 0.1)
        XCTAssertEqual(delta0, 0.0, accuracy: 0.1)
        
        let (cva15, delta15) = PostureMathEngine.computeRelativeCVA(forwardDisplacementD: 0.15, baseCvaDegrees: 55.0, kappa: 0.83)
        XCTAssertEqual(cva15, 48.7, accuracy: 0.5)
        XCTAssertEqual(delta15, -6.3, accuracy: 0.5)
        
        let (cva30, delta30) = PostureMathEngine.computeRelativeCVA(forwardDisplacementD: 0.30, baseCvaDegrees: 55.0, kappa: 0.83)
        XCTAssertEqual(cva30, 43.5, accuracy: 0.5)
        XCTAssertEqual(delta30, -11.5, accuracy: 0.5)
    }
    
    func testCalibrationEngineHysteresisAndWarningStates() {
        let engine = CalibrationEngine()
        engine.startCalibration()
        
        // Feed 20 neutral samples
        for _ in 0..<20 {
            let neutral = FacialConstellation(centroid: CGPoint(x: 0.5, y: 0.5), rmsSpread: 0.15)
            engine.recordCalibrationFrame(constellation: neutral)
        }
        XCTAssertEqual(engine.currentPhase, .collectingForward)
        
        // Feed 25 forward samples
        for _ in 0..<25 {
            let forward = FacialConstellation(centroid: CGPoint(x: 0.48, y: 0.46), rmsSpread: 0.17, scaleFactor: 1.13)
            engine.recordCalibrationFrame(constellation: forward)
        }
        XCTAssertEqual(engine.currentPhase, .calibrated)
        XCTAssertTrue(engine.baseline.isCalibrated)
        
        // Evaluate neutral frame -> ΔCVA ~ 0°
        let neutralEval = engine.evaluateFrame(
            constellation: FacialConstellation(centroid: CGPoint(x: 0.5, y: 0.5), rmsSpread: 0.15, scaleFactor: 1.0)
        )
        XCTAssertEqual(neutralEval.warningLevel, 0)
        XCTAssertGreaterThan(neutralEval.relativeCVA, 53.0)
    }
}
