import XCTest
import CoreGraphics
@testable import Posturify

// MARK: - Fixtures

/// Synthetic face constellations for a calibrated engine (same geometry as `CalibrationEngineTests`).
enum CalibrationFixture {
    static let neutralCenter = CGPoint(x: 0.50, y: 0.60)
    static let forwardCenter = CGPoint(x: 0.47, y: 0.57)
    static let forwardScale: CGFloat = 1.10
    static let uprightCenter = CGPoint(x: 0.515, y: 0.615)
    static let uprightScale: CGFloat = 0.95
    static let shoulderAnchor = CGPoint(x: 0.50, y: 0.35)

    static func constellation(center: CGPoint, scale: CGFloat = 1.0) -> FacialConstellation {
        let points = (0..<16).map { i -> CGPoint in
            let a = Double(i) / 16.0 * 2.0 * .pi
            return CGPoint(x: center.x + scale * 0.06 * CGFloat(cos(a)), y: center.y + scale * 0.08 * CGFloat(sin(a)))
        }
        return PostureMathEngine.computeConstellation(points: points)!
    }

    /// Runs the full calibration (neutral, forward, best upright, validation) and returns the next free timestamp.
    @discardableResult
    static func calibrate(_ engine: CalibrationEngine, from start: TimeInterval = 0) -> TimeInterval {
        engine.startCalibration()
        var t = start
        let steps: [(CalibrationEngine.CalibrationPhase, FacialConstellation)] = [
            (.collectingNeutral, constellation(center: neutralCenter)),
            (.collectingForward, constellation(center: forwardCenter, scale: forwardScale)),
            (.collectingUpright, constellation(center: uprightCenter, scale: uprightScale)),
            (.validating, constellation(center: neutralCenter)),
        ]
        for (phase, frame) in steps {
            var frames = 0
            while engine.currentPhase == phase && frames < 200 {
                engine.recordCalibrationFrame(constellation: frame, bodyAnchor: shoulderAnchor, timestamp: t)
                t += 0.1
                frames += 1
            }
        }
        return t
    }
}

// MARK: - Alert Escalation (A1)

final class AlertEscalatorTests: XCTestCase {
    private func run(
        _ escalator: inout AlertEscalator,
        _ kind: PostureAlertKind?,
        from start: TimeInterval,
        to end: TimeInterval,
        silenced: Bool = false,
        cooldown: TimeInterval = 45
    ) -> [(TimeInterval, PostureAlertDelivery)] {
        var deliveries: [(TimeInterval, PostureAlertDelivery)] = []
        var t = start
        while t <= end + 1e-9 {
            if let d = escalator.update(kind: kind, timestamp: t, isSilenced: silenced, cooldown: cooldown) {
                deliveries.append((t, d))
            }
            t += 1
        }
        return deliveries
    }

    func testKindMappingPrefersForwardHead() {
        XCTAssertEqual(PostureAlertKind.from(warningLevel: 2, isSustainedOverextension: true), .severeForward)
        XCTAssertEqual(PostureAlertKind.from(warningLevel: 1, isSustainedOverextension: false), .mildForward)
        XCTAssertEqual(PostureAlertKind.from(warningLevel: 0, isSustainedOverextension: true), .overextension)
        XCTAssertNil(PostureAlertKind.from(warningLevel: 0, isSustainedOverextension: false))
    }

    func testAmbientIsImmediateAndMildNotifiesAfterPersistence() {
        var escalator = AlertEscalator()
        XCTAssertNil(escalator.update(kind: .mildForward, timestamp: 0, isSilenced: false, cooldown: 45))
        XCTAssertEqual(escalator.ambient, .mildForward)

        let deliveries = run(&escalator, .mildForward, from: 1, to: 100)
        XCTAssertEqual(deliveries.map(\.0), [30, 75], "First after 30 s, then once per 45 s cooldown")
        XCTAssertTrue(deliveries.allSatisfy { !$0.1.playSound }, "Mild notifications are silent")
    }

    func testEscalationToSevereBypassesCooldown() {
        var escalator = AlertEscalator()
        let mild = run(&escalator, .mildForward, from: 0, to: 30)
        XCTAssertEqual(mild.map(\.0), [30])

        let severe = run(&escalator, .severeForward, from: 31, to: 41)
        XCTAssertEqual(severe.first?.0, 41, "Severe notifies after 10 s even inside the mild cooldown")
        XCTAssertEqual(severe.first?.1, PostureAlertDelivery(kind: .severeForward, playSound: false))
    }

