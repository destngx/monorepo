import Foundation
import SwiftUI
import AVFoundation
import CoreMedia
import QuartzCore
import os

/// Central state manager coordinating the live camera stream and Vision inference pipeline.
@MainActor
public final class AppState: ObservableObject {
    @Published public var kinematics: SkeletalKinematicState = SkeletalKinematicState()
    @Published public var cameraAngle: CameraAngleMode = PipelineSettings.defaults.cameraAngle {
        didSet { pipelineSettings.withLock { [cameraAngle] in $0.cameraAngle = cameraAngle } }
    }
    @Published public var cameraElevation: CameraElevationMode = PipelineSettings.defaults.cameraElevation {
        didSet { pipelineSettings.withLock { [cameraElevation] in $0.cameraElevation = cameraElevation } }
    }
    @Published public var isCameraRunning: Bool = false
    @Published public var selectedCameraID: String = ""
    @Published public var fps: Double = 0.0
    @Published public var cameraViewMode: CameraViewMode = .fillView
    @Published public var isCameraMirrored: Bool = true
    @Published public var isCenterStageSupported: Bool = false
    @Published public var isCenterStageActive: Bool = false
    @Published public var isPortraitEffectSupported: Bool = false
    @Published public var isPortraitEffectActive: Bool = false
    /// The user's posture profile (persisted). Sets the assumed neutral CVA and alert strictness.
    @Published public var postureProfile: PostureProfile = .default {
        didSet {
            defaults.set(postureProfile.rawValue, forKey: Self.postureProfileKey)
            calibrationEngine.setProfile(postureProfile)
            applyRehabTarget()
        }
    }
    /// Progressive rehab target and daily adherence history (persisted).
    @Published public private(set) var rehabProgram = RehabProgram()
    /// Calibrated distance from the habitual neutral to the best upright posture (face lengths, 0 when not captured).
    @Published public private(set) var correctionRange: Double = 0.0
    @Published public var calibrationPhaseText: String = "Calibrate"
    @Published public var isCalibrated: Bool = false
    @Published public var c7DepthRatio: Double = 0.15 {
        didSet { visionTracker.c7DepthRatio = c7DepthRatio }
    }
    @Published public var jDepthRatio: Double = 0.15 {
        didSet { visionTracker.jDepthRatio = jDepthRatio }
    }
    
    /// Controls whether the UI window is visible. When false, background throttling drops inference to backgroundVisionFPS.
    @Published public var isWindowVisible: Bool = PipelineSettings.defaults.isWindowVisible {
        didSet { pipelineSettings.withLock { [isWindowVisible] in $0.isWindowVisible = isWindowVisible } }
    }
    
    /// Target Vision inference rate while window is actively visible (Solution A: decoupled from 30 FPS camera preview).
    @Published public var targetVisionFPS: Double = PipelineSettings.defaults.targetVisionFPS {
        didSet { pipelineSettings.withLock { [targetVisionFPS] in $0.targetVisionFPS = targetVisionFPS } }
    }
    
    /// Target Vision inference rate when floating window is closed/hidden (Solution D: 0.5 FPS = 1 frame every 2s).
    @Published public var backgroundVisionFPS: Double = PipelineSettings.defaults.backgroundVisionFPS {
        didSet { pipelineSettings.withLock { [backgroundVisionFPS] in $0.backgroundVisionFPS = backgroundVisionFPS } }
    }
    
    public let cameraManager = CameraManager.shared
    public let visionTracker = VisionTracker.shared
    public let calibrationEngine = CalibrationEngine.shared
    /// Alerts, silence contexts, movement breaks, chin tucks and calibration health.
    public let coach: PostureCoach
    
    /// Snapshot of the settings the camera-queue frame callback needs. The @Published properties are
    /// main-actor state, so they are mirrored here (via didSet) for lock-protected access off the main thread.
    struct PipelineSettings: Sendable {
        static let defaults = PipelineSettings()
        
