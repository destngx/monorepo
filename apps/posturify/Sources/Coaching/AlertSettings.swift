import Foundation

/// User preferences for every alert channel, persisted as one value.
///
/// Decoding falls back to the default for any missing or unreadable field, so adding a setting never resets the others.
public struct AlertSettings: Equatable, Codable, Sendable {
    // Posture notifications
    public var notificationsEnabled = true
    /// How long mild forward drift must persist before a notification.
    public var notifyAfter: NotifyDelay = .thirtySeconds
    /// Play a sound once when severe drift is still held after a notification.
    public var severeSoundEnabled = true

    // Movement reminders
    public var breakRemindersEnabled = true
    /// One of `BreakMonitor.intervalOptions`.
    public var breakInterval: TimeInterval = BreakMonitor.defaultInterval

    // Screen-edge glow
    public var edgeGlowEnabled = true
    public var edgeGlow = EdgeGlowStyle()

    public init() {}

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        func value<T: Decodable>(_ key: CodingKeys, _ fallback: T) -> T {
            (try? container.decodeIfPresent(T.self, forKey: key)) ?? fallback
        }
        let defaults = AlertSettings()
        notificationsEnabled = value(.notificationsEnabled, defaults.notificationsEnabled)
        notifyAfter = value(.notifyAfter, defaults.notifyAfter)
        severeSoundEnabled = value(.severeSoundEnabled, defaults.severeSoundEnabled)
        breakRemindersEnabled = value(.breakRemindersEnabled, defaults.breakRemindersEnabled)
        let interval = value(.breakInterval, defaults.breakInterval)
        breakInterval = BreakMonitor.intervalOptions.contains(interval) ? interval : defaults.breakInterval
        edgeGlowEnabled = value(.edgeGlowEnabled, defaults.edgeGlowEnabled)
        edgeGlow = value(.edgeGlow, defaults.edgeGlow)
    }
}

public enum NotifyDelay: Int, CaseIterable, Identifiable, Codable, Sendable {
    case fifteenSeconds = 15
    case thirtySeconds = 30
    case oneMinute = 60
    case twoMinutes = 120

    public var id: Self { self }
    public var seconds: TimeInterval { TimeInterval(rawValue) }

    public var shortName: String {
        rawValue < 60 ? "\(rawValue)s" : "\(rawValue / 60)m"
    }
}

/// Look of the screen-edge glow. Every palette keeps distinct colors per alert kind, so severity stays readable.
public struct EdgeGlowStyle: Equatable, Codable, Sendable {
    public var intensity: GlowIntensity = .balanced
    public var palette: GlowPalette = .warm
    /// Show a short hint label at the top of the screen with the glow.
    public var showsHint = true

    public init(intensity: GlowIntensity = .balanced, palette: GlowPalette = .warm, showsHint: Bool = true) {
        self.intensity = intensity
        self.palette = palette
        self.showsHint = showsHint
    }

    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        let defaults = EdgeGlowStyle()
        intensity = (try? container.decodeIfPresent(GlowIntensity.self, forKey: .intensity)) ?? defaults.intensity
        palette = (try? container.decodeIfPresent(GlowPalette.self, forKey: .palette)) ?? defaults.palette
        showsHint = (try? container.decodeIfPresent(Bool.self, forKey: .showsHint)) ?? defaults.showsHint
    }
}

public enum GlowIntensity: String, CaseIterable, Identifiable, Codable, Sendable {
    case subtle
    case balanced
    case strong

    public var id: Self { self }

    public var displayName: String {
        switch self {
        case .subtle: return "Subtle"
        case .balanced: return "Balanced"
        case .strong: return "Strong"
        }
    }
}

public enum GlowPalette: String, CaseIterable, Identifiable, Codable, Sendable {
    case warm
    case sunset
    case cool
    case neon

    public var id: Self { self }

    public var displayName: String {
        switch self {
        case .warm: return "Warm"
        case .sunset: return "Sunset"
        case .cool: return "Cool"
        case .neon: return "Neon"
        }
    }

    /// sRGB components for an alert kind.
    public func rgb(for kind: PostureAlertKind) -> (red: Double, green: Double, blue: Double) {
        switch (self, kind) {
        case (.warm, .mildForward): return (1.00, 0.62, 0.10)
        case (.warm, .severeForward): return (1.00, 0.18, 0.22)
        case (.warm, .overextension): return (0.78, 0.36, 1.00)
        case (.sunset, .mildForward): return (1.00, 0.76, 0.32)
        case (.sunset, .severeForward): return (1.00, 0.30, 0.55)
        case (.sunset, .overextension): return (0.66, 0.52, 1.00)
        case (.cool, .mildForward): return (0.20, 0.82, 1.00)
        case (.cool, .severeForward): return (0.30, 0.42, 1.00)
        case (.cool, .overextension): return (0.62, 0.40, 1.00)
        case (.neon, .mildForward): return (0.72, 1.00, 0.20)
        case (.neon, .severeForward): return (1.00, 0.16, 0.78)
        case (.neon, .overextension): return (0.25, 0.95, 1.00)
        }
    }
}
