import Foundation
import UserNotifications
import AppKit

/// Notification manager responsible for requesting permissions and delivering native macOS system notifications
/// when sustained forward head carriage or turtle neck is detected.
public final class PostureNotificationManager: NSObject, UNUserNotificationCenterDelegate, @unchecked Sendable {
    public static let shared = PostureNotificationManager()
    
    private let center = UNUserNotificationCenter.current()
    private var isAuthorized: Bool = false
    
    // Cooldown management: minimum seconds between notifications to avoid spamming the user
    private var lastNotificationTime: TimeInterval = 0.0
    private let notificationCooldown: TimeInterval = 30.0 // 30 seconds cooldown
    
    // Track previous warning level to trigger immediately on transition to Turtle Neck
    private var previousWarningLevel: Int = 0
    
    public override init() {
        super.init()
        center.delegate = self
        requestAuthorization()
    }
    
    /// Requests notification permissions for alerts and sounds
    public func requestAuthorization() {
        center.requestAuthorization(options: [.alert, .sound]) { [weak self] granted, error in
            if let error = error {
                print("[PostureNotificationManager] Notification auth error: \(error.localizedDescription)")
            }
            self?.isAuthorized = granted
        }
    }
    
    /// Evaluates current warning level and posts macOS system notifications with cooldown
    public func checkAndNotify(warningLevel: Int, deltaCVA: Double, relativeCVA: Double) {
        guard warningLevel > 0 else {
            previousWarningLevel = 0
            return
        }
        
        let now = CACurrentMediaTime()
        let timeSinceLast = now - lastNotificationTime
        
        // Trigger notification if:
        // 1. It escalated from Level 1 to Level 2 (Turtle Neck), or
        // 2. Cooldown period (120s) has passed
        let escalated = (warningLevel == 2 && previousWarningLevel < 2)
        guard escalated || timeSinceLast >= notificationCooldown else {
            previousWarningLevel = warningLevel
            return
        }
        
        lastNotificationTime = now
        previousWarningLevel = warningLevel
        
        let content = UNMutableNotificationContent()
        if warningLevel == 2 {
            content.title = "Turtle Neck Posture Alert"
            content.subtitle = String(format: "Head forward drift: ΔCVA %.1f° (rel %.1f°)", deltaCVA, relativeCVA)
            content.body = "Your head has drifted significantly forward. Pull your chin back and align your ears over your shoulders."
            content.sound = UNNotificationSound.default
        } else {
            content.title = "Posture Reminder: Forward Head Drift"
            content.subtitle = String(format: "ΔCVA %.1f° (rel %.1f°)", deltaCVA, relativeCVA)
            content.body = "Gentle reminder to maintain neutral cervical alignment and avoid slouching toward the screen."
            content.sound = UNNotificationSound.default
        }
        
        let trigger = UNTimeIntervalNotificationTrigger(timeInterval: 0.1, repeats: false)
        let request = UNNotificationRequest(
            identifier: "PostureAlert-\(UUID().uuidString)",
            content: content,
            trigger: trigger
        )
        
        center.add(request) { [weak self] error in
            if let error = error {
                print("[PostureNotificationManager] UNUserNotificationCenter delivery rejected (\(error.localizedDescription)). Using macOS system notification fallback.")
                self?.deliverSystemNotificationFallback(
                    title: content.title,
                    subtitle: content.subtitle,
                    body: content.body
                )
            }
        }
        
        // Also play macOS system beep
        DispatchQueue.main.async {
            NSSound.beep()
        }
    }
    
    /// Fallback notification delivery via macOS system notifications (AppleScript)
    /// Guarantees delivery even when binary runs unbundled without code-sign provisioning
    private func deliverSystemNotificationFallback(title: String, subtitle: String, body: String) {
        let fullSubtitle = subtitle.isEmpty ? "" : " subtitle \"\(subtitle.replacingOccurrences(of: "\"", with: "\\\""))\""
        let script = "display notification \"\(body.replacingOccurrences(of: "\"", with: "\\\""))\" with title \"\(title.replacingOccurrences(of: "\"", with: "\\\""))\"\(fullSubtitle) sound name \"Basso\""
        
        _ = try? Process.run(URL(fileURLWithPath: "/usr/bin/osascript"), arguments: ["-e", script], terminationHandler: nil)
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
