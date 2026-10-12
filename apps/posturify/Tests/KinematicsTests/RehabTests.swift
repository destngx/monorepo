import XCTest
import CoreGraphics
@testable import Posturify

final class RehabCalibrationTests: XCTestCase {

    // MARK: - Synthetic Fixtures

    private let neutral = RehabCalibrationTests.constellation(center: CGPoint(x: 0.50, y: 0.60))
    private let forward = RehabCalibrationTests.constellation(center: CGPoint(x: 0.47, y: 0.57), scale: 1.10)
    private let upright = RehabCalibrationTests.constellation(center: CGPoint(x: 0.515, y: 0.615), scale: 0.95)

    private static func constellation(center: CGPoint, scale: CGFloat = 1.0) -> FacialConstellation {
        let points = (0..<16).map { i -> CGPoint in
            let a = Double(i) / 16.0 * 2.0 * .pi
            return CGPoint(x: center.x + scale * 0.06 * CGFloat(cos(a)), y: center.y + scale * 0.08 * CGFloat(sin(a)))
        }
        return PostureMathEngine.computeConstellation(points: points)!
    }

    /// Feeds `frame` at 10 fps while the engine stays in `phase`, returning the next timestamp.
    @discardableResult
    private func feed(
        _ engine: CalibrationEngine,
        while phase: CalibrationEngine.CalibrationPhase,
        from t: TimeInterval,
        maxFrames: Int = 300,
        _ frame: FacialConstellation
    ) -> TimeInterval {
        var t = t
        var frames = 0
        while engine.currentPhase == phase && frames < maxFrames {
            engine.recordCalibrationFrame(constellation: frame, timestamp: t)
            t += 0.1
            frames += 1
        }
        return t
    }

    private func engineInUprightPhase() -> (CalibrationEngine, TimeInterval) {
        let engine = CalibrationEngine()
        engine.startCalibration()
        var t = feed(engine, while: .collectingNeutral, from: 0, neutral)
        t = feed(engine, while: .collectingForward, from: t, forward)
        XCTAssertEqual(engine.currentPhase, .collectingUpright)
        return (engine, t)
    }

    private func calibratedEngine(profile: PostureProfile = .moreTurtleNeck) -> CalibrationEngine {
        var (engine, t) = engineInUprightPhase()
        engine.setProfile(profile)
        t = feed(engine, while: .collectingUpright, from: t, upright)
        feed(engine, while: .validating, from: t, neutral)
        XCTAssertEqual(engine.currentPhase, .calibrated)
        return engine
    }

    // MARK: - Best Upright Capture

    func testCalibrationCapturesBestUprightCorrectionRange() {
        let engine = calibratedEngine()
        let baseline = engine.baseline
        XCTAssertLessThan(baseline.bestUprightDisplacement, -CalibrationEngine.minUprightMagnitude)
        XCTAssertEqual(baseline.correctionRange, -baseline.bestUprightDisplacement, accuracy: 1e-12)

        // The best upright pose is evaluated at D = best upright (no target yet)
        let result = engine.evaluateFrame(constellation: upright, timestamp: 0)
        XCTAssertEqual(result.forwardDisplacementD, baseline.bestUprightDisplacement, accuracy: 1e-9)
    }

    func testUprightPhaseWaitsForClearRetraction() {
        let (engine, t) = engineInUprightPhase()
        feed(engine, while: .collectingUpright, from: t, maxFrames: 60, neutral)
        XCTAssertEqual(engine.currentPhase, .collectingUpright, "Staying at the habitual neutral is not a best upright hold")
    }

    func testUprightHoldMustBeContinuous() {
        var (engine, t) = engineInUprightPhase()
        t = feed(engine, while: .collectingUpright, from: t, maxFrames: 15, upright)
        t = feed(engine, while: .collectingUpright, from: t, maxFrames: 1, neutral)
        t = feed(engine, while: .collectingUpright, from: t, maxFrames: 15, upright)
        XCTAssertEqual(engine.currentPhase, .collectingUpright, "Leaving the upright hold restarts it")

        feed(engine, while: .collectingUpright, from: t, upright)
        XCTAssertEqual(engine.currentPhase, .validating)
    }

