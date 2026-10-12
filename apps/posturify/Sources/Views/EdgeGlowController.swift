import SwiftUI
import AppKit
import QuartzCore

/// Attention-grabbing, click-through glow around the edges of every screen: the ambient first stage of the alert
/// ladder. It never takes focus or intercepts input, and joins every Space.
///
/// Built from Core Animation layers so the animation runs in the render server without waking the app:
/// a soft halo fading in from each edge, a bright rim, a light travelling around the border, a breathing pulse and an
/// entrance flash. Severe alerts are deeper and faster. Reduce Motion keeps only the static glow.
@MainActor
public final class EdgeGlowController {
    public static let shared = EdgeGlowController()

    private var panels: [NSPanel] = []
    private var shown: (kind: PostureAlertKind, style: EdgeGlowStyle)?
    private var screenObserver: NSObjectProtocol?

    public init() {}

    public func show(_ kind: PostureAlertKind?, style: EdgeGlowStyle) {
        guard let kind else {
            hide()
            return
        }
        if let shown, shown.kind == kind, shown.style == style { return }
        shown = (kind, style)
        rebuild()
        observeScreenChanges()
    }

    private func hide() {
        guard shown != nil else { return }
        shown = nil
        panels.forEach { $0.orderOut(nil) }
        panels = []
    }

    private func rebuild() {
        panels.forEach { $0.orderOut(nil) }
        guard let shown else {
            panels = []
            return
        }
        let reduceMotion = NSWorkspace.shared.accessibilityDisplayShouldReduceMotion
        panels = NSScreen.screens.map { screen in
            let panel = makePanel()
            panel.setFrame(screen.frame, display: false)
            let menuBarHeight = screen.frame.maxY - screen.visibleFrame.maxY
            panel.contentView = EdgeGlowLayerView(
                frame: CGRect(origin: .zero, size: screen.frame.size),
                kind: shown.kind,
                style: shown.style,
                topInset: menuBarHeight,
                reduceMotion: reduceMotion
            )
            panel.orderFrontRegardless()
            return panel
        }
    }

    private func observeScreenChanges() {
        guard screenObserver == nil else { return }
        screenObserver = NotificationCenter.default.addObserver(
            forName: NSApplication.didChangeScreenParametersNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            MainActor.assumeIsolated { self?.rebuild() }
        }
    }

    private func makePanel() -> NSPanel {
        let panel = NSPanel(
            contentRect: .zero,
            styleMask: [.borderless, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )
        panel.level = .statusBar
        panel.backgroundColor = .clear
        panel.isOpaque = false
        panel.hasShadow = false
        panel.ignoresMouseEvents = true
        panel.hidesOnDeactivate = false
        panel.isReleasedWhenClosed = false
        panel.collectionBehavior = [.canJoinAllSpaces, .stationary, .ignoresCycle, .fullScreenAuxiliary]
        return panel
    }

    static func nsColor(_ palette: GlowPalette, _ kind: PostureAlertKind) -> NSColor {
        let c = palette.rgb(for: kind)
        return NSColor(srgbRed: c.red, green: c.green, blue: c.blue, alpha: 1)
    }

    static func color(_ palette: GlowPalette, _ kind: PostureAlertKind) -> Color {
        Color(nsColor: nsColor(palette, kind))
    }
}

// MARK: - Appearance Parameters

struct EdgeGlowAppearance: Equatable {
    /// How far the halo reaches into the screen, in points.
    var haloDepth: CGFloat
    var haloOpacity: Float
    var rimWidth: CGFloat
    var cometWidth: CGFloat
    /// Seconds per lap of the travelling light.
    var lapSeconds: CFTimeInterval
    /// Seconds per breath (one direction).
    var breathSeconds: CFTimeInterval
    /// Lowest opacity of the breathing halo, relative to full.
    var breathFloor: Float

    init(kind: PostureAlertKind, intensity: GlowIntensity) {
        switch intensity {
        case .subtle:
            haloDepth = 90; haloOpacity = 0.50; rimWidth = 2; cometWidth = 4
        case .balanced:
            haloDepth = 160; haloOpacity = 0.75; rimWidth = 4; cometWidth = 7
        case .strong:
            haloDepth = 240; haloOpacity = 0.92; rimWidth = 6; cometWidth = 10
        }
        switch kind {
        case .severeForward:
            haloDepth *= 1.3
            lapSeconds = 2.2
            breathSeconds = 0.55
            breathFloor = 0.35
        case .mildForward:
            lapSeconds = 4.5
            breathSeconds = 1.3
            breathFloor = 0.55
        case .overextension:
            lapSeconds = 3.5
            breathSeconds = 1.0
            breathFloor = 0.5
        }
    }
}

// MARK: - Layer View

private final class EdgeGlowLayerView: NSView {
    private static let cornerRadius: CGFloat = 12

