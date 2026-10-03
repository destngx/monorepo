import Foundation
import SwiftUI
import AVFoundation
import CoreMedia
import QuartzCore

/// Central state manager coordinating the live camera stream and Vision inference pipeline.
@MainActor
public final class AppState: ObservableObject {
    @Published public var kinematics: SkeletalKinematicState = SkeletalKinematicState()
    @Published public var cameraAngle: CameraAngleMode = .diagonal45
    @Published public var cameraElevation: CameraElevationMode = .eyeLevel
    @Published public var isCameraRunning: Bool = false
    @Published public var selectedCameraID: String = ""
    @Published public var fps: Double = 0.0
    @Published public var cameraViewMode: CameraViewMode = .fillView
    @Published public var isCameraMirrored: Bool = true
    @Published public var isCenterStageSupported: Bool = false
    @Published public var isCenterStageActive: Bool = false
    @Published public var isPortraitEffectSupported: Bool = false
    @Published public var isPortraitEffectActive: Bool = false
    @Published public var calibrationPhaseText: String = "Calibrate"
    @Published public var isCalibrated: Bool = false
    @Published public var c7DepthRatio: Double = 0.15 {
        didSet { visionTracker.c7DepthRatio = c7DepthRatio }
    }
    @Published public var jDepthRatio: Double = 0.15 {
        didSet { visionTracker.jDepthRatio = jDepthRatio }
    }
    
    /// Controls whether the UI window is visible. When false, background throttling drops inference to backgroundVisionFPS.
    @Published public var isWindowVisible: Bool = true
    
    /// Target Vision inference rate while window is actively visible (Solution A: decoupled from 30 FPS camera preview).
    @Published public var targetVisionFPS: Double = 10.0
    
    /// Target Vision inference rate when floating window is closed/hidden (Solution D: 0.5 FPS = 1 frame every 2s).
    @Published public var backgroundVisionFPS: Double = 0.5
    
    public let cameraManager = CameraManager.shared
    public let visionTracker = VisionTracker.shared
    public let calibrationEngine = CalibrationEngine.shared
    
    private var lastFrameTime: TimeInterval = 0.0
    private var lastInferenceTime: TimeInterval = 0.0
    private let inferenceLock = NSLock()
    private var frameCount: Int = 0
    private var fpsTimer: Timer?
    
    public init() {
        setupCameraPipeline()
        startFPSTracker()
        
        // Start in live camera mode
        if cameraManager.authorizationStatus == .authorized {
            startCamera()
        } else {
            cameraManager.requestPermission { [weak self] granted in
                guard let self = self else { return }
                if granted {
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
        cameraManager.onFrameCaptured = { [weak self] pixelBuffer, timestamp in
            guard let self = self else { return }
            
            let now = CACurrentMediaTime()
            let targetFPS = self.isWindowVisible ? self.targetVisionFPS : self.backgroundVisionFPS
            let minInterval = 1.0 / max(0.05, targetFPS)
            
            // Solution A & D: Throttle Vision inference (30fps camera preview, targetFPS inference)
            guard now - self.lastInferenceTime >= minInterval else { return }
            
            // Solution B: Concurrency guard - immediately drop frame if previous inference is still running
            guard self.inferenceLock.try() else { return }
            self.lastInferenceTime = now
            
            let currentAngle = self.cameraAngle
            let currentElevation = self.cameraElevation
            
            // Vision processing on background thread with selected perspective angle and elevation
            let state = self.visionTracker.processFrame(
                pixelBuffer,
                cameraAngle: currentAngle,
                cameraElevation: currentElevation
            )
            
            self.inferenceLock.unlock()
            
            DispatchQueue.main.async {
                self.kinematics = state
                self.frameCount += 1
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
    
    // MARK: - Performance Diagnostics
    
    private func startFPSTracker() {
        fpsTimer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                guard let self = self else { return }
                self.fps = Double(self.frameCount)
                self.frameCount = 0
                self.calibrationPhaseText = self.calibrationEngine.currentPhase.rawValue
                self.isCalibrated = self.calibrationEngine.baseline.isCalibrated
            }
        }
    }
}