    func testUprightPhaseTimeoutContinuesWithoutCorrectionRange() {
        var (engine, t) = engineInUprightPhase()
        t = feed(engine, while: .collectingUpright, from: t, neutral)
        XCTAssertEqual(engine.currentPhase, .validating, "The best upright step is optional and never fails calibration")

        feed(engine, while: .validating, from: t, neutral)
        XCTAssertEqual(engine.currentPhase, .calibrated)
        XCTAssertEqual(engine.baseline.correctionRange, 0.0)

        engine.setRehabTargetFraction(1.0)
        let result = engine.evaluateFrame(constellation: neutral, timestamp: 0)
        XCTAssertEqual(result.deltaCVA, 0.0, accuracy: 1e-9, "Without a correction range the target stays at the habitual neutral")
    }

    // MARK: - Rehab Target

    func testDeltaCvaIsMeasuredAgainstTheRehabTarget() {
        let engine = calibratedEngine()
        let profile = engine.profile
        let best = engine.baseline.bestUprightDisplacement
        let kappa = engine.baseline.kappa

        engine.setRehabTargetFraction(0.5)
        let atNeutral = engine.evaluateFrame(constellation: neutral, timestamp: 0)
        let (expectedTargetCva, _) = PostureMathEngine.computeRelativeCVA(
            forwardDisplacementD: 0.5 * best,
            baseCvaDegrees: profile.baseCvaDegrees,
            kappa: kappa
        )
        XCTAssertEqual(atNeutral.targetDisplacementD, 0.5 * best, accuracy: 1e-12)
        XCTAssertEqual(atNeutral.targetCVA, expectedTargetCva, accuracy: 1e-9)
        XCTAssertGreaterThan(atNeutral.targetCVA, profile.baseCvaDegrees, "The target is more upright than the habitual neutral")
        XCTAssertEqual(atNeutral.relativeCVA, profile.baseCvaDegrees, accuracy: 1e-9, "CVA itself is still anchored at the habitual neutral")
        XCTAssertEqual(atNeutral.deltaCVA, profile.baseCvaDegrees - expectedTargetCva, accuracy: 1e-9)
        XCTAssertLessThan(atNeutral.deltaCVA, 0.0, "Slumping back to the habitual neutral reads as behind the target")
    }

    func testHabitualNeutralAlertsOnceTargetIsFarEnoughAhead() {
        let engine = calibratedEngine()
        engine.setRehabTargetFraction(1.0)
        var level = 0
        for i in 0..<60 {
            level = engine.evaluateFrame(constellation: neutral, timestamp: Double(i) * 0.1).warningLevel
        }
        XCTAssertGreaterThanOrEqual(level, 1, "The full correction range exceeds the strictest mild threshold")

        var t = 6.0
        for _ in 0..<60 {
            level = engine.evaluateFrame(constellation: upright, timestamp: t).warningLevel
            t += 0.1
        }
        XCTAssertEqual(level, 0, "Holding the target posture recovers")
    }

    func testTargetChangeRestartsAlerts() {
        let engine = calibratedEngine()
        for i in 0..<40 {
            _ = engine.evaluateFrame(constellation: forward, timestamp: Double(i) * 0.1)
        }
        XCTAssertGreaterThan(engine.currentWarningLevel, 0)
        engine.setRehabTargetFraction(0.15)
        XCTAssertEqual(engine.currentWarningLevel, 0)
        XCTAssertEqual(engine.rehabTargetFraction, 0.15)
    }

    func testTargetFractionIsClamped() {
        let engine = CalibrationEngine()
        engine.setRehabTargetFraction(1.7)
        XCTAssertEqual(engine.rehabTargetFraction, 1.0)
        engine.setRehabTargetFraction(-0.2)
        XCTAssertEqual(engine.rehabTargetFraction, 0.0)
    }

    // MARK: - Adherence

    func testAdherenceCountsTrustedTimeOnTarget() {
        let engine = calibratedEngine()
        _ = engine.drainAdherence()
        var t = 0.0
        for _ in 0..<51 {
            _ = engine.evaluateFrame(constellation: neutral, timestamp: t)
            t += 0.1
        }
        // Gaps without a face are not tracked
        _ = engine.evaluateFrame(constellation: nil, timestamp: t)
        t += 10.0
        _ = engine.evaluateFrame(constellation: neutral, timestamp: t)

        let adherence = engine.drainAdherence()
        XCTAssertEqual(adherence.trackedSeconds, 5.0, accuracy: 1e-6)
        XCTAssertEqual(adherence.onTargetSeconds, 5.0, accuracy: 1e-6)
        XCTAssertEqual(engine.drainAdherence(), PostureAdherence(), "Draining clears the accumulator")
    }

