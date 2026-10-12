import Foundation
import AppKit
import AVFoundation
import CoreAudio
import os

/// Reads system activity used to silence interruptive alerts: calls (microphone running), video meetings (camera in
/// use by another app) and presentations or videos (a full-screen window of the frontmost app).
/// None of these probes need extra permissions; window bounds and owners are readable without screen recording access.
public enum SystemActivityMonitor {
    private static let logger = Logger(subsystem: "com.destngx.posturify", category: "SystemActivity")

    @MainActor
    public static func current() -> SystemActivity {
        SystemActivity(
            isMicrophoneInUse: isAnyInputDeviceRunning(),
            isCameraInUseByAnotherApp: isCameraInUseByAnotherApp(),
            isFullScreenAppActive: isFrontmostAppFullScreen()
        )
    }

    // MARK: - Microphone

    private static func isAnyInputDeviceRunning() -> Bool {
        audioDeviceIDs().contains { hasInputStreams($0) && isRunningSomewhere($0) }
    }

    private static func audioDeviceIDs() -> [AudioDeviceID] {
        var address = AudioObjectPropertyAddress(
            mSelector: kAudioHardwarePropertyDevices,
            mScope: kAudioObjectPropertyScopeGlobal,
            mElement: kAudioObjectPropertyElementMain
        )
        var size: UInt32 = 0
        var status = AudioObjectGetPropertyDataSize(AudioObjectID(kAudioObjectSystemObject), &address, 0, nil, &size)
        guard status == noErr else {
            logger.error("Audio device list size query failed: \(status)")
            return []
        }
        var ids = [AudioDeviceID](repeating: 0, count: Int(size) / MemoryLayout<AudioDeviceID>.size)
        status = AudioObjectGetPropertyData(AudioObjectID(kAudioObjectSystemObject), &address, 0, nil, &size, &ids)
        guard status == noErr else {
            logger.error("Audio device list query failed: \(status)")
            return []
        }
        return ids
    }

    private static func hasInputStreams(_ device: AudioDeviceID) -> Bool {
        var address = AudioObjectPropertyAddress(
            mSelector: kAudioDevicePropertyStreams,
            mScope: kAudioObjectPropertyScopeInput,
            mElement: kAudioObjectPropertyElementMain
        )
        var size: UInt32 = 0
        let status = AudioObjectGetPropertyDataSize(device, &address, 0, nil, &size)
        return status == noErr && size > 0
    }

    private static func isRunningSomewhere(_ device: AudioDeviceID) -> Bool {
        var address = AudioObjectPropertyAddress(
            mSelector: kAudioDevicePropertyDeviceIsRunningSomewhere,
            mScope: kAudioObjectPropertyScopeGlobal,
            mElement: kAudioObjectPropertyElementMain
        )
        var running: UInt32 = 0
        var size = UInt32(MemoryLayout<UInt32>.size)
        let status = AudioObjectGetPropertyData(device, &address, 0, nil, &size, &running)
        guard status == noErr else {
            logger.error("Audio device running query failed for device \(device): \(status)")
            return false
        }
        return running != 0
    }

    // MARK: - Camera

    private static func isCameraInUseByAnotherApp() -> Bool {
        let discovery = AVCaptureDevice.DiscoverySession(
            deviceTypes: [.builtInWideAngleCamera, .external, .continuityCamera],
            mediaType: .video,
            position: .unspecified
        )
        return discovery.devices.contains { $0.isInUseByAnotherApplication }
    }

    // MARK: - Full Screen

    @MainActor
    private static func isFrontmostAppFullScreen() -> Bool {
        guard let front = NSWorkspace.shared.frontmostApplication,
              front.processIdentifier != ProcessInfo.processInfo.processIdentifier,
              let windows = CGWindowListCopyWindowInfo([.optionOnScreenOnly, .excludeDesktopElements], kCGNullWindowID)
                as? [[String: Any]] else {
            return false
        }
        let screenSizes = NSScreen.screens.map(\.frame.size)
        return windows.contains { info in
            guard (info[kCGWindowOwnerPID as String] as? pid_t) == front.processIdentifier,
                  (info[kCGWindowLayer as String] as? Int) == 0,
                  let boundsInfo = info[kCGWindowBounds as String] as? NSDictionary,
                  let bounds = CGRect(dictionaryRepresentation: boundsInfo) else {
                return false
            }
            return screenSizes.contains { abs($0.width - bounds.width) < 1 && abs($0.height - bounds.height) < 1 }
        }
    }
}
