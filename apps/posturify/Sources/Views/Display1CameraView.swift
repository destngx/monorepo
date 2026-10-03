import SwiftUI
import AVFoundation

/// Display 1: Live camera view displaying the user feed, 2D facial/body skeleton lines,
/// landmark tracking dots (ear, jaw, chin, neck, shoulders, spine), and live kinematic telemetry.
public struct Display1CameraView: View {
    @ObservedObject var appState: AppState
    
    public init(appState: AppState) {
        self.appState = appState
    }
    
    public var body: some View {
        GeometryReader { geo in
            ZStack {
                // Background & Video Feed Layer
                Color(red: 0.06, green: 0.08, blue: 0.11)
                
                CameraPreviewRepresentable(
                    session: appState.cameraManager.captureSession,
                    videoGravity: appState.cameraViewMode == .openView ? .resizeAspect : .resizeAspectFill,
                    isMirrored: appState.isCameraMirrored
                )
                .clipped()
                
                if !appState.isCameraRunning {
                    VStack(spacing: 8) {
                        ProgressView()
                            .scaleEffect(1.1)
                        Text("Connecting camera feed...")
                            .font(.system(size: 11, weight: .medium, design: .monospaced))
                            .foregroundColor(.white.opacity(0.6))
                    }
                }
                
                // 2D Skeleton Lines & Landmark Dots Canvas
                SkeletonOverlayCanvas(
                    kinematics: appState.kinematics,
                    viewSize: geo.size,
                    viewMode: appState.cameraViewMode,
                    isMirrored: appState.isCameraMirrored
                )
                
                // In-Camera Calibration & Direct Angle Guidance Overlay
                VStack {
                    if appState.kinematics.posture.isDirectAngleFacing {
                        directAngleRecommendationBanner
                            .transition(.move(edge: .top).combined(with: .opacity))
                            .padding(.top, 12)
                    }
                    
                    Spacer()
                    
                    if !appState.isCalibrated {
                        calibrationPromptBanner
                            .transition(.move(edge: .bottom).combined(with: .opacity))
                            .padding(.bottom, 12)
                    }
                }
                .padding(12)
            }
        }
    }
    
    // MARK: - Calibration Onboarding & Guidance Banner
    