    func testAdherenceExcludesTimeBehindTarget() {
        let engine = calibratedEngine()
        _ = engine.drainAdherence()
        var t = 0.0
        for _ in 0..<101 {
            _ = engine.evaluateFrame(constellation: forward, timestamp: t)
            t += 0.1
        }
        let adherence = engine.drainAdherence()
        XCTAssertEqual(adherence.trackedSeconds, 10.0, accuracy: 1e-6)
        XCTAssertLessThan(adherence.onTargetFraction, 0.5)
    }
}

final class RehabProgramTests: XCTestCase {

    private let goodDay = PostureAdherence(trackedSeconds: 3600, onTargetSeconds: 3240)  // 90%
    private let mixedDay = PostureAdherence(trackedSeconds: 3600, onTargetSeconds: 2340) // 65%
    private let poorDay = PostureAdherence(trackedSeconds: 3600, onTargetSeconds: 1080)  // 30%

    private func program(stage: Int = 0, week: [PostureAdherence?], startDay: Int = 100) -> RehabProgram {
        var program = RehabProgram(stage: stage, stageStartDay: startDay)
        for (offset, day) in week.enumerated() {
            if let day { program.record(day, day: startDay + offset) }
        }
        return program
    }

    func testDefaultsStayWithinTheProposedRange() {
        XCTAssertGreaterThanOrEqual(RehabProgram.stepFraction, 0.10)
        XCTAssertLessThanOrEqual(RehabProgram.stepFraction, 0.20)
        XCTAssertEqual(RehabProgram.maxStage, 5)
        XCTAssertEqual(RehabProgram(stage: 99).targetFraction, RehabProgram.maxTargetFraction)
        XCTAssertEqual(RehabProgram().targetFraction, 0.0)
    }

    func testAdvancesAfterAGoodWeek() {
        var p = program(week: Array(repeating: goodDay, count: 7))
        XCTAssertNil(p.evaluate(today: 106), "No change before a full week at the stage")
        XCTAssertEqual(p.evaluate(today: 107), .advanced)
        XCTAssertEqual(p.stage, 1)
        XCTAssertEqual(p.targetFraction, RehabProgram.stepFraction, accuracy: 1e-12)
        XCTAssertNil(p.evaluate(today: 107), "At most one change per week")
        XCTAssertNil(p.evaluate(today: 113))
    }

    func testWeekendsOffStillAdvance() {
        var p = program(week: [goodDay, goodDay, goodDay, goodDay, goodDay, nil, nil])
        XCTAssertEqual(p.evaluate(today: 107), .advanced)
    }

    func testTooFewActiveDaysKeepTheStage() {
        let shortDay = PostureAdherence(trackedSeconds: 20 * 60, onTargetSeconds: 20 * 60)
        var p = program(week: [goodDay, goodDay, goodDay, shortDay, shortDay, shortDay, nil])
        XCTAssertNil(p.evaluate(today: 107), "Days under 30 minutes of tracking do not count")
        XCTAssertEqual(p.stage, 0)
    }

    func testMixedWeekKeepsTheStage() {
        var p = program(stage: 2, week: [goodDay, goodDay, goodDay, mixedDay, mixedDay, poorDay, goodDay])
        XCTAssertNil(p.evaluate(today: 107))
        XCTAssertEqual(p.stage, 2)
    }

    func testPoorWeekStepsBack() {
        var p = program(stage: 2, week: [poorDay, poorDay, poorDay, mixedDay, goodDay, nil, nil])
        XCTAssertEqual(p.evaluate(today: 107), .steppedBack)
        XCTAssertEqual(p.stage, 1)

        var atStart = program(stage: 0, week: Array(repeating: poorDay, count: 7))
        XCTAssertNil(atStart.evaluate(today: 107), "Never steps back below the habitual neutral")
    }