    func testSoundPlaysOncePerEpisodeAfterHeldSevere() {
        var escalator = AlertEscalator()
        let deliveries = run(&escalator, .severeForward, from: 0, to: 200, cooldown: 1000)
        XCTAssertEqual(deliveries.map(\.0), [10, 60])
        XCTAssertEqual(deliveries.map(\.1.playSound), [false, true])

        _ = escalator.update(kind: nil, timestamp: 201, isSilenced: false, cooldown: 1000)
        XCTAssertNil(escalator.ambient)
        let next = run(&escalator, .severeForward, from: 300, to: 400, cooldown: 0)
        XCTAssertEqual(next.filter(\.1.playSound).count, 1, "A new episode may play the sound again")
    }

    func testSilenceDefersDeliveryUntilLifted() {
        var escalator = AlertEscalator()
        XCTAssertTrue(run(&escalator, .mildForward, from: 0, to: 60, silenced: true).isEmpty)
        XCTAssertEqual(escalator.ambient, .mildForward, "Ambient state is kept while silenced")
        let after = run(&escalator, .mildForward, from: 61, to: 61)
        XCTAssertEqual(after.map(\.0), [61], "A still-active alert is delivered as soon as the silence ends")
    }

    func testRecoveryRestartsPersistence() {
        var escalator = AlertEscalator()
        _ = run(&escalator, .mildForward, from: 0, to: 20)
        _ = escalator.update(kind: nil, timestamp: 21, isSilenced: false, cooldown: 45)
        XCTAssertEqual(run(&escalator, .mildForward, from: 22, to: 51).count, 0)
        XCTAssertEqual(run(&escalator, .mildForward, from: 52, to: 52).count, 1)
    }

    func testOverextensionNotifiesAfterFifteenSeconds() {
        var escalator = AlertEscalator()
        let deliveries = run(&escalator, .overextension, from: 0, to: 20)
        XCTAssertEqual(deliveries.map(\.0), [15])
    }
}

// MARK: - Silence Contexts (A1)

final class AlertSilenceTests: XCTestCase {
    private let now = Date(timeIntervalSince1970: 1_800_000_000)

    func testPriorityOrder() {
        let busy = SystemActivity(isMicrophoneInUse: true, isCameraInUseByAnotherApp: true, isFullScreenAppActive: true)
        let later = now.addingTimeInterval(60)
        XCTAssertEqual(AlertSilence.reason(activity: busy, snoozedUntil: later, isExercising: true, now: now), .exercising)
        XCTAssertEqual(AlertSilence.reason(activity: busy, snoozedUntil: later, isExercising: false, now: now), .snoozed(until: later))
        XCTAssertEqual(AlertSilence.reason(activity: busy, snoozedUntil: nil, isExercising: false, now: now), .call)
        XCTAssertEqual(
            AlertSilence.reason(activity: SystemActivity(isCameraInUseByAnotherApp: true, isFullScreenAppActive: true), snoozedUntil: nil, isExercising: false, now: now),
            .videoMeeting
        )
        XCTAssertEqual(
            AlertSilence.reason(activity: SystemActivity(isFullScreenAppActive: true), snoozedUntil: nil, isExercising: false, now: now),
            .fullScreen
        )
        XCTAssertNil(AlertSilence.reason(activity: SystemActivity(), snoozedUntil: nil, isExercising: false, now: now))
    }

    func testExpiredSnoozeDoesNotSilence() {
        XCTAssertNil(AlertSilence.reason(activity: SystemActivity(), snoozedUntil: now, isExercising: false, now: now))
    }

