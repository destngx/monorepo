// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "Posturify",
    platforms: [
        .macOS(.v14)
    ],
    products: [
        .executable(
            name: "Posturify",
            targets: ["Posturify"]
        )
    ],
    targets: [
        .executableTarget(
            name: "Posturify",
            dependencies: [],
            path: "Sources",
            exclude: ["Resources/Info.plist"],
            resources: [
                .copy("Resources/Info.plist")
            ],
            linkerSettings: [
                .unsafeFlags([
                    "-Xlinker", "-sectcreate",
                    "-Xlinker", "__TEXT",
                    "-Xlinker", "__info_plist",
                    "-Xlinker", "Sources/Resources/Info.plist"
                ])
            ]
        ),
        .testTarget(
            name: "KinematicsTests",
            dependencies: ["Posturify"],
            path: "Tests/KinematicsTests"
        )
    ]
)