    func testStopsAtMaxStage() {
        var p = program(stage: RehabProgram.maxStage, week: Array(repeating: goodDay, count: 7))
        XCTAssertNil(p.evaluate(today: 107))
        XCTAssertEqual(p.targetFraction, RehabProgram.maxTargetFraction, accuracy: 1e-12)
    }

    func testFirstRecordStartsTheStage() {
        var p = RehabProgram()
        p.record(goodDay, day: 50)
        XCTAssertEqual(p.stageStartDay, 50)
        p.record(goodDay, day: 50)
        XCTAssertEqual(p.dailyAdherence[50]?.trackedSeconds, 7200)
    }

    func testHistoryIsPruned() {
        var p = RehabProgram()
        for day in 0..<40 { p.record(goodDay, day: day) }
        XCTAssertEqual(p.dailyAdherence.count, RehabProgram.historyDays)
        XCTAssertNil(p.dailyAdherence[39 - RehabProgram.historyDays])
    }

    func testResetReturnsToHabitualNeutral() {
        var p = program(stage: 3, week: [goodDay])
        p.reset(today: 120)
        XCTAssertEqual(p.stage, 0)
        XCTAssertEqual(p.stageStartDay, 120)
        XCTAssertNotNil(p.dailyAdherence[100], "History is kept")
    }

    func testCodableRoundTripAndClamping() throws {
        let p = program(stage: 2, week: [goodDay, poorDay])
        let decoded = try JSONDecoder().decode(RehabProgram.self, from: JSONEncoder().encode(p))
        XCTAssertEqual(decoded, p)

        let corrupt = Data(#"{"stage": 42, "dailyAdherence": {}}"#.utf8)
        XCTAssertEqual(try JSONDecoder().decode(RehabProgram.self, from: corrupt).stage, RehabProgram.maxStage)
    }

    func testDayNumbersAreConsecutiveAcrossDaylightSaving() throws {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = try XCTUnwrap(TimeZone(identifier: "America/New_York"))
        let before = try XCTUnwrap(calendar.date(from: DateComponents(year: 2026, month: 3, day: 7, hour: 23)))
        let after = try XCTUnwrap(calendar.date(from: DateComponents(year: 2026, month: 3, day: 8, hour: 0, minute: 30)))
        XCTAssertEqual(
            RehabProgram.dayNumber(for: after, calendar: calendar) - RehabProgram.dayNumber(for: before, calendar: calendar),
            1
        )
    }
}

@MainActor
final class RehabAppStateTests: XCTestCase {

    func testRehabTargetFollowsProfileEligibilityAndPersists() throws {
        let suite = "RehabAppStateTests-\(UUID().uuidString)"
        let defaults = try XCTUnwrap(UserDefaults(suiteName: suite))
        defer { defaults.removePersistentDomain(forName: suite) }

        let stored = RehabProgram(stage: 2, stageStartDay: 10)
        defaults.set(try JSONEncoder().encode(stored), forKey: "rehabProgram")
        defaults.set(PostureProfile.slightTurtleNeck.rawValue, forKey: "postureProfile")

        let appState = AppState(defaults: defaults)
        XCTAssertEqual(appState.rehabProgram, stored)
        XCTAssertEqual(appState.calibrationEngine.rehabTargetFraction, 2 * RehabProgram.stepFraction, accuracy: 1e-12)

        appState.postureProfile = .goodPosture
        XCTAssertEqual(appState.calibrationEngine.rehabTargetFraction, 0.0, "Good posture keeps the target at the calibrated neutral")

        appState.postureProfile = .moreTurtleNeck
        appState.resetRehabProgress()
        XCTAssertEqual(appState.calibrationEngine.rehabTargetFraction, 0.0)
        let reloaded = AppState(defaults: defaults)
        XCTAssertEqual(reloaded.rehabProgram.stage, 0)
    }

    func testUnreadableStoredProgramStartsFresh() throws {
        let suite = "RehabAppStateTests-\(UUID().uuidString)"
        let defaults = try XCTUnwrap(UserDefaults(suiteName: suite))
        defer { defaults.removePersistentDomain(forName: suite) }

        defaults.set(Data("not json".utf8), forKey: "rehabProgram")
        XCTAssertEqual(AppState(defaults: defaults).rehabProgram, RehabProgram())
    }
}