    func testSnoozeEndDates() throws {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = try XCTUnwrap(TimeZone(identifier: "America/New_York"))
        let evening = try XCTUnwrap(calendar.date(from: DateComponents(year: 2026, month: 3, day: 7, hour: 22, minute: 30)))
        XCTAssertEqual(SnoozeOption.fifteenMinutes.endDate(from: evening, calendar: calendar), evening.addingTimeInterval(900))
        XCTAssertEqual(SnoozeOption.oneHour.endDate(from: evening, calendar: calendar), evening.addingTimeInterval(3600))
        // Spring-forward night: tomorrow's midnight is 1 h 30 min away, not a fixed 24 h offset
        let midnight = try XCTUnwrap(calendar.date(from: DateComponents(year: 2026, month: 3, day: 8)))
        XCTAssertEqual(SnoozeOption.untilTomorrow.endDate(from: evening, calendar: calendar), midnight)
    }
}

// MARK: - Movement Breaks (A3)

final class BreakMonitorTests: XCTestCase {
    private func run(_ monitor: inout BreakMonitor, present: Bool = true, from start: TimeInterval, to end: TimeInterval, step: TimeInterval = 1, silenced: Bool = false) -> [TimeInterval] {
        var prompts: [TimeInterval] = []
        var t = start
        while t <= end + 1e-9 {
            if monitor.update(isPresent: present, timestamp: t, isSilenced: silenced) { prompts.append(t) }
            t += step
        }
        return prompts
    }

    func testPromptsAfterIntervalThenRepeats() {
        var monitor = BreakMonitor(interval: 30 * 60)
        let prompts = run(&monitor, from: 0, to: 50 * 60, step: 10)
        XCTAssertEqual(prompts, [30 * 60, 40 * 60, 50 * 60])
        XCTAssertEqual(monitor.seatedDuration(at: 50 * 60), 50 * 60)
    }

    func testBeingAwayResetsSeatedTime() {
        var monitor = BreakMonitor(interval: 30 * 60)
        XCTAssertTrue(run(&monitor, from: 0, to: 20 * 60, step: 10).isEmpty)
        // Face absent for 2 min
        XCTAssertTrue(run(&monitor, present: false, from: 20 * 60 + 10, to: 22 * 60 + 10, step: 10).isEmpty)
        XCTAssertNil(monitor.seatedSince)
        XCTAssertEqual(monitor.seatedSince, nil)
        let prompts = run(&monitor, from: 22 * 60 + 20, to: 53 * 60, step: 10)
        XCTAssertEqual(prompts, [52 * 60 + 20], "The interval restarts when the user returns")
    }

    func testShortAbsenceDoesNotReset() {
        var monitor = BreakMonitor(interval: 30 * 60)
        _ = run(&monitor, from: 0, to: 10 * 60, step: 10)
        _ = run(&monitor, present: false, from: 10 * 60 + 10, to: 11 * 60, step: 10)
        XCTAssertEqual(monitor.seatedSince, 0)
    }

    func testFrameGapEndsSession() {
        var monitor = BreakMonitor(interval: 30 * 60)
        _ = run(&monitor, from: 0, to: 10 * 60, step: 10)
        // Camera stopped: no frames for 5 min
        XCTAssertFalse(monitor.update(isPresent: true, timestamp: 15 * 60, isSilenced: false))
        XCTAssertEqual(monitor.seatedSince, 15 * 60)
    }

    func testSilenceDefersPrompt() {
        var monitor = BreakMonitor(interval: 30 * 60)
        XCTAssertTrue(run(&monitor, from: 0, to: 35 * 60, step: 10, silenced: true).isEmpty)
        XCTAssertEqual(run(&monitor, from: 35 * 60 + 10, to: 35 * 60 + 10), [35 * 60 + 10])
    }

    func testRecordedBreakRestartsInterval() {
        var monitor = BreakMonitor(interval: 30 * 60)
        _ = run(&monitor, from: 0, to: 25 * 60, step: 10)
        monitor.recordBreak(at: 25 * 60)
        XCTAssertTrue(run(&monitor, from: 25 * 60 + 10, to: 54 * 60, step: 10).isEmpty)
        XCTAssertEqual(run(&monitor, from: 55 * 60, to: 55 * 60), [55 * 60])
    }
}

// MARK: - Calibration Health (A4)

final class CalibrationHealthMonitorTests: XCTestCase {
    private func feed(_ monitor: inout CalibrationHealthMonitor, offset: Double, pass: Bool = true, from start: TimeInterval, to end: TimeInterval) {
        var t = start
        while t <= end + 1e-9 {
            monitor.recordFrame(orthogonalOffset: offset, isQualityPass: pass, timestamp: t)
            t += 1
        }
    }

