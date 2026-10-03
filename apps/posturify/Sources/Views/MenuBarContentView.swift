import SwiftUI
import AppKit

/// Menu bar status label rendered directly in the macOS status tray.
/// Displays dynamic icon and alert badge during Turtle Neck / forward head strain.
public struct MenuBarStatusLabel: View {
    @ObservedObject var appState: AppState
    
    public init(appState: AppState) {
        self.appState = appState
    }
    
    public var body: some View {
        Image(systemName: iconName)
    }
    
    private var iconName: String {
        switch appState.kinematics.posture.status {
        case .optimal:
            return "figure.walk.motion"
        case .caution:
            return "exclamationmark.circle"
        case .turtleNeck:
            return "exclamationmark.triangle.fill"
        case .slouching:
            return "arrow.down.right.and.arrow.up.left"
        case .overextended:
            return "arrow.up.and.line.horizontal.and.arrow.down"
        }
    }
}

/// Menu bar window popover view hosting the unified workspace (Display 1 + Display 2).
public struct MenuBarContentView: View {
    @ObservedObject var appState: AppState
    
    public init(appState: AppState) {
        self.appState = appState
    }
    
    public var body: some View {
        DualDisplayMainView(appState: appState)
    }
}

/// Convenience alias for the unified popover view.
public typealias MenuBarMainPopoverView = DualDisplayMainView
