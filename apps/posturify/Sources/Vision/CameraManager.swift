import Foundation
import AVFoundation
import CoreMedia

/// Hardware camera capture manager using AVFoundation for zero-latency frame streaming.
public final class CameraManager: NSObject, ObservableObject, AVCaptureVideoDataOutputSampleBufferDelegate, @unchecked Sendable {
    public static let shared = CameraManager()
    
    public let captureSession = AVCaptureSession()
    private let videoOutput = AVCaptureVideoDataOutput()
    private let sessionQueue = DispatchQueue(label: "com.destngx.posturify.sessionQueue")
    
    @Published public var isRunning = false
    @Published public var authorizationStatus: AVAuthorizationStatus = .notDetermined
    @Published public var availableCameras: [AVCaptureDevice] = []
    @Published public var currentCamera: AVCaptureDevice?
    
    public var onFrameCaptured: ((CVPixelBuffer, CMTime) -> Void)?
    
    public override init() {
        super.init()
        setupDeviceNotifications()
        refreshAvailableCameras()
        checkPermissions()
    }
    
    deinit {
        NotificationCenter.default.removeObserver(self)
    }
    
    private func setupDeviceNotifications() {
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(handleDeviceChange),
            name: AVCaptureDevice.wasConnectedNotification,
            object: nil
        )
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(handleDeviceChange),
            name: AVCaptureDevice.wasDisconnectedNotification,
            object: nil
        )
    }
    
    @objc private func handleDeviceChange(_ notification: Notification) {
        refreshAvailableCameras()
    }
    
    public func checkPermissions() {
        let status = AVCaptureDevice.authorizationStatus(for: .video)
        DispatchQueue.main.async {
            self.authorizationStatus = status
        }
    }
    
    public func requestPermission(completion: @escaping (Bool) -> Void) {
        AVCaptureDevice.requestAccess(for: .video) { granted in
            DispatchQueue.main.async {
                self.authorizationStatus = granted ? .authorized : .denied
                completion(granted)
            }
        }
    }
    
    public func refreshAvailableCameras() {
        var deviceTypes: [AVCaptureDevice.DeviceType] = [
            .builtInWideAngleCamera,
            .external,
            .continuityCamera
        ]
        if #available(macOS 13.0, *) {
            deviceTypes.append(.deskViewCamera)
        }
        
        let discoverySession = AVCaptureDevice.DiscoverySession(
            deviceTypes: deviceTypes,
            mediaType: .video,
            position: .unspecified
        )
        
        var devices = discoverySession.devices
        
        if #available(macOS 14.0, *) {
            if let sysPref = AVCaptureDevice.systemPreferredCamera,
               !devices.contains(where: { $0.uniqueID == sysPref.uniqueID }) {
                devices.append(sysPref)
            }
            if let userPref = AVCaptureDevice.userPreferredCamera,
               !devices.contains(where: { $0.uniqueID == userPref.uniqueID }) {
                devices.append(userPref)
            }
        }
        
        DispatchQueue.main.async {
            self.availableCameras = devices
            if self.currentCamera == nil || !devices.contains(where: { $0.uniqueID == self.currentCamera?.uniqueID }) {
                if let continuity = devices.first(where: { $0.deviceType == .continuityCamera }) {
                    self.currentCamera = continuity
                } else {
                    self.currentCamera = devices.first
                }
            }
        }
    }
    
    public var isCenterStageSupported: Bool {
        guard let current = currentCamera else { return false }
        if #available(macOS 12.3, *) {
            return current.activeFormat.isCenterStageSupported
        }
        return false
    }
    
    public var isCenterStageActive: Bool {
        if #available(macOS 12.3, *) {
            return AVCaptureDevice.isCenterStageEnabled
        }
        return false
    }
    
    public func setCenterStage(enabled: Bool) {
        if #available(macOS 12.3, *) {
            AVCaptureDevice.centerStageControlMode = .cooperative
            AVCaptureDevice.isCenterStageEnabled = enabled
        }
    }
    
    // MARK: - Portrait Effect (Background Blur)
    
    public var isPortraitEffectSupported: Bool {
        guard let current = currentCamera else { return false }
        if #available(macOS 12.3, *) {
            return current.activeFormat.isPortraitEffectSupported
        }
        return false
    }
    
    public var isPortraitEffectActive: Bool {
        if #available(macOS 12.3, *) {
            return AVCaptureDevice.isPortraitEffectEnabled
        }
        return false
    }
    
    public func switchDevice(to device: AVCaptureDevice) {
        sessionQueue.async {
            self.captureSession.beginConfiguration()
            for input in self.captureSession.inputs {
                self.captureSession.removeInput(input)
            }
            if let input = try? AVCaptureDeviceInput(device: device),
               self.captureSession.canAddInput(input) {
                self.captureSession.addInput(input)
            }
            self.captureSession.commitConfiguration()
            
            DispatchQueue.main.async {
                self.currentCamera = device
            }
        }
    }
    
    public func startSession(with device: AVCaptureDevice? = nil) {
        sessionQueue.async {
            guard !self.captureSession.isRunning else { return }
            
            self.captureSession.beginConfiguration()
            self.captureSession.sessionPreset = .high
            
            // Remove previous inputs
            for input in self.captureSession.inputs {
                self.captureSession.removeInput(input)
            }
            
            let selectedDevice = device ?? self.availableCameras.first ?? AVCaptureDevice.default(for: .video)
            guard let camera = selectedDevice,
                  let input = try? AVCaptureDeviceInput(device: camera) else {
                self.captureSession.commitConfiguration()
                return
            }
            
            if self.captureSession.canAddInput(input) {
                self.captureSession.addInput(input)
            }
            
            if !self.captureSession.outputs.contains(self.videoOutput) {
                self.videoOutput.alwaysDiscardsLateVideoFrames = true
                self.videoOutput.videoSettings = [
                    kCVPixelBufferPixelFormatTypeKey as String: Int(kCVPixelFormatType_32BGRA)
                ]
                self.videoOutput.setSampleBufferDelegate(self, queue: self.sessionQueue)
                if self.captureSession.canAddOutput(self.videoOutput) {
                    self.captureSession.addOutput(self.videoOutput)
                }
            }
            
            self.captureSession.commitConfiguration()
            self.captureSession.startRunning()
            
            DispatchQueue.main.async {
                self.isRunning = self.captureSession.isRunning
                self.currentCamera = camera
            }
        }
    }
    
    public func stopSession() {
        sessionQueue.async {
            guard self.captureSession.isRunning else { return }
            self.captureSession.stopRunning()
            DispatchQueue.main.async {
                self.isRunning = false
            }
        }
    }
    
    public func captureOutput(
        _ output: AVCaptureOutput,
        didOutput sampleBuffer: CMSampleBuffer,
        from connection: AVCaptureConnection
    ) {
        guard let pixelBuffer = CMSampleBufferGetImageBuffer(sampleBuffer) else { return }
        let timestamp = CMSampleBufferGetPresentationTimeStamp(sampleBuffer)
        onFrameCaptured?(pixelBuffer, timestamp)
    }
}