    init(frame: CGRect, kind: PostureAlertKind, style: EdgeGlowStyle, topInset: CGFloat, reduceMotion: Bool) {
        super.init(frame: frame)
        wantsLayer = true
        layer = CALayer()
        layer?.backgroundColor = .clear

        let appearance = EdgeGlowAppearance(kind: kind, intensity: style.intensity)
        let color = EdgeGlowController.nsColor(style.palette, kind)
        let scale = NSScreen.screens.first { $0.frame.size == frame.size }?.backingScaleFactor ?? 2

        let glow = CALayer()
        glow.frame = bounds
        layer?.addSublayer(glow)

        let halo = makeHalo(color: color, appearance: appearance)
        glow.addSublayer(halo)
        glow.addSublayer(makeRim(color: color, appearance: appearance, scale: scale))
        if !reduceMotion {
            glow.addSublayer(makeComet(color: color, appearance: appearance))
            halo.add(breathing(appearance), forKey: "breathing")
            glow.add(entranceFlash(), forKey: "entrance")
        }

        if style.showsHint {
            addHint(kind: kind, color: color, topInset: topInset)
        }
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        fatalError("init(coder:) is not supported")
    }

    /// Four linear gradients, each fading from an edge toward the center.
    private func makeHalo(color: NSColor, appearance: EdgeGlowAppearance) -> CALayer {
        let halo = CALayer()
        halo.frame = bounds
        let depth = min(appearance.haloDepth, min(bounds.width, bounds.height) / 3)
        let strong = color.withAlphaComponent(CGFloat(appearance.haloOpacity)).cgColor
        let mid = color.withAlphaComponent(CGFloat(appearance.haloOpacity) * 0.5).cgColor
        let faint = color.withAlphaComponent(CGFloat(appearance.haloOpacity) * 0.15).cgColor
        let clear = color.withAlphaComponent(0).cgColor

        // (frame, start, end) in unit coordinates; layer geometry is not flipped, so y = 1 is the top edge
        let edges: [(CGRect, CGPoint, CGPoint)] = [
            (CGRect(x: 0, y: bounds.height - depth, width: bounds.width, height: depth), CGPoint(x: 0.5, y: 1), CGPoint(x: 0.5, y: 0)),
            (CGRect(x: 0, y: 0, width: bounds.width, height: depth), CGPoint(x: 0.5, y: 0), CGPoint(x: 0.5, y: 1)),
            (CGRect(x: 0, y: 0, width: depth, height: bounds.height), CGPoint(x: 0, y: 0.5), CGPoint(x: 1, y: 0.5)),
            (CGRect(x: bounds.width - depth, y: 0, width: depth, height: bounds.height), CGPoint(x: 1, y: 0.5), CGPoint(x: 0, y: 0.5)),
        ]
        for (frame, start, end) in edges {
            let gradient = CAGradientLayer()
            gradient.frame = frame
            gradient.colors = [strong, mid, faint, clear]
            gradient.locations = [0, 0.25, 0.6, 1]
            gradient.startPoint = start
            gradient.endPoint = end
            halo.addSublayer(gradient)
        }
        return halo
    }

    private var borderPath: CGPath {
        CGPath(roundedRect: bounds, cornerWidth: Self.cornerRadius, cornerHeight: Self.cornerRadius, transform: nil)
    }

    /// Bright line hugging the screen edge with its own small bloom.
    private func makeRim(color: NSColor, appearance: EdgeGlowAppearance, scale: CGFloat) -> CALayer {
        let rim = CAShapeLayer()
        rim.frame = bounds
        rim.path = borderPath
        rim.fillColor = nil
        rim.strokeColor = color.blended(withFraction: 0.25, of: .white)?.cgColor ?? color.cgColor
        // Only the inner half of a stroke centred on the screen edge is visible
        rim.lineWidth = appearance.rimWidth * 2
        rim.shadowColor = color.cgColor
        rim.shadowRadius = appearance.rimWidth * 3
        rim.shadowOpacity = 1
        rim.shadowOffset = .zero
        rim.shouldRasterize = true
        rim.rasterizationScale = scale
        return rim
    }

