import SwiftUI
import AppKit

/// Floating utility window controller for Posturify.
/// Manages a borderless floating panel positioned neatly in the top-right corner of the active screen.
@MainActor
public final class WindowManager: NSObject, ObservableObject, NSWindowDelegate {
    public static let shared = WindowManager()
    
    private var window: NSPanel?
    private weak var activeAppState: AppState?
    
    public override init() {
        super.init()
    }
    
    /// Toggles visibility of the top-right aligned utility window
    public func toggleWindow(appState: AppState) {
        if let win = window, win.isVisible {
            closeWindow()
        } else {
            showWindow(appState: appState)
        }
    }
    
    /// Displays and positions the window at the top-right of the current visible screen frame
    public func showWindow(appState: AppState) {
        self.activeAppState = appState
        appState.isWindowVisible = true
        
        if window == nil {
            createWindow(appState: appState)
        }
        
        guard let win = window else { return }
        
        positionTopRight(win: win)
        win.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
    }
    
    public func closeWindow() {
        window?.orderOut(nil)
        activeAppState?.isWindowVisible = false
    }
    
    public func windowWillClose(_ notification: Notification) {
        activeAppState?.isWindowVisible = false
    }
    
    private func createWindow(appState: AppState) {
        let panel = NSPanel(
            contentRect: NSRect(x: 0, y: 0, width: 920, height: 560),
            styleMask: [.borderless, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )
        
        panel.title = "Posturify"
        panel.isFloatingPanel = true
        panel.level = .floating
        panel.isMovableByWindowBackground = true
        panel.backgroundColor = .clear
        panel.isOpaque = false
        panel.hasShadow = true
        panel.hidesOnDeactivate = false
        panel.delegate = self
        
        let contentView = NSHostingView(
            rootView: DualDisplayMainView(appState: appState)
                .preferredColorScheme(.dark)
        )
        panel.contentView = contentView
        self.window = panel
    }
    
    private func positionTopRight(win: NSPanel) {
        // Find screen with cursor, or primary screen
        let screen = NSScreen.screens.first { screen in
            NSMouseInRect(NSEvent.mouseLocation, screen.frame, false)
        } ?? NSScreen.main ?? NSScreen.screens.first
        
        guard let targetScreen = screen else { return }
        
        let visibleFrame = targetScreen.visibleFrame
        let padding: CGFloat = 16
        
        let x = visibleFrame.maxX - win.frame.width - padding
        let y = visibleFrame.maxY - win.frame.height - padding
        
        win.setFrameOrigin(NSPoint(x: x, y: y))
    }
}
