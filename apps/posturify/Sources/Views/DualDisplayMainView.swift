import SwiftUI
import AVFoundation

/// Unified native MenuBar Popover view for MacSkeletal3D.
/// Hosts the live camera feed with 2D skeleton overlay and real-time biometric telemetry.
public struct DualDisplayMainView: View {
    @ObservedObject var appState: AppState
    @State private var isShowingNeckDepthPopover: Bool = false
    
    public init(appState: AppState) {
        self.appState = appState
    }
    
    public var body: some View {
        VStack(spacing: 0) {
            // Global Top Navigation & Controls Toolbar
            topNavigationToolbar
                .frame(height: 44)
                .background(Color.black.opacity(0.18))
            
            // Unified Camera Viewport (Live camera feed with 2D skeleton and posture telemetry)
            Display1CameraView(appState: appState)
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 16, style: .continuous)
                        .stroke(Color.white.opacity(0.10), lineWidth: 1)
                )
                .padding(.horizontal, 10)
                .padding(.vertical, 6)
            
            // Bottom Clinical Telemetry & Guidance Strip
            bottomStatusToolbar
                .frame(height: 34)
                .background(Color.black.opacity(0.18))
        }
        .frame(width: 920, height: 560)
        .background(Color(red: 0.08, green: 0.10, blue: 0.14))
        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .stroke(Color.white.opacity(0.12), lineWidth: 1)
        )
    }
    
    // MARK: - Top Navigation Toolbar
    
    private var topNavigationToolbar: some View {
        HStack(spacing: 10) {
            // App Branding
            HStack(spacing: 7) {
                Image(systemName: "figure.walk.motion")
                    .font(.system(size: 13, weight: .bold))
                    .foregroundColor(Color(red: 0.20, green: 0.85, blue: 0.48))
                
                Text("Posturify")
                    .font(.system(size: 12, weight: .bold, design: .monospaced))
                    .foregroundColor(.white)
            }
            
            Spacer()
            
            // Camera Device Menu
            Menu {
                if appState.cameraManager.availableCameras.isEmpty {
                    Text("No Cameras Detected").foregroundColor(.secondary)
                } else {
                    ForEach(appState.cameraManager.availableCameras, id: \.uniqueID) { camera in
                        Button(action: {
                            appState.switchCamera(to: camera)
                        }) {
                            HStack {
                                Text(camera.localizedName)
                                if appState.selectedCameraID == camera.uniqueID {
                                    Image(systemName: "checkmark")
                                }
                            }
                        }
                    }
                }
                
                Divider()
                
                Button("Refresh Connected Cameras") {
                    appState.cameraManager.refreshAvailableCameras()
                }
            } label: {
                HStack(spacing: 4) {
                    Image(systemName: "video.fill")
                        .font(.system(size: 10))
                    Text(appState.cameraManager.currentCamera?.localizedName ?? "Camera")
                        .font(.system(size: 11, weight: .medium))
                        .lineLimit(1)
                        .truncationMode(.tail)
                        .frame(maxWidth: 110, alignment: .leading)
                }
                .foregroundColor(.white.opacity(0.85))
                .padding(.horizontal, 7)
                .padding(.vertical, 4)
                .background(Color.white.opacity(0.08))
                .clipShape(RoundedRectangle(cornerRadius: 6))
            }
            .menuStyle(.borderlessButton)
            .fixedSize()
            
            // Viewport Framing Click Options (Icon-only with tooltips)
            HStack(spacing: 2) {
                Button(action: {
                    appState.cameraViewMode = .openView
                }) {
                    Image(systemName: "aspectratio")
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundColor(appState.cameraViewMode == .openView ? .white : .white.opacity(0.55))
                        .frame(width: 24, height: 24)
                        .background(appState.cameraViewMode == .openView ? Color.white.opacity(0.20) : Color.clear)
                        .clipShape(RoundedRectangle(cornerRadius: 5))
                }
                .buttonStyle(.plain)
                .help("Open View (16:9 full sensor FOV)")
                
                Button(action: {
                    appState.cameraViewMode = .fillView
                }) {
                    Image(systemName: "arrow.up.left.and.arrow.down.right")
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundColor(appState.cameraViewMode == .fillView ? .white : .white.opacity(0.55))
                        .frame(width: 24, height: 24)
                        .background(appState.cameraViewMode == .fillView ? Color.white.opacity(0.20) : Color.clear)
                        .clipShape(RoundedRectangle(cornerRadius: 5))
                }
                .buttonStyle(.plain)
                .help("Fill Frame (Cropped to view)")
            }
            .padding(2)
            .background(Color.black.opacity(0.35))
            .clipShape(RoundedRectangle(cornerRadius: 6))
            .overlay(
                RoundedRectangle(cornerRadius: 6)
                    .stroke(Color.white.opacity(0.12), lineWidth: 0.8)
            )
            
            // Flip / Mirror Camera Button (Icon-only with tooltip)
            Button(action: {
                appState.toggleCameraMirrored()
            }) {
                Image(systemName: "arrow.left.and.right.righttriangle.left.righttriangle.right")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundColor(appState.isCameraMirrored ? .cyan : .white.opacity(0.85))
                    .frame(width: 26, height: 26)
                    .background(Color.white.opacity(0.08))
                    .clipShape(RoundedRectangle(cornerRadius: 6))
            }
            .buttonStyle(.plain)
            .help(appState.isCameraMirrored ? "Camera Mirrored (Click to switch to Normal)" : "Camera Normal (Click to switch to Mirrored)")
            
            // Interactive Relative CVA Calibration Button (Icon-only with tooltip)
            Button(action: {
                if appState.isCalibrated {
                    appState.resetCalibration()
                } else {
                    appState.triggerCalibration()
                }
            }) {
                Image(systemName: appState.isCalibrated ? "checkmark.circle.fill" : "target")
                    .font(.system(size: 12, weight: .bold))
                    .foregroundColor(appState.isCalibrated ? Color(red: 0.18, green: 0.80, blue: 0.44) : (appState.calibrationEngine.currentPhase == .idle ? .cyan : .orange))
                    .frame(width: 26, height: 26)
                    .background(Color.white.opacity(0.08))
                    .clipShape(RoundedRectangle(cornerRadius: 6))
            }
            .buttonStyle(.plain)
            .help(appState.isCalibrated ? "Calibrated (Relative CVA Active). Click to re-calibrate." : "Posture Calibration: \(appState.calibrationPhaseText). Click to run 2s calibration.")
            
            // Neck / Collar Depth Ratio Slider Popover Button
            neckDepthSliderButton
            
            // Close Panel Button
            Button(action: {
                WindowManager.shared.closeWindow()
            }) {
                Image(systemName: "xmark")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundColor(.white.opacity(0.75))
                    .frame(width: 26, height: 26)
                    .background(Color.white.opacity(0.08))
                    .clipShape(RoundedRectangle(cornerRadius: 6))
            }
            .buttonStyle(.plain)
            .help("Close Window (⌘W)")
            .keyboardShortcut("w", modifiers: [.command])
            
            // Quit Button
            Button(action: {
                NSApp.terminate(nil)
            }) {
                Image(systemName: "power")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundColor(Color.red.opacity(0.85))
                    .frame(width: 26, height: 26)
                    .background(Color.red.opacity(0.12))
                    .clipShape(RoundedRectangle(cornerRadius: 6))
            }
            .buttonStyle(.plain)
            .help("Quit Posturify (⌘Q)")
            .keyboardShortcut("q", modifiers: [.command])
        }
        .padding(.horizontal, 12)
    }
    
    // MARK: - Neck / Collar Depth Ratio Slider Popover
    
    private var neckDepthSliderButton: some View {
        Button(action: {
            isShowingNeckDepthPopover.toggle()
        }) {
            Image(systemName: "slider.vertical.3")
                .font(.system(size: 11, weight: .semibold))
                .foregroundColor((appState.jDepthRatio > 0.20 || appState.c7DepthRatio != 0.15) ? .cyan : .white.opacity(0.85))
                .frame(width: 26, height: 26)
                .background((appState.jDepthRatio > 0.20 || appState.c7DepthRatio != 0.15) ? Color.cyan.opacity(0.18) : Color.white.opacity(0.08))
                .clipShape(RoundedRectangle(cornerRadius: 6))
        }
        .buttonStyle(.plain)
        .help("Adjust Jugular Notch (J) and C7 Landmark Positions")
        .popover(isPresented: $isShowingNeckDepthPopover, arrowEdge: .bottom) {
            VStack(alignment: .leading, spacing: 12) {
                // Section 1: Jugular Notch (J) Offset
                VStack(alignment: .leading, spacing: 5) {
                    HStack {
                        Text("J Point (Jugular Notch)")
                            .font(.system(size: 11, weight: .bold, design: .monospaced))
                            .foregroundColor(.white)
                        Spacer()
                        HStack(spacing: 4) {
                            if (0.35...0.55).contains(appState.jDepthRatio) || (0.12...0.18).contains(appState.jDepthRatio) {
                                Text("Rec")
                                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                    .padding(.horizontal, 4)
                                    .padding(.vertical, 1)
                                    .background(Color(red: 0.20, green: 0.85, blue: 0.48).opacity(0.20))
                                    .foregroundColor(Color(red: 0.20, green: 0.85, blue: 0.48))
                                    .clipShape(RoundedRectangle(cornerRadius: 3))
                            }
                            Text(String(format: "%.2f×", appState.jDepthRatio))
                                .font(.system(size: 11, weight: .bold, design: .monospaced))
                                .foregroundColor(isJRecommended(appState.jDepthRatio) ? Color(red: 0.20, green: 0.85, blue: 0.48) : .cyan)
                        }
                    }
                    
                    Slider(value: $appState.jDepthRatio, in: 0.10...0.90, step: 0.05)
                        .tint(isJRecommended(appState.jDepthRatio) ? Color(red: 0.20, green: 0.85, blue: 0.48) : .cyan)
                    
                    // Track Bar indicating Recommended Zones (Bare: ~0.15, Collar: 0.35-0.55)
                    GeometryReader { geo in
                        ZStack(alignment: .leading) {
                            // Background track guide
                            RoundedRectangle(cornerRadius: 2)
                                .fill(Color.white.opacity(0.12))
                                .frame(height: 3)
                            
                            // Bare Neck recommended zone (~0.12 - 0.18) -> normalized (0.15 - 0.10)/0.80 = 6%
                            RoundedRectangle(cornerRadius: 2)
                                .fill(Color(red: 0.20, green: 0.85, blue: 0.48).opacity(0.85))
                                .frame(width: max(4, geo.size.width * (0.08 / 0.80)), height: 3)
                                .offset(x: geo.size.width * ((0.12 - 0.10) / 0.80))
                            
                            // Collar recommended zone (0.35 - 0.55) -> normalized (0.35 - 0.10)/0.80 = 31% to 56%
                            RoundedRectangle(cornerRadius: 2)
                                .fill(Color(red: 0.20, green: 0.85, blue: 0.48).opacity(0.85))
                                .frame(width: geo.size.width * ((0.55 - 0.35) / 0.80), height: 3)
                                .offset(x: geo.size.width * ((0.35 - 0.10) / 0.80))
                        }
                    }
                    .frame(height: 3)
                    
                    HStack {
                        Text("0.15 (Bare)")
                            .font(.system(size: 8.5, design: .monospaced))
                            .foregroundColor(isBareJ(appState.jDepthRatio) ? Color(red: 0.20, green: 0.85, blue: 0.48) : .white.opacity(0.50))
                        Spacer()
                        Text("0.35–0.55 (Collar Rec)")
                            .font(.system(size: 8.5, design: .monospaced))
                            .foregroundColor(isCollarJ(appState.jDepthRatio) ? Color(red: 0.20, green: 0.85, blue: 0.48) : .white.opacity(0.50))
                    }
                }
                
                Divider().background(Color.white.opacity(0.15))
                
                // Section 2: C7 (Vertebra) Depth Offset
                VStack(alignment: .leading, spacing: 5) {
                    HStack {
                        Text("C7 (Posterior Neck)")
                            .font(.system(size: 11, weight: .bold, design: .monospaced))
                            .foregroundColor(.white)
                        Spacer()
                        HStack(spacing: 4) {
                            if (0.12...0.20).contains(appState.c7DepthRatio) {
                                Text("Rec")
                                    .font(.system(size: 8.5, weight: .bold, design: .monospaced))
                                    .padding(.horizontal, 4)
                                    .padding(.vertical, 1)
                                    .background(Color(red: 0.20, green: 0.85, blue: 0.48).opacity(0.20))
                                    .foregroundColor(Color(red: 0.20, green: 0.85, blue: 0.48))
                                    .clipShape(RoundedRectangle(cornerRadius: 3))
                            }
                            Text(String(format: "%.2f×", appState.c7DepthRatio))
                                .font(.system(size: 11, weight: .bold, design: .monospaced))
                                .foregroundColor(isC7Recommended(appState.c7DepthRatio) ? Color(red: 0.20, green: 0.85, blue: 0.48) : .orange)
                        }
                    }
                    
                    Slider(value: $appState.c7DepthRatio, in: 0.10...0.50, step: 0.05)
                        .tint(isC7Recommended(appState.c7DepthRatio) ? Color(red: 0.20, green: 0.85, blue: 0.48) : .orange)
                    
                    // Track Bar indicating Recommended Zone for C7 (0.12 - 0.20)
                    GeometryReader { geo in
                        ZStack(alignment: .leading) {
                            RoundedRectangle(cornerRadius: 2)
                                .fill(Color.white.opacity(0.12))
                                .frame(height: 3)
                            
                            // Recommended zone (0.12 - 0.20)
                            RoundedRectangle(cornerRadius: 2)
                                .fill(Color(red: 0.20, green: 0.85, blue: 0.48).opacity(0.85))
                                .frame(width: geo.size.width * ((0.20 - 0.12) / 0.40), height: 3)
                                .offset(x: geo.size.width * ((0.12 - 0.10) / 0.40))
                        }
                    }
                    .frame(height: 3)
                    
                    HStack {
                        Text("0.15 (Optimal Rec)")
                            .font(.system(size: 8.5, design: .monospaced))
                            .foregroundColor(isC7Recommended(appState.c7DepthRatio) ? Color(red: 0.20, green: 0.85, blue: 0.48) : .white.opacity(0.50))
                        Spacer()
                        Text("0.50 (Too Low)")
                            .font(.system(size: 8.5, design: .monospaced))
                            .foregroundColor(.white.opacity(0.50))
                    }
                }
                
                Divider().background(Color.white.opacity(0.15))
                
                // Quick Presets
                HStack(spacing: 8) {
                    Button("Default (0.15)") {
                        appState.jDepthRatio = 0.15
                        appState.c7DepthRatio = 0.15
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                    
                    Button("Collar (J: 0.45)") {
                        appState.jDepthRatio = 0.45
                        appState.c7DepthRatio = 0.15
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                }
            }
            .padding(14)
            .frame(width: 250)
            .background(Color(red: 0.12, green: 0.14, blue: 0.18))
        }
    }
    
    // MARK: - Bottom Status Toolbar
    
    private var bottomStatusToolbar: some View {
        HStack(spacing: 10) {
            // Posture Guidance Cue (Color-coded text only, no icons)
            Text(appState.kinematics.posture.guidanceCue.isEmpty ? "Calibrating posture telemetry..." : appState.kinematics.posture.guidanceCue)
                .font(.system(size: 11, weight: .semibold, design: .monospaced))
                .foregroundColor(
                    appState.kinematics.posture.isDirectAngleFacing
                    ? Color(red: 1.0, green: 0.75, blue: 0.20)
                    : statusColor(appState.kinematics.posture.status)
                )
                .lineLimit(1)
            
            Spacer()
            
            // Biometric Metrics Readout
            HStack(spacing: 8) {
                // Absolute Clinical CVA
                HStack(spacing: 3) {
                    Text("CVA:")
                        .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                        .foregroundColor(.white.opacity(0.6))
                    Text(String(format: "%.1f°", appState.kinematics.posture.cvaDegrees))
                        .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                        .foregroundColor(statusColor(appState.kinematics.posture.status))
                }
                
                Text("•")
                    .foregroundColor(.white.opacity(0.3))
                
                // Relative CVA (Shows '-' when not calibrated, shows Rel CVA + Δ when calibrated)
                HStack(spacing: 3) {
                    Text("Rel CVA:")
                        .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                        .foregroundColor(.white.opacity(0.6))
                    if appState.isCalibrated && appState.kinematics.posture.isRelativeProxy {
                        Text(String(format: "%.1f° (%+.1f°)", appState.kinematics.posture.cvaDegrees, appState.kinematics.posture.deltaCvaDegrees))
                            .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                            .foregroundColor(statusColor(appState.kinematics.posture.status))
                    } else {
                        Text("-")
                            .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                            .foregroundColor(.white.opacity(0.45))
                    }
                }
                
                Text("•")
                    .foregroundColor(.white.opacity(0.3))
                
                Text(String(format: "%.0f FPS", appState.fps))
                    .font(.system(size: 9.5, weight: .bold, design: .monospaced))
                    .foregroundColor(.cyan)
            }
        }
        .padding(.horizontal, 12)
    }
    
    private func statusColor(_ status: PostureStatus) -> Color {
        switch status {
        case .optimal:
            return Color(red: 0.18, green: 0.80, blue: 0.44)
        case .caution:
            return Color(red: 1.0, green: 0.65, blue: 0.15)
        case .turtleNeck:
            return Color(red: 0.95, green: 0.22, blue: 0.22)
        case .slouching:
            return Color(red: 1.0, green: 0.55, blue: 0.20)
        case .overextended:
            return Color(red: 0.85, green: 0.40, blue: 0.95)
        }
    }
    
    // MARK: - Recommended Landmark Range Helpers
    
    private func isBareJ(_ val: Double) -> Bool {
        (0.12...0.18).contains(val)
    }
    
    private func isCollarJ(_ val: Double) -> Bool {
        (0.35...0.55).contains(val)
    }
    
    private func isJRecommended(_ val: Double) -> Bool {
        isBareJ(val) || isCollarJ(val)
    }
    
    private func isC7Recommended(_ val: Double) -> Bool {
        (0.12...0.20).contains(val)
    }
}