    func testPersistentShiftBecomesStale() {
        var monitor = CalibrationHealthMonitor()
        feed(&monitor, offset: 0.05, from: 0, to: 60)
        feed(&monitor, offset: 0.8, from: 61, to: 61 + 3 * 60)
        XCTAssertEqual(monitor.health, .good, "The average must cross the limit and then persist for 3 min")
        feed(&monitor, offset: 0.8, from: 61 + 3 * 60 + 1, to: 61 + 6 * 60)
        XCTAssertEqual(monitor.health, .stale(.positionShifted))
    }

    func testBriefLeanIsIgnored() {
        var monitor = CalibrationHealthMonitor()
        for cycle in 0..<20 {
            let base = TimeInterval(cycle * 120)
            feed(&monitor, offset: 1.0, from: base, to: base + 20)
            feed(&monitor, offset: 0.05, from: base + 21, to: base + 119)
        }
        XCTAssertEqual(monitor.health, .good)
    }

    func testPersistentGateFailureBecomesStale() {
        var monitor = CalibrationHealthMonitor()
        feed(&monitor, offset: 0.05, pass: false, from: 0, to: 10 * 60)
        XCTAssertEqual(monitor.health, .stale(.unreliableTracking))
    }

    func testGeometryMismatchAfterTenSeconds() {
        var monitor = CalibrationHealthMonitor()
        monitor.recordGeometryMismatch(timestamp: 0)
        monitor.recordGeometryMismatch(timestamp: 9)
        XCTAssertEqual(monitor.health, .good)
        monitor.recordGeometryMismatch(timestamp: 10)
        XCTAssertEqual(monitor.health, .stale(.cameraGeometryChanged))
    }

    func testComparableFrameInterruptsGeometryMismatch() {
        var monitor = CalibrationHealthMonitor()
        monitor.recordGeometryMismatch(timestamp: 0)
        monitor.recordFrame(orthogonalOffset: 0, isQualityPass: true, timestamp: 5)
        monitor.recordGeometryMismatch(timestamp: 6)
        monitor.recordGeometryMismatch(timestamp: 12)
        XCTAssertEqual(monitor.health, .good)
    }

    func testStaleIsStickyUntilReset() {
        var monitor = CalibrationHealthMonitor()
        feed(&monitor, offset: 1.0, from: 0, to: 6 * 60)
        XCTAssertTrue(monitor.health.isStale)
        feed(&monitor, offset: 0.0, from: 6 * 60 + 1, to: 20 * 60)
        XCTAssertTrue(monitor.health.isStale)
        monitor.reset()
        XCTAssertEqual(monitor.health, .good)
    }

    func testLongGapRestartsPersistence() {
        var monitor = CalibrationHealthMonitor()
        feed(&monitor, offset: 1.0, from: 0, to: 150)
        // Away for 10 min, then 100 s more: never 3 continuous minutes above the limit
        feed(&monitor, offset: 1.0, from: 750, to: 850)
        XCTAssertEqual(monitor.health, .good)
    }

    func testEngineDetectsShiftedSeatingAndRecalibrationClearsIt() {
        let engine = CalibrationEngine()
        var t = CalibrationFixture.calibrate(engine)
        XCTAssertEqual(engine.calibrationHealth, .good)

        // Sitting a hand-width to the side: orthogonal to the calibrated forward axis
        let shifted = CalibrationFixture.constellation(center: CGPoint(x: 0.62, y: 0.66))
        let end = t + 8 * 60
        while t < end {
            _ = engine.evaluateFrame(constellation: shifted, bodyAnchor: CalibrationFixture.shoulderAnchor, timestamp: t)
            t += 1
        }
        XCTAssertEqual(engine.calibrationHealth, .stale(.positionShifted))

        engine.dismissStaleCalibration()
        XCTAssertEqual(engine.calibrationHealth, .good)

        CalibrationFixture.calibrate(engine, from: t)
        XCTAssertEqual(engine.calibrationHealth, .good)
    }

