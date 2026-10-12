import XCTest
import CoreGraphics
import simd
@testable import Posturify

final class CalibrationEngineTests: XCTestCase {

    // MARK: - Synthetic Fixtures

    private let neutralCenter = CGPoint(x: 0.50, y: 0.60)
    private let forwardCenter = CGPoint(x: 0.47, y: 0.57)
    private let forwardScale: CGFloat = 1.10
    /// Best upright: chin tucked, head retracted roughly half the forward excursion in the opposite direction.
    private let uprightCenter = CGPoint(x: 0.515, y: 0.615)
    private let uprightScale: CGFloat = 0.95
    private let shoulderAnchor = CGPoint(x: 0.50, y: 0.35)

    /// Absolute-coordinate rigid face constellation (elliptical ring), as produced by Vision.
    private func facePoints(center: CGPoint, scale: CGFloat = 1.0, distortion: CGFloat = 0.0) -> [CGPoint] {
        (0..<16).map { i in
            let a = Double(i) / 16.0 * 2.0 * .pi
            let wobble = 1.0 + distortion * CGFloat(i % 2 == 0 ? 1.0 : -1.0)
            return CGPoint(
                x: center.x + scale * 0.06 * CGFloat(cos(a)) * wobble,
                y: center.y + scale * 0.08 * CGFloat(sin(a))
            )
        }
    }

    private func constellation(center: CGPoint, scale: CGFloat = 1.0, distortion: CGFloat = 0.0) -> FacialConstellation {
        PostureMathEngine.computeConstellation(points: facePoints(center: center, scale: scale, distortion: distortion))!
    }

    private func interpolatedPose(_ t: CGFloat) -> FacialConstellation {
        constellation(
            center: CGPoint(
                x: neutralCenter.x + (forwardCenter.x - neutralCenter.x) * t,
                y: neutralCenter.y + (forwardCenter.y - neutralCenter.y) * t
            ),
            scale: 1.0 + (forwardScale - 1.0) * t
        )
    }

    /// Feeds `frame` at 10 fps while the engine stays in `phase`, returning the next timestamp.
    @discardableResult
    private func feed(
        _ engine: CalibrationEngine,
        while phase: CalibrationEngine.CalibrationPhase,
        from t: TimeInterval,
        maxFrames: Int = 200,
        bodyAnchor: CGPoint? = nil,
        _ frame: FacialConstellation?
    ) -> TimeInterval {
        var t = t
        var frames = 0
        while engine.currentPhase == phase && frames < maxFrames {
            engine.recordCalibrationFrame(constellation: frame, bodyAnchor: bodyAnchor, timestamp: t)
            t += 0.1
            frames += 1
        }
        return t
    }

    private func calibratedEngine(withBodyAnchor: Bool = true) -> CalibrationEngine {
        let engine = CalibrationEngine()
        engine.startCalibration()
        let anchor = withBodyAnchor ? shoulderAnchor : nil
        var t = feed(engine, while: .collectingNeutral, from: 0, bodyAnchor: anchor, constellation(center: neutralCenter))
        t = feed(engine, while: .collectingForward, from: t, bodyAnchor: anchor, constellation(center: forwardCenter, scale: forwardScale))
        t = feed(engine, while: .collectingUpright, from: t, bodyAnchor: anchor, constellation(center: uprightCenter, scale: uprightScale))
        feed(engine, while: .validating, from: t, bodyAnchor: anchor, constellation(center: neutralCenter))
        XCTAssertEqual(engine.currentPhase, .calibrated)
        XCTAssertTrue(engine.baseline.isCalibrated)
        return engine
    }

    // MARK: - Constellation & Shape Residual

    func testEmptyConstellationIsRejected() {
        XCTAssertNil(PostureMathEngine.computeConstellation(points: []))
    }

    func testShapeResidualIsInvariantToTranslationAndScale() {
        let base = constellation(center: neutralCenter)
        let movedAndScaled = constellation(center: CGPoint(x: 0.3, y: 0.4), scale: 1.25)
        let distorted = constellation(center: neutralCenter, distortion: 0.15)

        XCTAssertEqual(PostureMathEngine.shapeResidual(base.normalizedShape, baseline: base.normalizedShape)!, 0.0, accuracy: 1e-9)
        XCTAssertEqual(PostureMathEngine.shapeResidual(movedAndScaled.normalizedShape, baseline: base.normalizedShape)!, 0.0, accuracy: 1e-9)
        XCTAssertGreaterThan(PostureMathEngine.shapeResidual(distorted.normalizedShape, baseline: base.normalizedShape)!, 0.05)
    }