    private var calibrationPromptBanner: some View {
        HStack(spacing: 12) {
            Image(systemName: isCalibrating ? "hourglass.bottomhalf.filled" : "target")
                .font(.system(size: 15, weight: .bold))
                .foregroundColor(isCalibrating ? .orange : .cyan)
                .symbolEffect(.pulse, isActive: isCalibrating)
            
            VStack(alignment: .leading, spacing: 2) {
                Text(isCalibrating ? appState.calibrationPhaseText : "Posture Calibration Recommended")
                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                    .foregroundColor(.white)
                
                Text(isCalibrating ? "Follow on-screen instruction for accurate 3D vector learning." : "Calibrate once (takes 2 seconds) for precise forward head detection.")
                    .font(.system(size: 9.5, weight: .regular))
                    .foregroundColor(.white.opacity(0.75))
            }
            
            Spacer()
            
            if !isCalibrating {
                Button(action: {
                    appState.triggerCalibration()
                }) {
                    HStack(spacing: 5) {
                        Image(systemName: "play.fill")
                            .font(.system(size: 8))
                        Text("Start (2s)")
                            .font(.system(size: 10.5, weight: .bold, design: .monospaced))
                    }
                    .foregroundColor(.black)
                    .padding(.horizontal, 10)
                    .padding(.vertical, 5)
                    .background(Color.cyan)
                    .clipShape(Capsule())
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 8)
        .background(
            RoundedRectangle(cornerRadius: 10)
                .fill(Color(red: 0.10, green: 0.12, blue: 0.16).opacity(0.92))
                .shadow(color: Color.black.opacity(0.4), radius: 6, x: 0, y: 3)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 10)
                .stroke(isCalibrating ? Color.orange.opacity(0.6) : Color.cyan.opacity(0.4), lineWidth: 1.0)
        )
        .frame(maxWidth: 480)
    }
    
    private var isCalibrating: Bool {
        appState.calibrationEngine.currentPhase != .idle && appState.calibrationEngine.currentPhase != .calibrated
    }
    
    // MARK: - Direct Angle Recommendation Banner
    
    private var directAngleRecommendationBanner: some View {
        HStack(spacing: 10) {
            Image(systemName: "camera.metering.matrix")
                .font(.system(size: 14, weight: .bold))
                .foregroundColor(Color(red: 1.0, green: 0.75, blue: 0.20))
            
            VStack(alignment: .leading, spacing: 2) {
                Text("Direct Angle Detected")
                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                    .foregroundColor(.white)
                
                Text("Frontal view compresses neck depth. Angle camera 30°–60° (or turn slightly) for optimal 3D posture analysis.")
                    .font(.system(size: 9.5, weight: .regular))
                    .foregroundColor(.white.opacity(0.80))
            }
            
            Spacer()
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 8)
        .background(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .fill(Color(red: 0.12, green: 0.10, blue: 0.06).opacity(0.92))
                .shadow(color: Color.black.opacity(0.4), radius: 6, x: 0, y: 3)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .stroke(Color(red: 1.0, green: 0.75, blue: 0.20).opacity(0.55), lineWidth: 1.0)
        )
        .frame(maxWidth: 520)
    }
}

// MARK: - 2D Skeleton Lines & Landmarks Overlay Canvas

struct SkeletonOverlayCanvas: View {
    let kinematics: SkeletalKinematicState
    let viewSize: CGSize
    let viewMode: CameraViewMode
    var isMirrored: Bool = false
    
