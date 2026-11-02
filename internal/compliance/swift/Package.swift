// swift-tools-version:6.0

// Runs packed's Swift output (Sources/Compliance/Messages.swift, written by
// ../generate.go) against the bytes its Go output writes to
// ../testdata/vectors.txt. go test in ../ runs swift test when swift is on
// PATH.
import PackageDescription

let package = Package(
    name: "Compliance",
    // The Testing macros need macOS 10.15 or newer.
    platforms: [.macOS(.v13)],
    targets: [
        .target(name: "Compliance"),
        .testTarget(name: "ComplianceTests", dependencies: ["Compliance"]),
    ]
)