    /// A light that travels around the border: a rotating conic gradient seen through a border-shaped mask.
    private func makeComet(color: NSColor, appearance: EdgeGlowAppearance) -> CALayer {
        let container = CALayer()
        container.frame = bounds

        let mask = CAShapeLayer()
        mask.frame = bounds
        mask.path = borderPath
        mask.fillColor = nil
        mask.strokeColor = NSColor.black.cgColor
        mask.lineWidth = appearance.cometWidth * 2
        container.mask = mask

        let side = hypot(bounds.width, bounds.height)
        let conic = CAGradientLayer()
        conic.type = .conic
        conic.frame = CGRect(x: bounds.midX - side / 2, y: bounds.midY - side / 2, width: side, height: side)
        conic.startPoint = CGPoint(x: 0.5, y: 0.5)
        conic.endPoint = CGPoint(x: 0.5, y: 0)
        let head = NSColor.white.cgColor
        let body = color.blended(withFraction: 0.35, of: .white)?.cgColor ?? color.cgColor
        let clear = color.withAlphaComponent(0).cgColor
        // Two lights on opposite sides
        conic.colors = [clear, body, head, clear, clear, body, head, clear]
        conic.locations = [0, 0.38, 0.47, 0.5, 0.5, 0.88, 0.97, 1]
        container.addSublayer(conic)

        let spin = CABasicAnimation(keyPath: "transform.rotation.z")
        spin.fromValue = 0
        spin.toValue = -2 * Double.pi
        spin.duration = appearance.lapSeconds
        spin.repeatCount = .infinity
        conic.add(spin, forKey: "spin")
        return container
    }

    private func breathing(_ appearance: EdgeGlowAppearance) -> CAAnimation {
        let breath = CABasicAnimation(keyPath: "opacity")
        breath.fromValue = 1
        breath.toValue = appearance.breathFloor
        breath.duration = appearance.breathSeconds
        breath.autoreverses = true
        breath.repeatCount = .infinity
        breath.timingFunction = CAMediaTimingFunction(name: .easeInEaseOut)
        return breath
    }

    /// Two quick flashes when the glow appears, so the change is noticed in peripheral vision.
    private func entranceFlash() -> CAAnimation {
        let flash = CAKeyframeAnimation(keyPath: "opacity")
        flash.values = [0, 1, 0.25, 1, 0.4, 1]
        flash.keyTimes = [0, 0.15, 0.35, 0.5, 0.7, 1]
        flash.duration = 1.1
        return flash
    }

    private func addHint(kind: PostureAlertKind, color: NSColor, topInset: CGFloat) {
        let hint = NSHostingView(rootView: EdgeGlowHint(kind: kind, color: Color(nsColor: color)))
        let size = hint.fittingSize
        hint.frame = CGRect(
            x: (bounds.width - size.width) / 2,
            y: bounds.height - topInset - size.height - 14,
            width: size.width,
            height: size.height
        )
        addSubview(hint)
    }
}

// MARK: - Hint Label

private struct EdgeGlowHint: View {
    let kind: PostureAlertKind
    let color: Color

    var body: some View {
        HStack(spacing: 8) {
            Image(systemName: icon)
                .font(.system(size: 13, weight: .bold))
                .foregroundStyle(color)
            Text(text)
                .font(.system(size: 13, weight: .semibold, design: .rounded))
                .foregroundStyle(.white)
                .lineLimit(1)
                .fixedSize()
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
        .background(Capsule().fill(Color.black.opacity(0.72)))
        .overlay(Capsule().stroke(color.opacity(0.9), lineWidth: 1.5))
        .shadow(color: color.opacity(0.7), radius: 10)
        .padding(12)
    }

    private var icon: String {
        switch kind {
        case .mildForward: return "arrow.uturn.backward"
        case .severeForward: return "exclamationmark.triangle.fill"
        case .overextension: return "arrow.down.to.line"
        }
    }

    private var text: String {
        switch kind {
        case .mildForward: return "Head drifting forward - sit tall"
        case .severeForward: return "Head far forward - glide your chin back"
        case .overextension: return "Head tilted back - level your gaze"
        }
    }
}
