import Foundation
import UserNotifications
import AppKit
import os

/// Delivers native macOS notifications. What to notify and when is decided by `PostureCoach`.
public final class PostureNotificationManager: NSObject, UNUserNotificationCenterDelegate, @unchecked Sendable {
    public static let shared = PostureNotificationManager()
    
    private static let logger = Logger(subsystem: "com.destngx.posturify", category: "Notifications")
    
    private let center = UNUserNotificationCenter.current()
    
    public override init() {
        super.init()
        center.delegate = self
        requestAuthorization()
    }
    
    /// Requests notification permissions for alerts and sounds
    public func requestAuthorization() {
        center.requestAuthorization(options: [.alert, .sound]) { granted, error in
            if let error = error {
                Self.logger.error("Notification authorization failed: \(error.localizedDescription, privacy: .public)")
            } else if !granted {
                Self.logger.info("Notifications not authorized; the osascript fallback will be used")
            }
        }
    }
    
    /// Posts a native notification. Sound goes through the notification system only, so macOS Focus modes and
    /// notification settings can silence it.
    public func deliver(title: String, subtitle: String = "", body: String, playSound: Bool) {
        let content = UNMutableNotificationContent()
        content.title = title
        content.subtitle = subtitle
        content.body = body
        content.sound = playSound ? .default : nil
        
        let request = UNNotificationRequest(
            identifier: "Posturify-\(UUID().uuidString)",
            content: content,
            trigger: nil
        )
        
        center.add(request) { [weak self] error in
            if let error = error {
                Self.logger.error("Notification delivery rejected (\(error.localizedDescription, privacy: .public)); using osascript fallback")
                self?.deliverSystemNotificationFallback(title: title, subtitle: subtitle, body: body, playSound: playSound)
            }
        }
    }
    
    /// Fallback notification delivery via macOS system notifications (AppleScript)
    /// Guarantees delivery even when binary runs unbundled without code-sign provisioning
    private func deliverSystemNotificationFallback(title: String, subtitle: String, body: String, playSound: Bool) {
        let fullSubtitle = subtitle.isEmpty ? "" : " subtitle \"\(subtitle.replacingOccurrences(of: "\"", with: "\\\""))\""
        let script = "display notification \"\(body.replacingOccurrences(of: "\"", with: "\\\""))\" with title \"\(title.replacingOccurrences(of: "\"", with: "\\\""))\"\(fullSubtitle)\(playSound ? " sound name \"Basso\"" : "")"
        
        do {
            _ = try Process.run(URL(fileURLWithPath: "/usr/bin/osascript"), arguments: ["-e", script]) { process in
                if process.terminationStatus != 0 {
                    Self.logger.error("osascript notification fallback exited with status \(process.terminationStatus)")
                }
            }
        } catch {
            Self.logger.error("osascript notification fallback failed to launch: \(error.localizedDescription, privacy: .public)")
        }
    }
    
    // MARK: - UNUserNotificationCenterDelegate
    
    // Deliver notification even when the app is active in foreground
    public func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        completionHandler([.banner, .sound])
    }
}