    var body: some View {
        Canvas { context, size in
            let videoRect = computeVideoRect(in: size, mode: viewMode)
            
            // Coordinate helper: Vision (0,0 bottom-left of video frame) to View (0,0 top-left of window)
            // Properly accounts for camera horizontal mirroring
            func toViewCoord(_ pt: CGPoint) -> CGPoint {
                let mapped = isMirrored ? CGPoint(x: 1.0 - pt.x, y: pt.y) : pt
                return convertVisionPointToView(mapped, in: videoRect)
            }
            
            let statusCol = postureColor(kinematics.posture.status)
            let clavicleColor = Color(red: 0.20, green: 0.85, blue: 0.50)
            let jawlineColor = Color(red: 1.0, green: 0.78, blue: 0.28)
            let cvaLineColor = statusCol
            
            let vT = kinematics.keypoints.tragus.map { toViewCoord($0) }
            let vA = kinematics.keypoints.acromion.map { toViewCoord($0) }
            let vJ = kinematics.keypoints.jugularNotch.map { toViewCoord($0) }
            let vC7 = kinematics.keypoints.c7.map { toViewCoord($0) }
            let vChin = (kinematics.face.chinGnathionPoint != .zero) ? toViewCoord(kinematics.face.chinGnathionPoint) : nil
            
            // 1. Draw Dynamic Body Outline (Apple Vision Segmentation Silhouette)
            if kinematics.silhouetteContour.count >= 10 {
                var outlinePath = Path()
                let firstPt = toViewCoord(kinematics.silhouetteContour[0])
                outlinePath.move(to: firstPt)
                for i in 1..<kinematics.silhouetteContour.count {
                    outlinePath.addLine(to: toViewCoord(kinematics.silhouetteContour[i]))
                }
                outlinePath.closeSubpath()
                
                // Ambient outer glow
                context.stroke(
                    outlinePath,
                    with: .color(Color(red: 0.0, green: 0.95, blue: 0.85).opacity(0.18)),
                    lineWidth: 5.0
                )
                // Crisp contour edge
                context.stroke(
                    outlinePath,
                    with: .color(Color(red: 0.0, green: 0.95, blue: 0.85).opacity(0.65)),
                    style: StrokeStyle(lineWidth: 1.8, lineCap: .round, lineJoin: .round)
                )
            }
            
            // 2. Draw Mandibular Jawline Contour (Tragus to Chin)
            let jawlineIndices: [Int] = {
                guard kinematics.face.jawlinePoints.count >= 17 else {
                    return Array(kinematics.face.jawlinePoints.indices)
                }
                let yawMag = abs(kinematics.face.anatomicalYawDegrees != 0 ? kinematics.face.anatomicalYawDegrees : kinematics.face.yawDegrees)
                if yawMag < 22.0 {
                    // Frontal: complete mandibular jawline from ear to ear
                    return Array(2...14)
                } else if kinematics.facingSign < 0 {
                    // Facing camera-left (right profile visible): right ear to chin
                    return Array(2...8)
                } else {
                    // Facing camera-right (left profile visible): chin to left ear
                    return Array(8...14)
                }
            }()
            
            if jawlineIndices.count >= 2 {
                var jawPath = Path()
                let firstPt = toViewCoord(kinematics.face.jawlinePoints[jawlineIndices[0]])
                jawPath.move(to: firstPt)
                for idx in jawlineIndices.dropFirst() {
                    jawPath.addLine(to: toViewCoord(kinematics.face.jawlinePoints[idx]))
                }
                
                // Outer glow
                context.stroke(
                    jawPath,
                    with: .color(jawlineColor.opacity(0.35)),
                    lineWidth: 5.5
                )
                // Crisp core stroke
                context.stroke(
                    jawPath,
                    with: .color(jawlineColor.opacity(0.90)),
                    style: StrokeStyle(lineWidth: 2.4, lineCap: .round, lineJoin: .round)
                )
            } else if let t = vT, let chin = vChin {
                // Fallback direct mandibular chord if discrete contour points are absent
                var chordPath = Path()
                chordPath.move(to: t)
                chordPath.addLine(to: chin)
                context.stroke(chordPath, with: .color(jawlineColor.opacity(0.35)), lineWidth: 5.0)
                context.stroke(chordPath, with: .color(jawlineColor.opacity(0.85)), style: StrokeStyle(lineWidth: 2.2, lineCap: .round))
            }
            
            // 3. Draw Clavicles / Xương Quai Xanh (From Jugular Notch J to Acromion A and far shoulder)
            if let j = vJ {
                // Primary near clavicle: from Jugular Notch J to Acromion A
                if let a = vA {
                    drawClavicle(context: &context, from: j, to: a, color: clavicleColor)
                }
                // Contralateral clavicle: from J to the other detected shoulder (only if high confidence)
                let farShoulder: CGPoint? = {
                    guard let l = kinematics.body.leftShoulder,
                          let r = kinematics.body.rightShoulder,
                          l.confidence > 0.35, r.confidence > 0.35 else { return nil }
                    let nearPt = kinematics.keypoints.acromion
                    return (l.point == nearPt) ? r.point : l.point
                }()
                if let far = farShoulder {
                    let vFar = toViewCoord(far)
                    drawClavicle(context: &context, from: j, to: vFar, color: clavicleColor.opacity(0.55))
                }
            }
            
            // 4. Draw CVA Line (C7 to T) and Horizontal Baseline at C7
            if let c7 = vC7, let t = vT {
                // Horizontal reference baseline through C7
                let baselineLen: CGFloat = 55.0
                var baselinePath = Path()
                baselinePath.move(to: CGPoint(x: c7.x - baselineLen, y: c7.y))
                baselinePath.addLine(to: CGPoint(x: c7.x + baselineLen, y: c7.y))
                context.stroke(baselinePath, with: .color(Color.white.opacity(0.40)), style: StrokeStyle(lineWidth: 1.4, dash: [4, 4]))
                
                // C7 -> T line
                var cvaPath = Path()
                cvaPath.move(to: c7)
                cvaPath.addLine(to: t)
                // Outer glow
                context.stroke(cvaPath, with: .color(cvaLineColor.opacity(0.3)), lineWidth: 5.0)
                // Solid core
                context.stroke(cvaPath, with: .color(cvaLineColor), style: StrokeStyle(lineWidth: 2.6, lineCap: .round))
            }
            
            // 5. Draw Landmark Indicators & Badges (T, Chin, A, J, C7)
            if let t = vT {
                drawLandmarkDot(context: &context, at: t, color: .cyan, radius: 5.0)
                drawBadge(context: &context, text: "T (Tragus)", at: CGPoint(x: t.x - 36, y: t.y - 12), color: .cyan)
            }
            
            if let chin = vChin {
                drawLandmarkDot(context: &context, at: chin, color: jawlineColor, radius: 4.5)
                drawBadge(context: &context, text: "Chin", at: CGPoint(x: chin.x, y: chin.y + 14), color: jawlineColor)
            }
            
            if let c7 = vC7 {
                let c7Label = kinematics.keypoints.isC7Estimated ? "C7 (est)" : "C7"
                let c7Col = kinematics.keypoints.isC7Estimated ? Color.orange : Color(red: 0.18, green: 0.80, blue: 0.44)
                drawLandmarkDot(context: &context, at: c7, color: c7Col, radius: 5.5)
                drawBadge(context: &context, text: c7Label, at: CGPoint(x: c7.x + 24, y: c7.y + 12), color: c7Col)
            }
            
            if let a = vA {
                drawLandmarkDot(context: &context, at: a, color: Color(red: 0.20, green: 0.85, blue: 0.50), radius: 5.0)
                drawBadge(context: &context, text: "A (Acromion)", at: CGPoint(x: a.x, y: a.y - 16), color: Color(red: 0.20, green: 0.85, blue: 0.50))
            }
            
            if let j = vJ {
                drawLandmarkDot(context: &context, at: j, color: .white, radius: 4.5)
                drawBadge(context: &context, text: "J (Jugular notch)", at: CGPoint(x: j.x, y: j.y + 14), color: .white)
            }
        }
    }
    