        var cameraAngle: CameraAngleMode = .diagonal45
        var cameraElevation: CameraElevationMode = .eyeLevel
        var isWindowVisible: Bool = true
        var targetVisionFPS: Double = 10.0
        var backgroundVisionFPS: Double = 0.5
        var lastInferenceTime: TimeInterval = 0.0
    }
    private let pipelineSettings = OSAllocatedUnfairLock(initialState: PipelineSettings.defaults)
    
    private static let postureProfileKey = "postureProfile"
    private static let rehabProgramKey = "rehabProgram"
    private static let logger = Logger(subsystem: "com.destngx.posturify", category: "AppState")
    private let defaults: UserDefaults
    
    private var frameCount: Int = 0
    private var fpsTimer: Timer?
    
    public init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
        coach = PostureCoach(
            engine: CalibrationEngine.shared,
            defaults: defaults,
            notify: { notification in
                PostureNotificationManager.shared.deliver(
                    title: notification.title,
                    subtitle: notification.subtitle,
                    body: notification.body,
                    playSound: notification.playSound
                )
            },
            readActivity: { SystemActivityMonitor.current() },
            presentGlow: { EdgeGlowController.shared.show($0, style: $1) }
        )
        postureProfile = defaults.string(forKey: Self.postureProfileKey).flatMap(PostureProfile.init(rawValue:)) ?? .default
        calibrationEngine.setProfile(postureProfile)
        rehabProgram = Self.loadRehabProgram(from: defaults)
        applyRehabTarget()
        
        setupCameraPipeline()
        startFPSTracker()
        