    func testEngineNeutralSittingStaysHealthy() {
        let engine = CalibrationEngine()
        var t = CalibrationFixture.calibrate(engine)
        let neutral = CalibrationFixture.constellation(center: CalibrationFixture.neutralCenter)
        let end = t + 8 * 60
        while t < end {
            _ = engine.evaluateFrame(constellation: neutral, bodyAnchor: CalibrationFixture.shoulderAnchor, timestamp: t)
            t += 1
        }
        XCTAssertEqual(engine.calibrationHealth, .good)
    }
}

// MARK: - Chin Tucks (A2)

final class ChinTuckSessionTests: XCTestCase {
    private let range = 0.5

    /// One full rep: glide back, hold, release. Returns the next timestamp.
    private func rep(_ session: inout ChinTuckSession, from start: TimeInterval) -> TimeInterval {
        var t = start
        for _ in 0..<4 { session.update(displacement: -0.4, deltaPitch: 0, timestamp: t); t += 0.1 }
        let holdEnd = t + ChinTuckSession.holdSeconds + 0.5
        while t <= holdEnd { session.update(displacement: -0.4, deltaPitch: 0, timestamp: t); t += 0.1 }
        for _ in 0..<6 { session.update(displacement: 0, deltaPitch: 0, timestamp: t); t += 0.1 }
        return t
    }

    func testRepCycle() {
        var session = ChinTuckSession(correctionRange: range)
        session.update(displacement: 0, deltaPitch: 0, timestamp: 0)
        XCTAssertEqual(session.cue, .glideBack)
        XCTAssertEqual(session.depth, 0, accuracy: 1e-9)

        _ = rep(&session, from: 0.1)
        XCTAssertEqual(session.completedReps, 1)
        XCTAssertEqual(session.stage, .retract)
    }

    func testHoldCountsDown() {
        var session = ChinTuckSession(correctionRange: range)
        var t = 0.0
        for _ in 0..<4 { session.update(displacement: -0.4, deltaPitch: 0, timestamp: t); t += 0.1 }
        guard case .hold(let since) = session.stage else { return XCTFail("Expected a hold, got \(session.stage)") }
        session.update(displacement: -0.4, deltaPitch: 0, timestamp: since + 2.2)
        XCTAssertEqual(session.cue, .holding(remainingSeconds: 3))
        XCTAssertEqual(session.holdProgress, 2.2 / ChinTuckSession.holdSeconds, accuracy: 1e-9)
    }

    func testNoddingRestartsHold() {
        var session = ChinTuckSession(correctionRange: range)
        var t = 0.0
        for _ in 0..<4 { session.update(displacement: -0.4, deltaPitch: 0, timestamp: t); t += 0.1 }
        XCTAssertTrue({ if case .hold = session.stage { return true }; return false }())
        session.update(displacement: -0.4, deltaPitch: -15, timestamp: t)
        XCTAssertEqual(session.stage, .retract)
        XCTAssertEqual(session.cue, .keepHeadLevel)
        XCTAssertEqual(session.completedReps, 0)
    }

    func testLosingDepthRestartsHold() {
        var session = ChinTuckSession(correctionRange: range)
        var t = 0.0
        for _ in 0..<4 { session.update(displacement: -0.4, deltaPitch: 0, timestamp: t); t += 0.1 }
        for _ in 0..<4 { session.update(displacement: -0.1, deltaPitch: 0, timestamp: t); t += 0.1 }
        XCTAssertEqual(session.stage, .retract)
        XCTAssertEqual(session.completedReps, 0)
    }

    func testUntrackedFrameAsksToFaceCamera() {
        var session = ChinTuckSession(correctionRange: range)
        session.update(displacement: nil, deltaPitch: 0, timestamp: 0)
        XCTAssertEqual(session.cue, .faceCamera)
    }

    func testSetCompletesAfterTenReps() {
        var session = ChinTuckSession(correctionRange: range)
        var t = 0.0
        for _ in 0..<ChinTuckSession.repsPerSet { t = rep(&session, from: t) }
        XCTAssertTrue(session.isComplete)
        XCTAssertEqual(session.completedReps, ChinTuckSession.repsPerSet)
        XCTAssertEqual(session.cue, .setComplete)
        session.update(displacement: -0.4, deltaPitch: 0, timestamp: t)
        XCTAssertEqual(session.completedReps, ChinTuckSession.repsPerSet, "A completed set ignores further frames")
    }
}

// MARK: - Coach