    func testNeutralFrameAfterCalibrationPassesQualityGate() {
        let engine = calibratedEngine()
        let result = engine.evaluateFrame(constellation: constellation(center: neutralCenter), bodyAnchor: shoulderAnchor, timestamp: 0)
        XCTAssertTrue(result.quality.isPass, "Identical neutral frame must pass the quality gate (qShape=\(result.quality.qShape))")
        XCTAssertEqual(result.quality.qShape, 1.0, accuracy: 1e-6)
    }

    // MARK: - Coordinates & Units

    func testConstellationUsesIsotropicWidthNormalizedCoordinates() throws {
        // 4:3 frame: a Vision-normalized square is 1.0 wide but only 0.75 tall in isotropic units.
        let c = try XCTUnwrap(PostureMathEngine.computeConstellation(
            points: [CGPoint(x: 0.4, y: 0.4), CGPoint(x: 0.6, y: 0.4), CGPoint(x: 0.6, y: 0.6), CGPoint(x: 0.4, y: 0.6)],
            imageAspect: 0.75,
            faceHeight: 0.2
        ))
        XCTAssertEqual(c.centroid.x, 0.5, accuracy: 1e-9)
        XCTAssertEqual(c.centroid.y, 0.375, accuracy: 1e-9)
        XCTAssertEqual(c.principalPoint.y, 0.375, accuracy: 1e-9)
        XCTAssertEqual(try XCTUnwrap(c.faceLength), 0.15, accuracy: 1e-9)
        XCTAssertEqual(c.isotropic(CGPoint(x: 0.5, y: 0.2)).y, 0.15, accuracy: 1e-9)
        XCTAssertEqual(Double(c.rmsSpread), sqrt(0.1 * 0.1 + 0.075 * 0.075), accuracy: 1e-9)
    }

    func testDisplacementIsMeasuredInFaceLengths() {
        let engine = CalibrationEngine()
        engine.startCalibration()
        let neutral = PostureMathEngine.computeConstellation(points: facePoints(center: neutralCenter), faceHeight: 0.2)!
        let forward = PostureMathEngine.computeConstellation(points: facePoints(center: forwardCenter, scale: forwardScale), faceHeight: 0.22)!
        var t = feed(engine, while: .collectingNeutral, from: 0, neutral)
        t = feed(engine, while: .collectingForward, from: t, forward)
        let upright = PostureMathEngine.computeConstellation(points: facePoints(center: uprightCenter, scale: uprightScale), faceHeight: 0.19)!
        t = feed(engine, while: .collectingUpright, from: t, upright)
        feed(engine, while: .validating, from: t, neutral)
        XCTAssertEqual(Double(engine.baseline.faceLength0), 0.2, accuracy: 1e-9)

        // Pure lateral shift of 0.02 (one tenth of a face length) at constant scale
        let shifted = PostureMathEngine.computeConstellation(
            points: facePoints(center: CGPoint(x: neutralCenter.x + 0.02, y: neutralCenter.y)),
            faceHeight: 0.2
        )!
        let q = engine.displacement(of: shifted)
        XCTAssertEqual(q.x, 0.1, accuracy: 1e-9)
        XCTAssertEqual(q.y, 0.0, accuracy: 1e-9)
        XCTAssertEqual(q.z, 0.0, accuracy: 1e-9)
    }

    func testFaceLengthFallsBackToNominalSpreadRatio() {
        let engine = calibratedEngine()
        let spread = constellation(center: neutralCenter).rmsSpread
        XCTAssertEqual(engine.baseline.faceLength0, spread / PostureMathEngine.nominalSpreadToFaceLength, accuracy: 1e-9)
    }

    // MARK: - Calibration Flow

