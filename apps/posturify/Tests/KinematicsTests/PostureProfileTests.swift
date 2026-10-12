import XCTest
import CoreGraphics
@testable import Posturify

final class PostureProfileTests: XCTestCase {

    // MARK: - Presets

    func testPresetParameters() {
        let good = PostureProfile.goodPosture
        XCTAssertEqual(good.baseCvaDegrees, 55)
        XCTAssertEqual(good.notificationCooldown, 30)
        XCTAssertEqual(good.alertThresholds, ForwardHeadAlertThresholds(
            mild: -5, substantial: -10, severe: -15, recovery: -2.5,
            mildHoldSeconds: 3, substantialHoldSeconds: 4, severeHoldSeconds: 1.5, recoveryHoldSeconds: 3
        ))

        let slight = PostureProfile.slightTurtleNeck
        XCTAssertEqual(slight.baseCvaDegrees, 48)
        XCTAssertEqual(slight.notificationCooldown, 45)
        XCTAssertEqual(slight.alertThresholds, ForwardHeadAlertThresholds(
            mild: -4, substantial: -8, severe: -12, recovery: -2,
            mildHoldSeconds: 2.5, substantialHoldSeconds: 3.5, severeHoldSeconds: 1.5, recoveryHoldSeconds: 3
        ))

        let more = PostureProfile.moreTurtleNeck
        XCTAssertEqual(more.baseCvaDegrees, 42)
        XCTAssertEqual(more.notificationCooldown, 60)
        XCTAssertEqual(more.alertThresholds, ForwardHeadAlertThresholds(
            mild: -3, substantial: -6, severe: -10, recovery: -1.5,
            mildHoldSeconds: 2, substantialHoldSeconds: 3, severeHoldSeconds: 1.5, recoveryHoldSeconds: 3
        ))
    }

    func testProfilesAreProgressivelyStricter() {
        let profiles = PostureProfile.allCases
        XCTAssertEqual(profiles, [.goodPosture, .slightTurtleNeck, .moreTurtleNeck])
        for (looser, stricter) in zip(profiles, profiles.dropFirst()) {
            XCTAssertGreaterThan(stricter.alertThresholds.mild, looser.alertThresholds.mild)
            XCTAssertGreaterThan(stricter.alertThresholds.substantial, looser.alertThresholds.substantial)
            XCTAssertGreaterThan(stricter.alertThresholds.severe, looser.alertThresholds.severe)
            XCTAssertLessThan(stricter.baseCvaDegrees, looser.baseCvaDegrees)
        }
    }

    // MARK: - Hysteresis

    private func level(after delta: Double, for seconds: Double, profile: PostureProfile) -> Int {
        var h = ForwardHeadAlertHysteresis(thresholds: profile.alertThresholds)
        var t = 0.0
        var level = 0
        while t <= seconds + 1e-9 {
            level = h.update(deltaCva: delta, timestamp: t)
            t += 0.1
        }
        return level
    }

    func testSmallDriftAlertsOnlyForTurtleNeckProfiles() {
        XCTAssertEqual(level(after: -4.5, for: 10, profile: .goodPosture), 0)
        XCTAssertEqual(level(after: -4.5, for: 2.4, profile: .slightTurtleNeck), 0)
        XCTAssertEqual(level(after: -4.5, for: 2.5, profile: .slightTurtleNeck), 1)
        XCTAssertEqual(level(after: -3.5, for: 2.0, profile: .moreTurtleNeck), 1)
    }

    func testSevereThresholdFollowsProfile() {
        XCTAssertEqual(level(after: -11, for: 1.5, profile: .goodPosture), 0)
        XCTAssertEqual(level(after: -11, for: 1.5, profile: .moreTurtleNeck), 2)
    }

    // MARK: - Engine

    private func facePoints(center: CGPoint, scale: CGFloat = 1.0) -> [CGPoint] {
        (0..<16).map { i in
            let a = Double(i) / 16.0 * 2.0 * .pi
            return CGPoint(x: center.x + scale * 0.06 * CGFloat(cos(a)), y: center.y + scale * 0.08 * CGFloat(sin(a)))
        }
    }