@MainActor
final class PostureCoachTests: XCTestCase {
    private var defaults: UserDefaults!
    private var suiteName = ""
    private var notifications: [CoachNotification] = []
    private var glow: [PostureAlertKind?] = []
    private var glowStyles: [EdgeGlowStyle] = []
    private var activity = SystemActivity()

    override func setUp() async throws {
        suiteName = "PostureCoachTests-\(UUID().uuidString)"
        defaults = try XCTUnwrap(UserDefaults(suiteName: suiteName))
        notifications = []
        glow = []
        glowStyles = []
        activity = SystemActivity()
    }

    override func tearDown() async throws {
        defaults.removePersistentDomain(forName: suiteName)
    }

    private func makeCoach(engine: CalibrationEngine = CalibrationEngine()) -> PostureCoach {
        PostureCoach(
            engine: engine,
            defaults: defaults,
            notify: { [unowned self] in self.notifications.append($0) },
            readActivity: { [unowned self] in self.activity },
            presentGlow: { [unowned self] kind, style in
                self.glow.append(kind)
                self.glowStyles.append(style)
            }
        )
    }

    private func frame(_ t: TimeInterval, warning: Int = 0, displacement: Double = 0, pass: Bool = true, face: Bool = true) -> SkeletalKinematicState {
        SkeletalKinematicState(
            timestamp: t,
            isFacePresent: face,
            relative: RelativeCVAResult(
                deltaCVA: warning == 0 ? 0 : -8,
                displacementFace: displacement,
                quality: PostureQualityGate(qShape: pass ? 1.0 : 0.0),
                warningLevel: warning,
                profile: .goodPosture
            )
        )
    }

    func testForwardDriftLightsAmbientThenNotifies() {
        let coach = makeCoach()
        coach.tick(now: Date(), mediaTime: 0)
        for t in stride(from: 0.0, through: 29, by: 1) { coach.process(frame(t, warning: 1)) }
        XCTAssertEqual(coach.ambientAlert, .mildForward)
        XCTAssertEqual(glow.last, .mildForward)
        XCTAssertTrue(notifications.isEmpty)

        coach.process(frame(30, warning: 1))
        XCTAssertEqual(notifications.map(\.title), ["Posture Reminder"])
        XCTAssertFalse(notifications[0].playSound)
    }

    func testCallSilencesNotificationsAndGlowButKeepsAmbient() {
        let coach = makeCoach()
        activity = SystemActivity(isMicrophoneInUse: true)
        coach.tick(now: Date(), mediaTime: 0)
        XCTAssertEqual(coach.silenceReason, .call)
        for t in stride(from: 0.0, through: 60, by: 1) { coach.process(frame(t, warning: 1)) }
        XCTAssertTrue(notifications.isEmpty)
        XCTAssertEqual(coach.ambientAlert, .mildForward, "The menu bar tint still follows posture")
        XCTAssertEqual(glow.last, .some(nil), "The edge glow stays hidden during a call")

        activity = SystemActivity()
        coach.tick(now: Date(), mediaTime: 61)
        XCTAssertNil(coach.silenceReason)
        XCTAssertEqual(glow.last, .mildForward)
        coach.process(frame(62, warning: 1))
        XCTAssertEqual(notifications.count, 1)
    }

    func testSnoozeSilencesUntilExpiry() {
        let coach = makeCoach()
        let now = Date(timeIntervalSince1970: 1_800_000_000)
        coach.snooze(.fifteenMinutes, now: now)
        XCTAssertEqual(coach.silenceReason, .snoozed(until: now.addingTimeInterval(900)))
        coach.tick(now: now.addingTimeInterval(901), mediaTime: 0)
        XCTAssertNil(coach.snoozedUntil)
        XCTAssertNil(coach.silenceReason)
    }

    func testAmbientClearsWithoutTrustedFrames() {
        let coach = makeCoach()
        coach.process(frame(0, warning: 2))
        XCTAssertEqual(coach.ambientAlert, .severeForward)
        coach.process(frame(5, warning: 2, pass: false))
        coach.tick(now: Date(), mediaTime: 9)
        XCTAssertEqual(coach.ambientAlert, .severeForward)
        coach.tick(now: Date(), mediaTime: 10)
        XCTAssertNil(coach.ambientAlert)
        XCTAssertEqual(glow.last, .some(nil))
    }