    // MARK: - Drawing Helpers
    
    private func drawClavicle(context: inout GraphicsContext, from: CGPoint, to: CGPoint, color: Color) {
        let ctrl = CGPoint(x: (from.x + to.x) * 0.5, y: min(from.y, to.y) - 4.0)
        var path = Path()
        path.move(to: from)
        path.addQuadCurve(to: to, control: ctrl)
        
        // Outer glow
        context.stroke(path, with: .color(color.opacity(0.30)), lineWidth: 6.0)
        // Solid core
        context.stroke(path, with: .color(color), style: StrokeStyle(lineWidth: 3.0, lineCap: .round, lineJoin: .round))
    }
    
    private func drawLandmarkDot(context: inout GraphicsContext, at pt: CGPoint, color: Color, radius: CGFloat) {
        // Outer glow
        context.fill(Circle().path(in: CGRect(x: pt.x - radius - 2, y: pt.y - radius - 2, width: (radius + 2) * 2, height: (radius + 2) * 2)), with: .color(color.opacity(0.35)))
        // Inner solid
        context.fill(Circle().path(in: CGRect(x: pt.x - radius, y: pt.y - radius, width: radius * 2, height: radius * 2)), with: .color(color))
    }
    
    private func drawBadge(context: inout GraphicsContext, text: String, at pt: CGPoint, color: Color) {
        let approxWidth = CGFloat(text.count) * 6.2 + 10.0
        let badgeRect = CGRect(x: pt.x - approxWidth * 0.5, y: pt.y - 8.0, width: approxWidth, height: 16.0)
        
        // Pill background
        context.fill(Path(roundedRect: badgeRect, cornerRadius: 4), with: .color(Color.black.opacity(0.75)))
        context.stroke(Path(roundedRect: badgeRect, cornerRadius: 4), with: .color(color.opacity(0.6)), lineWidth: 0.8)
        
        // Text
        context.draw(
            Text(text)
                .font(.system(size: 9, weight: .bold, design: .monospaced))
                .foregroundColor(color),
            at: pt
        )
    }
    