    func testForwardPhaseWaitsForMovement() {
        let engine = CalibrationEngine()
        engine.startCalibration()
        let t = feed(engine, while: .collectingNeutral, from: 0, constellation(center: neutralCenter))
        feed(engine, while: .collectingForward, from: t, maxFrames: 60, constellation(center: neutralCenter))
        XCTAssertEqual(engine.currentPhase, .collectingForward, "Staying neutral must not be learned as the forward vector")
        XCTAssertFalse(engine.baseline.isCalibrated)
    }

    func testForwardHoldMustBeContinuous() {
        let engine = CalibrationEngine()
        engine.startCalibration()
        var t = feed(engine, while: .collectingNeutral, from: 0, constellation(center: neutralCenter))
        let forward = constellation(center: forwardCenter, scale: forwardScale)
        t = feed(engine, while: .collectingForward, from: t, maxFrames: 20, forward)
        t = feed(engine, while: .collectingForward, from: t, maxFrames: 1, constellation(center: neutralCenter))
        feed(engine, while: .collectingForward, from: t, maxFrames: 20, forward)
        XCTAssertEqual(engine.currentPhase, .collectingForward, "A return to neutral restarts the forward hold")

        feed(engine, while: .collectingForward, from: t + 2.0, forward)
        XCTAssertEqual(engine.currentPhase, .collectingUpright)
    }

    func testValidationRequiresReturnToNeutralBeforeCommitting() {
        let engine = CalibrationEngine()
        engine.startCalibration()
        var t = feed(engine, while: .collectingNeutral, from: 0, constellation(center: neutralCenter))
        let forward = constellation(center: forwardCenter, scale: forwardScale)
        t = feed(engine, while: .collectingForward, from: t, forward)
        t = feed(engine, while: .collectingUpright, from: t, constellation(center: uprightCenter, scale: uprightScale))
        XCTAssertEqual(engine.currentPhase, .validating)
        XCTAssertFalse(engine.baseline.isCalibrated, "Baseline is only committed after validation")

        t = feed(engine, while: .validating, from: t, maxFrames: 30, forward)
        XCTAssertEqual(engine.currentPhase, .validating)

        feed(engine, while: .validating, from: t, constellation(center: neutralCenter))
        XCTAssertEqual(engine.currentPhase, .calibrated)
        XCTAssertTrue(engine.baseline.isCalibrated)
        XCTAssertFalse(engine.isCollecting)
    }

    func testCalibrationTimesOutWithoutFace() {
        let engine = CalibrationEngine()
        engine.startCalibration()
        var t = feed(engine, while: .collectingNeutral, from: 0, maxFrames: 5, constellation(center: neutralCenter))
        t = feed(engine, while: .collectingNeutral, from: t, maxFrames: 300, nil)
        XCTAssertEqual(engine.currentPhase, .failed)
        XCTAssertFalse(engine.isCollecting)
        XCTAssertFalse(engine.baseline.isCalibrated)
        XCTAssertLessThanOrEqual(t, CalibrationEngine.phaseTimeoutSeconds + 1.0)
    }

    func testRecalibrationSuspendsEvaluation() {
        let engine = calibratedEngine()
        engine.startCalibration()
        XCTAssertFalse(engine.baseline.isCalibrated)
        let result = engine.evaluateFrame(constellation: interpolatedPose(1.0), bodyAnchor: shoulderAnchor, timestamp: 0)
        XCTAssertEqual(result.warningLevel, 0)
        XCTAssertEqual(result.deltaCVA, 0.0, accuracy: 1e-9)
    }

    func testAspectChangeInvalidatesFrames() throws {
        let engine = calibratedEngine()
        let widescreen = try XCTUnwrap(PostureMathEngine.computeConstellation(points: facePoints(center: neutralCenter), imageAspect: 0.5625))
        XCTAssertFalse(engine.evaluateFrame(constellation: widescreen, timestamp: 0).quality.isPass)
    }

    // MARK: - Calibration Result

    func testForwardCalibrationCapturesDepthComponent() {
        let engine = calibratedEngine()
        XCTAssertGreaterThan(engine.baseline.forwardVectorG.z, 0.5, "Leaning closer scales the face up, so g must have a strong +Z component")
    }

