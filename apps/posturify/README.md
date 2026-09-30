# Posturify

A native macOS application providing real-time biomechanical tracking and posture coaching using Apple Vision framework and native SwiftUI.

Powered by native **Swift 5.9+**, **SwiftUI**, **AVFoundation**, and **Vision Framework (Apple Neural Engine)** with 0 third-party dependencies.

---

## 1. Ergonomic Biomechanics & Posture Telemetry

- **Craniovertebral Angle (CVA)**: Continuous clinical measurement of head protrusion and forward neck posture.
- **Relative Posture Calibration**: 2-second user baseline calibration for relative $\Delta\text{CVA}$ delta coaching.
- **Vision-Driven Silhouette Isolation**: Real-time neural person segmentation filter isolates the user and dims background distractions.
- **Accessory MenuBar Integration**: Sits natively in macOS status bar (`LSUIElement`) with live status chips and zero clutter.

---

## 2. Directory Layout

```
apps/posturify/
├── Package.swift                               # Swift 5.9 / macOS 14+ SPM configuration
├── project.json                                # Nx monorepo target mappings
├── Makefile                                    # Quick development & bundle build commands
├── Resources/
│   ├── AppIcon.svg                             # High-resolution vector master
│   └── AppIcon.icns                            # Multi-resolution native macOS icon bundle
├── Sources/
│   ├── App/
│   │   ├── AppState.swift                      # Central state coordinator & 60 FPS clock
│   │   ├── PosturifyApp.swift                  # Main SwiftUI App entry point
│   │   └── PostureNotificationManager.swift    # Posture alerts & cooldown management
│   ├── Biomechanics/
│   │   ├── KinematicsCalculator.swift          # Biomechanics & CVA trigonometry
│   │   ├── PostureMathEngine.swift             # Projection & elevation adjustments
│   │   ├── CalibrationEngine.swift             # Baseline calibration & hysteresis
│   │   └── SkeletalKinematicsModels.swift      # Joint models & status definitions
│   ├── Vision/
│   │   ├── CameraManager.swift                 # AVFoundation zero-latency video pipeline
│   │   └── VisionTracker.swift                 # Concurrent face & body pose inference on ANE
│   ├── Views/
│   │   ├── Display1CameraView.swift            # Camera feed, 2D skeleton line overlays & HUD
│   │   ├── MenuBarContentView.swift            # Native MenuBar Popover container
│   │   └── DualDisplayMainView.swift           # Main control center & telemetry strip
│   └── Resources/
│       └── Info.plist                          # Bundle metadata & NSCameraUsageDescription
└── Tests/
    └── KinematicsTests/
        └── KinematicsTests.swift               # Kinematics & telemetry unit tests
```

---

## 3. Build & Packaging

### macOS Application Bundle (`Posturify.app`)

To produce the standalone macOS Application Bundle with icon and Info.plist:

```bash
cd apps/posturify
make bundle
```

This creates `build/Posturify.app/`:

```
Posturify.app/
  Contents/
    Info.plist
    MacOS/
      Posturify       # Mach-O executable (arm64)
    Resources/
      AppIcon.icns    # Native multi-resolution icon
```

### Running via Nx

```bash
# Run unit test suite
pnpx nx test posturify

# Build application bundle
pnpx nx run posturify:bundle

# Launch development build
pnpx nx run posturify:run
```

### Running via Swift PM

```bash
cd apps/posturify
swift test
swift run
```
