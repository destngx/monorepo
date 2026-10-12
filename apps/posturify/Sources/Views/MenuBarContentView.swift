import SwiftUI
import AppKit

/// Menu bar status label rendered directly in the macOS status tray.
/// Displays dynamic icon and alert badge during Turtle Neck / forward head strain.
public struct MenuBarStatusLabel: View {
    @ObservedObject var appState: AppState
    @ObservedObject var coach: PostureCoach
    
    public init(appState: AppState) {
        self.appState = appState
        self.coach = appState.coach
    }
    
    public var body: some View {
        if let alert = coach.ambientAlert {
            // Menu bar items render as templates; a non-template image keeps the alert tint
            Image(nsImage: Self.tintedSymbol(iconName, color: EdgeGlowController.nsColor(coach.settings.edgeGlow.palette, alert)))
        } else {
            Image(systemName: iconName)
        }
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
    
    private static func tintedSymbol(_ name: String, color: NSColor) -> NSImage {
        let configuration = NSImage.SymbolConfiguration(pointSize: 14, weight: .regular)
            .applying(NSImage.SymbolConfiguration(paletteColors: [color]))
        guard let image = NSImage(systemSymbolName: name, accessibilityDescription: "Posture alert")?
            .withSymbolConfiguration(configuration) else {
            return NSImage()
        }
        image.isTemplate = false
        return image
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