    func testNoFaceFramesDoNotDisturbSmoothedDisplacement() {
        let engine = calibratedEngine()
        let first = engine.evaluateFrame(constellation: interpolatedPose(0.1), bodyAnchor: shoulderAnchor, timestamp: 0)

        let missing = engine.evaluateFrame(constellation: nil, bodyAnchor: nil, timestamp: 0.1)
        XCTAssertFalse(missing.quality.isPass)
        XCTAssertEqual(missing.forwardDisplacementD, first.forwardDisplacementD, accuracy: 1e-12)

        for i in 0..<10 {
            _ = engine.evaluateFrame(constellation: nil, bodyAnchor: nil, timestamp: 0.2 + Double(i) * 0.1)
        }
        let after = engine.evaluateFrame(constellation: interpolatedPose(0.1), bodyAnchor: shoulderAnchor, timestamp: 2.0)
        XCTAssertEqual(after.forwardDisplacementD, first.forwardDisplacementD, accuracy: 1e-9)
    }

    // MARK: - Body Anchor Fusion

    func testBodyAnchorDoesNotBiasNeutralPosture() {
        let engine = calibratedEngine()
        let result = engine.evaluateFrame(constellation: constellation(center: neutralCenter), bodyAnchor: shoulderAnchor, timestamp: 0)
        XCTAssertEqual(result.displacementBody, 0.0, accuracy: 1e-6)
        XCTAssertEqual(result.deltaCVA, 0.0, accuracy: 0.1)
        XCTAssertEqual(result.quality.qConsistency, 1.0, accuracy: 1e-6)
    }

    func testHeadOnlyForwardMovementAgreesAcrossEstimators() {
        let engine = calibratedEngine()
        let result = engine.evaluateFrame(constellation: interpolatedPose(0.5), bodyAnchor: shoulderAnchor, timestamp: 0)
        XCTAssertGreaterThan(result.displacementFace, 0.15)
        XCTAssertEqual(result.displacementBody, result.displacementFace, accuracy: result.displacementFace * 0.15)
        XCTAssertTrue(result.quality.isPass)
    }

    func testWholeTorsoMovingTowardCameraDoesNotEscalate() {
        let engine = calibratedEngine()
        // Entire upper body translates and scales together: head-to-shoulder geometry is unchanged.
        let rho: CGFloat = 1.10
        let shift = CGPoint(x: -0.03, y: -0.03)
        let face = constellation(center: CGPoint(x: neutralCenter.x + shift.x, y: neutralCenter.y + shift.y), scale: rho)
        let anchor = CGPoint(
            x: face.centroid.x + (shoulderAnchor.x - neutralCenter.x) * rho,
            y: face.centroid.y + (shoulderAnchor.y - neutralCenter.y) * rho
        )
        var last = RelativeCVAResult()
        for i in 0..<60 {
            last = engine.evaluateFrame(constellation: face, bodyAnchor: anchor, timestamp: Double(i) * 0.1)
        }
        XCTAssertEqual(last.displacementBody, 0.0, accuracy: 1e-6)
        XCTAssertGreaterThan(last.displacementFace, 0.25)
        XCTAssertFalse(last.quality.isPass, "Face/body disagreement must fail the gate")
        XCTAssertEqual(last.warningLevel, 0)
    }

    // MARK: - End-to-End Escalation

    func testSustainedForwardHeadEscalatesAndRecovers() {
        let engine = calibratedEngine()
        var t = 0.0
        var result = RelativeCVAResult()

        for _ in 0..<20 { // 2s neutral
            result = engine.evaluateFrame(constellation: constellation(center: neutralCenter), bodyAnchor: shoulderAnchor, timestamp: t)
            t += 0.1
        }
        XCTAssertEqual(result.warningLevel, 0)

        for _ in 0..<30 { // 3s full forward head
            result = engine.evaluateFrame(constellation: interpolatedPose(1.0), bodyAnchor: shoulderAnchor, timestamp: t)
            t += 0.1
        }
        XCTAssertLessThanOrEqual(result.deltaCVA, -15.0)
        XCTAssertEqual(result.warningLevel, 2)
        XCTAssertTrue(result.isSustainedForwardHead)

        for _ in 0..<60 { // 6s neutral
            result = engine.evaluateFrame(constellation: constellation(center: neutralCenter), bodyAnchor: shoulderAnchor, timestamp: t)
            t += 0.1
        }
        XCTAssertEqual(result.warningLevel, 0)
    }