        // Start in live camera mode
        if cameraManager.authorizationStatus == .authorized {
            startCamera()
        } else {
            // CameraManager delivers the completion on the main queue
            cameraManager.requestPermission { [weak self] granted in
                MainActor.assumeIsolated {
                    guard let self, granted else { return }
                    self.startCamera()
                }
            }
        }
    }
    
    deinit {
        fpsTimer?.invalidate()
    }
    
    // MARK: - Camera Pipeline
    
    private func setupCameraPipeline() {
        let settings = pipelineSettings
        let tracker = visionTracker
        
        // Runs on the camera session queue. Frames are delivered serially, so inference never overlaps.
        cameraManager.onFrameCaptured = { @Sendable [weak self] pixelBuffer, _ in
            let now = CACurrentMediaTime()
            
            // Solution A & D: Throttle Vision inference (30fps camera preview, targetFPS inference)
            let perspective: (angle: CameraAngleMode, elevation: CameraElevationMode)? = settings.withLock { s in
                let targetFPS = s.isWindowVisible ? s.targetVisionFPS : s.backgroundVisionFPS
                guard now - s.lastInferenceTime >= 1.0 / max(0.05, targetFPS) else { return nil }
                s.lastInferenceTime = now
                return (s.cameraAngle, s.cameraElevation)
            }
            guard let perspective else { return }
            
            // Vision processing on background thread with selected perspective angle and elevation
            let state = tracker.processFrame(
                pixelBuffer,
                cameraAngle: perspective.angle,
                cameraElevation: perspective.elevation
            )
            
            DispatchQueue.main.async {
                MainActor.assumeIsolated {
                    guard let self else { return }
                    self.kinematics = state
                    self.frameCount += 1
                    self.coach.process(state)
                }
            }
        }
    }
    
    public func setCameraAngle(_ angle: CameraAngleMode) {
        cameraAngle = angle
    }
    
    public func setCameraElevation(_ elevation: CameraElevationMode) {
        cameraElevation = elevation
    }
    
    public func setCameraMirrored(_ mirrored: Bool) {
        isCameraMirrored = mirrored
    }
    
    public func toggleCameraMirrored() {
        setCameraMirrored(!isCameraMirrored)
    }
    
    public func startCamera() {
        cameraManager.startSession()
        isCameraRunning = true
        updateCenterStageStatus()
    }
    
    public func stopCamera() {
        cameraManager.stopSession()
        isCameraRunning = false
    }
    
    public func switchCamera(to device: AVCaptureDevice) {
        // Calibration geometry (focal length, framing, image aspect) is specific to the camera it was learned with
        if device.uniqueID != cameraManager.currentCamera?.uniqueID {
            resetCalibration()
        }
        selectedCameraID = device.uniqueID
        cameraManager.switchDevice(to: device)
        updateCenterStageStatus()
    }
    
    public func updateCenterStageStatus() {
        updateCameraEffectsStatus()
    }
    
    public func updateCameraEffectsStatus() {
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) { [weak self] in
            guard let self = self else { return }
            self.isCenterStageSupported = self.cameraManager.isCenterStageSupported
            self.isCenterStageActive = self.cameraManager.isCenterStageActive
            self.isPortraitEffectSupported = self.cameraManager.isPortraitEffectSupported
            self.isPortraitEffectActive = self.cameraManager.isPortraitEffectActive
        }
    }
    
    public func toggleCenterStage() {
        let next = !cameraManager.isCenterStageActive
        cameraManager.setCenterStage(enabled: next)
        self.isCenterStageActive = next
    }
    
    // MARK: - Relative CVA Calibration
    
    public func triggerCalibration() {
        calibrationEngine.startCalibration()
        calibrationPhaseText = calibrationEngine.currentPhase.rawValue
        isCalibrated = false
    }
    
    public func resetCalibration() {
        calibrationEngine.resetCalibration()
        calibrationPhaseText = "Calibrate"
        isCalibrated = false
    }
    
    // MARK: - Rehab Target
    
    /// True when alerts follow the progressive rehab target: a turtle-neck profile with a captured correction range.
    public var isRehabActive: Bool {
        postureProfile.isRehabEligible && isCalibrated && correctionRange > 0
    }
    
    /// Restarts the rehab target at the habitual neutral.
    public func resetRehabProgress(now: Date = Date()) {
        rehabProgram.reset(today: RehabProgram.dayNumber(for: now))
        saveRehabProgram()
        applyRehabTarget()
    }
    
    /// Folds the adherence measured since the last call into today's stats and runs the daily target evaluation.
    func refreshRehab(now: Date, calendar: Calendar = .current) {
        let today = RehabProgram.dayNumber(for: now, calendar: calendar)
        let adherence = calibrationEngine.drainAdherence()
        var program = rehabProgram
        // Exercise movements are deliberate, not habitual posture
        if adherence.trackedSeconds > 0, !coach.isExercising {
            program.record(adherence, day: today)
        }
        if isRehabActive, let change = program.evaluate(today: today) {
            Self.logger.info("Rehab target \(change == .advanced ? "advanced" : "stepped back", privacy: .public) to stage \(program.stage)")
        }
        guard program != rehabProgram else { return }
        rehabProgram = program
        saveRehabProgram()
        applyRehabTarget()
    }
    
    private func applyRehabTarget() {
        calibrationEngine.setRehabTargetFraction(postureProfile.isRehabEligible ? rehabProgram.targetFraction : 0.0)
    }
    
    private func saveRehabProgram() {
        do {
            defaults.set(try JSONEncoder().encode(rehabProgram), forKey: Self.rehabProgramKey)
        } catch {
            Self.logger.error("Failed to save rehab program: \(error.localizedDescription, privacy: .public)")
        }
    }
    
    private static func loadRehabProgram(from defaults: UserDefaults) -> RehabProgram {
        guard let data = defaults.data(forKey: rehabProgramKey) else { return RehabProgram() }
        do {
            return try JSONDecoder().decode(RehabProgram.self, from: data)
        } catch {
            logger.error("Discarding unreadable rehab program: \(error.localizedDescription, privacy: .public)")
            return RehabProgram()
        }
    }
    
    // MARK: - Performance Diagnostics
    
    private func startFPSTracker() {
        fpsTimer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                guard let self = self else { return }
                self.fps = Double(self.frameCount)
                self.frameCount = 0
                self.calibrationPhaseText = self.calibrationEngine.currentPhase.rawValue
                let baseline = self.calibrationEngine.baseline
                self.isCalibrated = baseline.isCalibrated
                self.correctionRange = baseline.correctionRange
                self.refreshRehab(now: Date())
                self.coach.tick(now: Date(), mediaTime: CACurrentMediaTime())
            }
        }
    }
}