    func testEdgeGlowSettingPersistsAndHidesGlow() {
        let coach = makeCoach()
        coach.process(frame(0, warning: 1))
        coach.settings.edgeGlowEnabled = false
        XCTAssertEqual(glow.last, .some(nil))
        XCTAssertFalse(makeCoach().settings.edgeGlowEnabled)
    }

    func testBreakPromptAfterInterval() {
        let coach = makeCoach()
        coach.settings.breakInterval = 30 * 60
        for t in stride(from: 0.0, through: 30 * 60, by: 10) { coach.process(frame(t)) }
        XCTAssertTrue(coach.isBreakDue)
        XCTAssertEqual(notifications.map(\.title), ["Time to Move"])
        XCTAssertEqual(notifications[0].subtitle, "Seated for 30 min")
        XCTAssertEqual(makeCoach().settings.breakInterval, 30 * 60, "The interval is persisted")

        coach.acknowledgeBreak()
        XCTAssertFalse(coach.isBreakDue)
    }

    func testBreakRemindersCanBeDisabled() {
        let coach = makeCoach()
        coach.settings.breakRemindersEnabled = false
        for t in stride(from: 0.0, through: 2 * 60 * 60, by: 10) { coach.process(frame(t)) }
        XCTAssertTrue(notifications.isEmpty)
    }

    func testStaleCalibrationNotifiesOnce() {
        let engine = CalibrationEngine()
        var t = CalibrationFixture.calibrate(engine)
        let coach = makeCoach(engine: engine)
        let shifted = CalibrationFixture.constellation(center: CGPoint(x: 0.62, y: 0.66))
        let end = t + 8 * 60
        while t < end {
            _ = engine.evaluateFrame(constellation: shifted, bodyAnchor: CalibrationFixture.shoulderAnchor, timestamp: t)
            coach.tick(now: Date(), mediaTime: t)
            t += 1
        }
        XCTAssertEqual(coach.calibrationHealth, .stale(.positionShifted))
        XCTAssertEqual(notifications.filter { $0.title == "Calibration May Be Out of Date" }.count, 1)

        coach.dismissStaleCalibration()
        XCTAssertEqual(coach.calibrationHealth, .good)
        XCTAssertEqual(engine.calibrationHealth, .good)
    }

    func testChinTucksRequireCorrectionRange() {
        let coach = makeCoach()
        XCTAssertFalse(coach.canStartChinTucks)
        coach.startChinTucks()
        XCTAssertNil(coach.chinTuckSession)
    }

    func testCompletedChinTuckSetSilencesAlertsCountsAndResetsBreak() throws {
        let engine = CalibrationEngine()
        CalibrationFixture.calibrate(engine)
        let range = engine.baseline.correctionRange
        XCTAssertGreaterThan(range, 0)
        let coach = makeCoach(engine: engine)
        coach.settings.breakInterval = 30 * 60

        var t = 0.0
        while t < 20 * 60 { coach.process(frame(t)); t += 10 }

        coach.startChinTucks()
        XCTAssertTrue(coach.isExercising)
        XCTAssertEqual(coach.silenceReason, .exercising)

        for _ in 0..<ChinTuckSession.repsPerSet {
            // Severe warnings during deliberate exercise must not alert
            let deep = -range
            for _ in 0..<70 { coach.process(frame(t, warning: 2, displacement: deep)); t += 0.1 }
            for _ in 0..<6 { coach.process(frame(t, displacement: 0)); t += 0.1 }
        }
        let session = try XCTUnwrap(coach.chinTuckSession)
        XCTAssertTrue(session.isComplete)
        XCTAssertFalse(coach.isExercising)
        XCTAssertEqual(coach.chinTuckSetsToday, 1)
        XCTAssertTrue(notifications.isEmpty)
        XCTAssertNil(coach.silenceReason)

        coach.endChinTucks()
        XCTAssertNil(coach.chinTuckSession)
        // The seated session restarted at the end of the set: no prompt 15 min later
        let restart = t
        while t < restart + 15 * 60 { coach.process(frame(t)); t += 10 }
        XCTAssertFalse(coach.isBreakDue)
        XCTAssertEqual(makeCoach(engine: engine).chinTuckSetsToday, 1, "Daily sets are persisted")
    }