    func testFaceOnlyModeStillEscalates() {
        let engine = calibratedEngine(withBodyAnchor: false)
        var result = RelativeCVAResult()
        for i in 0..<30 {
            result = engine.evaluateFrame(constellation: interpolatedPose(1.0), bodyAnchor: nil, timestamp: Double(i) * 0.1)
        }
        XCTAssertEqual(result.warningLevel, 2)
    }

    // MARK: - Overextension

    func testOverextensionRequiresSustainedHold() {
        let engine = calibratedEngine()
        let neutral = constellation(center: neutralCenter)
        var result = RelativeCVAResult()
        var t = 0.0
        while t < CalibrationEngine.overextensionHoldSeconds - 0.15 {
            result = engine.evaluateFrame(constellation: neutral, bodyAnchor: shoulderAnchor, pitch: 15, timestamp: t)
            t += 0.1
        }
        XCTAssertTrue(result.isOverextended)
        XCTAssertFalse(result.isSustainedOverextension, "A brief head tilt must not notify")

        result = engine.evaluateFrame(constellation: neutral, bodyAnchor: shoulderAnchor, pitch: 15, timestamp: CalibrationEngine.overextensionHoldSeconds)
        XCTAssertTrue(result.isSustainedOverextension)

        result = engine.evaluateFrame(constellation: neutral, bodyAnchor: shoulderAnchor, pitch: 0, timestamp: CalibrationEngine.overextensionHoldSeconds + 0.1)
        XCTAssertFalse(result.isOverextended)
        XCTAssertFalse(result.isSustainedOverextension)
    }

    // MARK: - Concurrency

    func testConcurrentCalibrationAndEvaluationIsSafe() {
        let engine = calibratedEngine()
        let face = constellation(center: neutralCenter)
        DispatchQueue.concurrentPerform(iterations: 400) { i in
            switch i % 4 {
            case 0: engine.startCalibration()
            case 1: engine.recordCalibrationFrame(constellation: face, bodyAnchor: shoulderAnchor, timestamp: Double(i))
            case 2: _ = engine.evaluateFrame(constellation: face, bodyAnchor: shoulderAnchor, timestamp: Double(i))
            default: _ = engine.baseline; _ = engine.currentPhase
            }
        }
    }
}

final class ForwardHeadAlertHysteresisTests: XCTestCase {

    private func feed(_ h: inout ForwardHeadAlertHysteresis, delta: Double, from start: Double, to end: Double, step: Double = 0.1) -> Int {
        var t = start
        var level = h.level
        while t <= end + 1e-9 {
            level = h.update(deltaCva: delta, timestamp: t)
            t += step
        }
        return level
    }

    func testMildDriftRequiresSustainedDuration() {
        var h = ForwardHeadAlertHysteresis()
        XCTAssertEqual(feed(&h, delta: -6, from: 0, to: 2.9), 0)
        XCTAssertEqual(feed(&h, delta: -6, from: 3.0, to: 3.0), 1)
    }

    func testStaleTimerDoesNotCauseImmediateAlert() {
        var h = ForwardHeadAlertHysteresis()
        _ = h.update(deltaCva: -6, timestamp: 0)
        XCTAssertEqual(feed(&h, delta: -3, from: 0.1, to: 60), 0) // buffer zone for a minute
        XCTAssertEqual(h.update(deltaCva: -6, timestamp: 60.1), 0, "A new dip must start a fresh timer")
        XCTAssertEqual(feed(&h, delta: -6, from: 60.2, to: 62.9), 0)
        XCTAssertEqual(feed(&h, delta: -6, from: 63.1, to: 63.1), 1)
    }

    func testOscillationAcrossInnerThresholdKeepsOuterTimerRunning() {
        var h = ForwardHeadAlertHysteresis()
        var t = 0.0
        var level = 0
        while t <= 5.0 {
            level = h.update(deltaCva: Int(t * 10) % 2 == 0 ? -9.9 : -10.1, timestamp: t)
            t += 0.1
        }
        XCTAssertEqual(level, 1, "Continuous drift beyond -5° must raise level 1; -10° band was never held continuously")
    }