    private func postureColor(_ status: PostureStatus) -> Color {
        switch status {
        case .optimal: return Color(red: 0.20, green: 0.85, blue: 0.50)
        case .caution: return Color(red: 1.0, green: 0.75, blue: 0.20)
        case .turtleNeck: return Color(red: 0.95, green: 0.25, blue: 0.25)
        case .slouching: return Color(red: 1.0, green: 0.55, blue: 0.20)
        case .overextended: return Color(red: 0.85, green: 0.40, blue: 0.95)
        }
    }
}

// MARK: - AppKit Camera Preview Representable

final class CameraVideoView: NSView {
    let previewLayer: AVCaptureVideoPreviewLayer
    private var isMirrored: Bool
    private var sessionObserver: NSObjectProtocol?
    private var isDisposed = false
    private var retryCount = 0
    
    init(session: AVCaptureSession, videoGravity: AVLayerVideoGravity, isMirrored: Bool = false) {
        self.previewLayer = AVCaptureVideoPreviewLayer(session: session)
        self.previewLayer.videoGravity = videoGravity
        self.isMirrored = isMirrored
        super.init(frame: .zero)
        self.wantsLayer = true
        self.layer?.masksToBounds = true
        self.previewLayer.masksToBounds = true
        self.layer?.addSublayer(previewLayer)
        
        applyMirroring()
        
        // Listen for session starting so mirroring is immediately applied once previewLayer has connection
        sessionObserver = NotificationCenter.default.addObserver(
            forName: .AVCaptureSessionDidStartRunning,
            object: session,
            queue: .main
        ) { [weak self] _ in
            self?.retryCount = 0
            self?.applyMirroring()
        }
    }
    
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }
    
    deinit {
        isDisposed = true
        if let observer = sessionObserver {
            NotificationCenter.default.removeObserver(observer)
        }
    }
    
    override func layout() {
        super.layout()
        CATransaction.begin()
        CATransaction.setDisableActions(true)
        previewLayer.frame = bounds
        CATransaction.commit()
        applyMirroring()
    }
    
    func update(videoGravity: AVLayerVideoGravity, isMirrored: Bool) {
        if previewLayer.videoGravity != videoGravity {
            previewLayer.videoGravity = videoGravity
        }
        if self.isMirrored != isMirrored {
            self.isMirrored = isMirrored
            self.retryCount = 0
        }
        applyMirroring()
    }
    
    private func applyMirroring() {
        guard !isDisposed else { return }
        guard let connection = previewLayer.connection else {
            if retryCount < 20 {
                retryCount += 1
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) { [weak self] in
                    self?.applyMirroring()
                }
            }
            return
        }
        retryCount = 0
        if connection.isVideoMirroringSupported {
            connection.automaticallyAdjustsVideoMirroring = false
            if connection.isVideoMirrored != isMirrored {
                connection.isVideoMirrored = isMirrored
            }
        }
    }
}

struct CameraPreviewRepresentable: NSViewRepresentable {
    let session: AVCaptureSession
    let videoGravity: AVLayerVideoGravity
    var isMirrored: Bool = false
    
    func makeNSView(context: Context) -> CameraVideoView {
        CameraVideoView(session: session, videoGravity: videoGravity, isMirrored: isMirrored)
    }
    
    func updateNSView(_ nsView: CameraVideoView, context: Context) {
        nsView.update(videoGravity: videoGravity, isMirrored: isMirrored)
    }
}
