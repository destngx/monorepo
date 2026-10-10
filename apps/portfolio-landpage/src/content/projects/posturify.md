---
title: Posturify
summary: A native macOS menu bar app that uses the webcam and Apple Vision to measure head and neck posture as a craniovertebral angle, relative to a personal baseline, and warns on forward head posture.
year: 2026
status: active
stack: [Swift, SwiftUI, AVFoundation, Vision, SwiftPM]
repo: https://github.com/destngx/monorepo/tree/main/apps/posturify
featured: true
order: 4
sfx: ゾクッ
---

## The problem

I spend long stretches at a laptop, and my head drifts forward without my noticing. I wanted continuous feedback while I work, without wearables or manual checks. Generic "sit up straight" reminders do not tell me anything about my actual neck angle, so I built the app around a measurement instead.

## What it does

- Shows the camera feed with a 2D skeleton overlay, the person segmentation outline, and a HUD with the current status and guidance.
- Computes the craniovertebral angle (CVA). The code labels values above 60 degrees as normal, 55 to 60 as caution, and below 55 as turtle neck.
- Calibrates a personal neutral pose in three phases: sit neutral, lean forward and hold, then return to neutral. Readings are compared against that baseline.
- Flags overextension, meaning the head tilted back past roughly 12 degrees of pitch or jaw change, or 8 degrees of CVA change.
- Posts macOS notifications when the warning level changes. Repeat notifications are limited to one every 30 seconds unless the level escalates.
- Runs as an accessory app (`LSUIElement`) with a menu bar popover.

## How it's built

I wrote it as a Swift 5.9 package targeting macOS 14, with SwiftUI for the views, AVFoundation for capture, and Vision for inference. It has no third-party dependencies.

- `VisionTracker` runs face landmarks, human body pose, and person segmentation on each frame. The segmentation mask goes through contour detection to produce the outline.
- `KinematicsCalculator` and `PostureMathEngine` turn 2D landmarks into angles, correcting for camera angle and elevation.
- `CalibrationEngine` holds the baseline and the alert state machine.
- `AppState` coordinates everything. Vision inference targets 10 FPS while the window is visible and drops to 0.5 FPS when it is hidden.

## Interesting bits

- Warnings are based on deviation from the calibrated neutral pose, not only on absolute angles. Overextension, for example, is judged on delta values.
- Alerts go through a hysteresis state machine with a buffer zone of -5.0 to -2.5 degrees, where the current status is held, so the level does not flicker near a threshold.
- Frame-to-frame displacement is smoothed with an exponential moving average (`emaAlpha = 0.25`).
- The primary user is the face with the largest bounding box, which keeps a background person from taking over the readings.
- One pitch path blends landmark-based and body-based estimates at 60/40.
- The `KinematicsTests` target has 34 test functions covering angle math, camera modes, calibration hysteresis, and throttling.

## What's next

The repo has no roadmap. The test suite covers the kinematics and calibration math but not the notification or camera layers, so those are the untested parts.