    func testSevereDropEscalatesQuickly() {
        var h = ForwardHeadAlertHysteresis()
        XCTAssertEqual(feed(&h, delta: -16, from: 0, to: 1.4), 0)
        XCTAssertEqual(feed(&h, delta: -16, from: 1.5, to: 1.5), 2)
    }

    func testRecoveryRequiresSustainedUprightPosture() {
        var h = ForwardHeadAlertHysteresis()
        XCTAssertEqual(feed(&h, delta: -16, from: 0, to: 2), 2)
        XCTAssertEqual(feed(&h, delta: -3, from: 2.1, to: 30), 2, "Buffer zone holds the current level")
        XCTAssertEqual(feed(&h, delta: -1, from: 30.1, to: 33.0), 2)
        XCTAssertEqual(feed(&h, delta: -1, from: 33.1, to: 33.2), 0)
    }
}

final class KinematicsCalculatorRegressionTests: XCTestCase {

    private func expectedStatus(for cva: Double) -> PostureStatus {
        if cva < KinematicsCalculator.PostureConstraints.turtleNeckCvaMaxDegrees { return .turtleNeck }
        if cva <= KinematicsCalculator.PostureConstraints.optimalCvaMinDegrees { return .caution }
        return .optimal
    }

    func testElevationAdjustedCvaIsReclassified() {
        // Eye-level geometry lands in the turtle-neck band; a low camera lifts the adjusted CVA above 60°.
        let eyeLevel = KinematicsCalculator.computeClinicalPosture(
            head: CGPoint(x: 0.62, y: 0.70), neck: CGPoint(x: 0.50, y: 0.55), shoulder: CGPoint(x: 0.52, y: 0.50),
            effectiveElevationDegrees: 0.0
        )
        let elevated = KinematicsCalculator.computeClinicalPosture(
            head: CGPoint(x: 0.62, y: 0.70), neck: CGPoint(x: 0.50, y: 0.55), shoulder: CGPoint(x: 0.52, y: 0.50),
            effectiveElevationDegrees: 40.0
        )
        XCTAssertNotEqual(expectedStatus(for: eyeLevel.cvaDegrees), expectedStatus(for: elevated.cvaDegrees), "Fixture must cross a band")
        XCTAssertEqual(elevated.status, expectedStatus(for: elevated.cvaDegrees))
        XCTAssertTrue(elevated.guidanceCue.contains(String(format: "%.1f", elevated.cvaDegrees)) || elevated.status == .optimal)
    }

    func testCameraElevationCompensatesLiveAbsoluteCVA() {
        let tracker = VisionTracker()
        let eyeLevel = tracker.resolveKinematicState(faceObs: nil, bodyObs: nil, cameraElevation: .eyeLevel, timestamp: 0)
        let high = tracker.resolveKinematicState(faceObs: nil, bodyObs: nil, cameraElevation: .highAboveMonitor, timestamp: 0)
        XCTAssertEqual(high.posture.cvaDegrees - eyeLevel.posture.cvaDegrees, CameraElevationMode.highAboveMonitor.pitchDegrees * 0.4, accuracy: 1e-6)
        XCTAssertEqual(high.posture.status, KinematicsCalculator.classifyCVA(high.posture.cvaDegrees).status)
    }

    func testCoincidentEyesDoNotProduceNaN() {
        let eye = CGPoint(x: 0.5, y: 0.6)
        let angles = KinematicsCalculator.computeFaceAnglesFromLandmarks(
            leftEye: eye, rightEye: eye, nose: CGPoint(x: 0.5, y: 0.5), chin: CGPoint(x: 0.5, y: 0.4),
            visionPitch: 3, visionYaw: 4, visionRoll: 5
        )
        XCTAssertTrue(angles.pitchDegrees.isFinite)
        XCTAssertTrue(angles.yawDegrees.isFinite)
        XCTAssertTrue(angles.rollDegrees.isFinite)
        XCTAssertEqual(angles.yawDegrees, 4, accuracy: 1e-6, "Degenerate eye line falls back to the Vision estimate")
        XCTAssertEqual(angles.pitchDegrees, 3, accuracy: 1e-6)
    }
}
