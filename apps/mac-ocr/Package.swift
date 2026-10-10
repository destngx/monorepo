// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "MacOCR",
    platforms: [
        .macOS(.v13)
    ],
    products: [
        .executable(name: "mac-ocr", targets: ["mac-ocr"])
    ],
    targets: [
        .target(
            name: "MacOCRCore",
            path: "Sources/MacOCRCore"
        ),
        .executableTarget(
            name: "mac-ocr",
            dependencies: ["MacOCRCore"],
            path: "Sources/mac-ocr"
        ),
        .testTarget(
            name: "MacOCRCoreTests",
            dependencies: ["MacOCRCore"],
            path: "Tests/MacOCRCoreTests"
        )
    ]
)