    private func calibratedEngine() -> CalibrationEngine {
        let engine = CalibrationEngine()
        engine.startCalibration()
        let neutral = PostureMathEngine.computeConstellation(points: facePoints(center: CGPoint(x: 0.5, y: 0.6)))
        let forward = PostureMathEngine.computeConstellation(points: facePoints(center: CGPoint(x: 0.47, y: 0.57), scale: 1.1))
        let upright = PostureMathEngine.computeConstellation(points: facePoints(center: CGPoint(x: 0.515, y: 0.615), scale: 0.95))
        var t = 0.0
        for (phase, frame) in [
            (CalibrationEngine.CalibrationPhase.collectingNeutral, neutral),
            (.collectingForward, forward),
            (.collectingUpright, upright),
            (.validating, neutral),
        ] {
            var frames = 0
            while engine.currentPhase == phase && frames < 200 {
                engine.recordCalibrationFrame(constellation: frame, timestamp: t)
                t += 0.1
                frames += 1
            }
        }
        XCTAssertTrue(engine.baseline.isCalibrated)
        return engine
    }

    func testEngineUsesProfileNeutralCva() throws {
        let engine = calibratedEngine()
        let neutral = try XCTUnwrap(PostureMathEngine.computeConstellation(points: facePoints(center: CGPoint(x: 0.5, y: 0.6))))
        for profile in PostureProfile.allCases {
            engine.setProfile(profile)
            XCTAssertEqual(engine.profile, profile)
            let result = engine.evaluateFrame(constellation: neutral, timestamp: 0)
            XCTAssertEqual(result.relativeCVA, profile.baseCvaDegrees, accuracy: 1e-6)
            XCTAssertEqual(result.profile, profile)
        }
    }

    func testProfileChangeKeepsCalibrationAndResetsAlerts() throws {
        let engine = calibratedEngine()
        let forward = try XCTUnwrap(PostureMathEngine.computeConstellation(points: facePoints(center: CGPoint(x: 0.47, y: 0.57), scale: 1.1)))
        for i in 0..<30 {
            _ = engine.evaluateFrame(constellation: forward, timestamp: Double(i) * 0.1)
        }
        XCTAssertEqual(engine.currentWarningLevel, 2)

        engine.setProfile(.moreTurtleNeck)
        XCTAssertTrue(engine.baseline.isCalibrated, "Profiles do not change the learned geometry")
        XCTAssertEqual(engine.currentWarningLevel, 0, "Alert state restarts under the new thresholds")
    }

    func testRecalibrationKeepsProfile() {
        let engine = CalibrationEngine()
        engine.setProfile(.slightTurtleNeck)
        engine.startCalibration()
        engine.resetCalibration()
        XCTAssertEqual(engine.profile, .slightTurtleNeck)
    }

    // MARK: - Notifications

    func testNotificationCooldownFollowsProfile() {
        // A stricter profile repeats reminders less often (escalation is covered in AlertEscalatorTests)
        XCTAssertLessThan(PostureProfile.goodPosture.notificationCooldown, PostureProfile.slightTurtleNeck.notificationCooldown)
        XCTAssertLessThan(PostureProfile.slightTurtleNeck.notificationCooldown, PostureProfile.moreTurtleNeck.notificationCooldown)
        var escalator = AlertEscalator()
        var deliveries: [TimeInterval] = []
        for t in stride(from: 0.0, through: 120, by: 1) {
            if escalator.update(kind: .mildForward, timestamp: t, isSilenced: false, cooldown: PostureProfile.moreTurtleNeck.notificationCooldown) != nil {
                deliveries.append(t)
            }
        }
        XCTAssertEqual(deliveries, [30, 90])
    }

    // MARK: - Persistence

    @MainActor
    func testProfileSelectionPersists() throws {
        let suite = "PostureProfileTests-\(UUID().uuidString)"
        let defaults = try XCTUnwrap(UserDefaults(suiteName: suite))
        defer { defaults.removePersistentDomain(forName: suite) }

        let first = AppState(defaults: defaults)
        XCTAssertEqual(first.postureProfile, .goodPosture)
        first.postureProfile = .moreTurtleNeck
        XCTAssertEqual(first.calibrationEngine.profile, .moreTurtleNeck)

        let second = AppState(defaults: defaults)
        XCTAssertEqual(second.postureProfile, .moreTurtleNeck)
        XCTAssertEqual(second.calibrationEngine.profile, .moreTurtleNeck)

        first.postureProfile = .goodPosture
    }
}