    // MARK: Settings

    func testDisabledNotificationsKeepAmbientAlert() {
        let coach = makeCoach()
        coach.settings.notificationsEnabled = false
        for t in stride(from: 0.0, through: 120, by: 1) { coach.process(frame(t, warning: 2)) }
        XCTAssertTrue(notifications.isEmpty)
        XCTAssertEqual(coach.ambientAlert, .severeForward)
        XCTAssertEqual(glow.last, .severeForward)
    }

    func testNotifyDelayAppliesToMildDrift() {
        let coach = makeCoach()
        coach.settings.notifyAfter = .twoMinutes
        for t in stride(from: 0.0, through: 119, by: 1) { coach.process(frame(t, warning: 1)) }
        XCTAssertTrue(notifications.isEmpty)
        coach.process(frame(120, warning: 1))
        XCTAssertEqual(notifications.count, 1)
    }

    func testSevereSoundCanBeDisabled() {
        let coach = makeCoach()
        coach.settings.severeSoundEnabled = false
        for t in stride(from: 0.0, through: 200, by: 1) { coach.process(frame(t, warning: 2)) }
        XCTAssertFalse(notifications.isEmpty)
        XCTAssertFalse(notifications.contains { $0.playSound })
    }

    func testGlowStyleIsPassedAndPersisted() {
        let coach = makeCoach()
        let style = EdgeGlowStyle(intensity: .strong, palette: .neon, showsHint: false)
        coach.settings.edgeGlow = style
        coach.process(frame(0, warning: 1))
        XCTAssertEqual(glowStyles.last, style)
        XCTAssertEqual(makeCoach().settings.edgeGlow, style)
    }

    func testGlowPreviewCyclesKindsThenRestoresLiveState() async throws {
        let coach = makeCoach()
        coach.previewGlow()
        XCTAssertTrue(coach.isPreviewingGlow)
        coach.process(frame(0, warning: 0))
        try await Task.sleep(for: .seconds(PostureCoach.glowPreviewStepSeconds * 3 + 0.5))
        XCTAssertFalse(coach.isPreviewingGlow)
        XCTAssertEqual(glow, [.mildForward, .severeForward, .overextension, nil])
    }

    func testDisablingGlowCancelsPreview() {
        let coach = makeCoach()
        coach.previewGlow()
        coach.settings.edgeGlowEnabled = false
        XCTAssertFalse(coach.isPreviewingGlow)
        XCTAssertEqual(glow.last, .some(nil))
    }
}

final class AlertSettingsTests: XCTestCase {
    func testMissingAndInvalidFieldsFallBackToDefaults() throws {
        let json = #"{"notificationsEnabled": false, "breakInterval": 7, "notifyAfter": 999, "edgeGlow": {"palette": "cool"}}"#
        let settings = try JSONDecoder().decode(AlertSettings.self, from: Data(json.utf8))
        XCTAssertFalse(settings.notificationsEnabled)
        XCTAssertEqual(settings.breakInterval, BreakMonitor.defaultInterval, "Unsupported intervals are rejected")
        XCTAssertEqual(settings.notifyAfter, .thirtySeconds, "Unknown delays fall back to the default")
        XCTAssertEqual(settings.edgeGlow, EdgeGlowStyle(intensity: .balanced, palette: .cool, showsHint: true))
        XCTAssertTrue(settings.edgeGlowEnabled)
    }

    func testRoundTrip() throws {
        var settings = AlertSettings()
        settings.notifyAfter = .oneMinute
        settings.breakInterval = 60 * 60
        settings.edgeGlow = EdgeGlowStyle(intensity: .subtle, palette: .sunset, showsHint: false)
        let decoded = try JSONDecoder().decode(AlertSettings.self, from: JSONEncoder().encode(settings))
        XCTAssertEqual(decoded, settings)
    }

    func testEveryPaletteDistinguishesAlertKinds() {
        for palette in GlowPalette.allCases {
            let colors = [PostureAlertKind.mildForward, .severeForward, .overextension].map { kind -> [Double] in
                let c = palette.rgb(for: kind)
                return [c.red, c.green, c.blue]
            }
            XCTAssertEqual(Set(colors).count, 3, "\(palette) must use a distinct color per alert kind")
        }
    }
}
