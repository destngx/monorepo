import SwiftUI
import AppKit

@main
public struct PosturifyApp: App {
    private static let sharedAppState = AppState()
    @StateObject private var appState = PosturifyApp.sharedAppState
    
    public init() {
        // Run purely as an accessory MenuBar app (no dock icon, popover in status bar)
        NSApplication.shared.setActivationPolicy(.accessory)
        
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
            WindowManager.shared.showWindow(appState: PosturifyApp.sharedAppState)
        }
    }
    
    public var body: some Scene {
        // Native MenuBar Status Item that toggles the top-right floating window
        MenuBarExtra {
            Button("Open Posturify") {
                WindowManager.shared.showWindow(appState: appState)
            }
            .keyboardShortcut("o", modifiers: [.command])
            
            Divider()
            
            Button(appState.isCalibrated ? "Calibrated (Click to Re-calibrate)" : "Calibrate Posture") {
                if appState.isCalibrated {
                    appState.resetCalibration()
                } else {
                    appState.triggerCalibration()
                }
            }
            
            Divider()
            
            Button("Quit Posturify") {
                NSApp.terminate(nil)
            }
            .keyboardShortcut("q", modifiers: [.command])
        } label: {
            MenuBarStatusLabel(appState: appState)
        }
    }
}
