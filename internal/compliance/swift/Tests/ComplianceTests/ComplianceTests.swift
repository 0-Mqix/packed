import Compliance
import Foundation
import Testing

// The values must match samples() in ../../../compliance_test.go, which
// writes their Go encoding to testdata/vectors.txt.

private let vectors: [String: [UInt8]] = {
    let path = URL(fileURLWithPath: #filePath)
        .deletingLastPathComponent()
        .deletingLastPathComponent()
        .deletingLastPathComponent()
        .deletingLastPathComponent()
        .appendingPathComponent("testdata/vectors.txt")
    let text = try! String(contentsOf: path, encoding: .utf8)
    var result: [String: [UInt8]] = [:]
    for line in text.split(separator: "\n") where !line.isEmpty {
        let parts = line.split(separator: " ")
        let hex = Array(parts[1])
        result[String(parts[0])] = stride(from: 0, to: hex.count, by: 2).map {
            UInt8(String(hex[$0..<$0 + 2]), radix: 16)!
        }
    }
    return result
}()

private func vector(_ name: String) -> [UInt8] {
    vectors[name]!
}

private let primitives = Primitives(
    flag: true,
    small: -5,
    byte: 250,
    short: -12345,
    unsignedShort: 54321,
    integer: -123456789,
    unsignedInteger: 3123456789,
    long: Int64.min + 7,
    unsignedLong: UInt64.max - 3,
    single: -1.25,
    double: Double.pi,
    littleInteger: 0x01020304,
    code: "ab·"
)

private let bits = Bits(
    active: true,
    mode: 9,
    delta: -11,
    wide: 0x0ABCDEF012345678,
    straddle: -54321,
    tail: 1500,
    inner: Inner(a: 5, b: -200, c: 0xBEEF),
    bigInner: Inner(a: 3, b: 255, c: 0x1234),
    inners: [Inner(a: 1, b: -1, c: 1), Inner(a: 7, b: -256, c: 65535)]
)

private let record = Record(
    id: 4000000000,
    name: "Pump 3 · west 🚰",
    severity: .high,
    signed: .positive,
    flags: RecordFlags(active: true, level: 6),
    labels: [Label(name: "inlet", weight: 0.5), Label(name: "", weight: -3)],
    values: [-32768, 0, 32767],
    trailer: 0xCAFE,
    pair: ["left", ""],
    nested: [[1, 2, 3], [], [255]],
    severities: [.low, .medium, .high],
    matrix: [[1, -2], [-3, 4]]
)

private let ping = Ping(sequence: 0xDEADBEEF)

/// Encodes value, requires Go's bytes, decodes Go's bytes back to value,
/// and requires every shorter input to throw instead of trapping.
private func check<Value: Equatable>(
    _ name: String,
    _ value: Value,
    size: Int,
    toBytes: (inout [UInt8], Int) -> Int,
    fromBytes: ([UInt8], Int) throws -> (value: Value, end: Int)
) throws {
    let expected = vector(name)

    var bytes = [UInt8](repeating: 0, count: size)
    #expect(toBytes(&bytes, 0) == size)
    #expect(bytes == expected)

    let decoded = try fromBytes(expected, 0)
    #expect(decoded.value == value)
    #expect(decoded.end == expected.count)

    let shifted = [0xAA, 0xBB, 0xCC] + expected + [0xDD]
    let offset = try fromBytes(shifted, 3)
    #expect(offset.value == value)
    #expect(offset.end == 3 + expected.count)

    for length in 0..<expected.count {
        #expect(throws: PackedError.truncated) {
            _ = try fromBytes(Array(expected.prefix(length)), 0)
        }
    }
}

@Test func primitivesMatchGo() throws {
    try check("primitives", primitives, size: primitives.size(), toBytes: primitives.toBytes, fromBytes: Primitives.fromBytes)
}

@Test func bitsMatchGo() throws {
    try check("bits", bits, size: bits.size(), toBytes: bits.toBytes, fromBytes: Bits.fromBytes)
}

@Test func recordMatchesGo() throws {
    try check("record", record, size: record.size(), toBytes: record.toBytes, fromBytes: Record.fromBytes)
}

@Test func pingMatchesGo() throws {
    try check("ping", ping, size: ping.size(), toBytes: ping.toBytes, fromBytes: Ping.fromBytes)
}

@Test func blobMatchesGo() throws {
    let blob = Blob(payload: [0, 1, 254, 255], small: [], chunks: [[7], [], [8, 9]])
    try check("blob", blob, size: blob.size(), toBytes: blob.toBytes, fromBytes: Blob.fromBytes)
}

@Test func statusMatchesGo() throws {
    let status = Status(identifier: 0x5E, version: 2, keys: 0x15, frames: 0x1234, log: [0x6F, 0x6B, 0, 255])
    try check("status", status, size: status.size(), toBytes: status.toBytes, fromBytes: Status.fromBytes)
}

@Test func descriptorMatchesGo() throws {
    let descriptor = Descriptor(
        level: .warning,
        message: 0xBEEF,
        keys: [KeyEvent(key: 1, kind: .press), KeyEvent(key: 0x0203, kind: .hold)]
    )
    try check("descriptor", descriptor, size: descriptor.size(), toBytes: descriptor.toBytes, fromBytes: Descriptor.fromBytes)
}

@Test func outerMatchesGo() throws {
    // Structs nested every way, as outer() in compliance_test.go builds them.
    let tag = Tagged(on: true, mode: .fast, at: Point(x: -3, y: 7))
    let outer = Outer(
        inner: Named(name: "inner", points: [Point(x: 1, y: 2)]),
        pair: [
            Named(name: "left", points: []),
            Named(name: "right", points: [Point(x: -1, y: -2), Point(x: 300, y: -300)]),
        ],
        grid: [[Point(x: 1, y: 2), Point(x: 3, y: 4)], [Point(x: 5, y: 6), Point(x: 7, y: 8)]],
        tags: [tag, Tagged(on: false, mode: .slow, at: Point(x: 9, y: 10))],
        names: [Named(name: "a", points: [Point(x: 11, y: 12)]), Named(name: "bb", points: [])],
        bigs: [Big(value: 0x1234), Big(value: 0xABCD)],
        deep: Deep(level: Deeper(tags: [tag], mode: .slow))
    )
    try check("outer", outer, size: outer.size(), toBytes: outer.toBytes, fromBytes: Outer.fromBytes)
}

@Test func displaysMatchGo() throws {
    // Pixel i is 3i+1, as frame() in compliance_test.go builds it.
    let frame = SetFrame(pixels: (0..<192).map { UInt8(truncatingIfNeeded: $0 * 3 + 1) })
    #expect(Display.setFrame(frame).encode() == vector("display-setframe"))
    #expect(Display.read(Read()).encode() == vector("display-read"))
    #expect(try Display.decode(vector("display-setframe")) == .setFrame(frame))
    #expect(try Display.decode(vector("display-read")) == .read(Read()))
}

@Test func envelopesMatchGo() throws {
    #expect(record.encode() == vector("envelope-record"))
    #expect(Envelope.record(record).encode() == vector("envelope-record"))
    #expect(Ping(sequence: 7).encode() == vector("envelope-ping"))
    #expect(try Envelope.decode(vector("envelope-record")) == .record(record))
    #expect(try Envelope.decode(vector("envelope-ping")) == .ping(Ping(sequence: 7)))
}

@Test func envelopeRejectsBadInput() {
    let encoded = record.encode()
    #expect(throws: PackedError.empty(set: "Envelope")) {
        try Envelope.decode([])
    }
    #expect(throws: PackedError.unknownType(set: "Envelope", type: 99)) {
        try Envelope.decode([99, 1, 2])
    }
    #expect(throws: PackedError.truncatedMessage(set: "Envelope", type: 7)) {
        try Envelope.decode(Array(encoded.dropLast(3)))
    }
    #expect(throws: PackedError.trailingBytes(set: "Envelope", type: 7, count: 1)) {
        try Envelope.decode(encoded + [0])
    }
}

private let toggle = Toggle(on: true, channel: 9, duration: 0xBEEF)

// The messages of the sets with TagBits, as taggedSamples() in
// compliance_test.go builds them.
private let commands: [(String, Command)] = [
    ("command-toggle", .toggle(toggle)),
    ("command-rename", .rename(Rename(name: "pump"))),
    ("command-dial", .dial(Dial(level: -20, fine: true, severity: .high))),
    ("command-halt", .halt(Halt())),
]

private let beacon = Beacon(urgent: true, level: -7, signed: .positive, count: 0x1234)

private let heartbeat = Heartbeat(count: 42)

@Test func tagBitsMatchGo() throws {
    for (name, command) in commands {
        #expect(command.encode() == vector(name), "\(name)")
        #expect(try Command.decode(vector(name)) == command, "\(name)")
    }
    #expect(beacon.encode() == vector("signal-beacon"))
    #expect(try Signal.decode(vector("signal-beacon")) == .beacon(beacon))
    #expect(Quiet().encode() == vector("signal-quiet"))
    #expect(try Signal.decode(vector("signal-quiet")) == .quiet(Quiet()))
    #expect(heartbeat.encode() == vector("pulse-heartbeat"))
    #expect(try Pulse.decode(vector("pulse-heartbeat")) == .heartbeat(heartbeat))
}

@Test func prefixAndIndexMatchGo() throws {
    let prefixed = Command.toggle(toggle).encode(prefix: [0xAB])
    #expect(prefixed == vector("prefixed-toggle"))
    #expect(try Command.decode(prefixed, from: 1) == .toggle(toggle))
    #expect(Array(Ping(sequence: 7).encode(prefix: [1, 2]).dropFirst(2)) == vector("envelope-ping"))
    #expect(try Envelope.decode([1, 2] + vector("envelope-ping"), from: 2) == .ping(Ping(sequence: 7)))
}

@Test func tagBitsRejectBadInput() {
    let encoded = toggle.encode()
    #expect(throws: PackedError.empty(set: "Command")) {
        try Command.decode([])
    }
    #expect(throws: PackedError.empty(set: "Command")) {
        try Command.decode(encoded, from: encoded.count)
    }
    #expect(throws: PackedError.unknownType(set: "Command", type: 1)) {
        try Command.decode([0x01])
    }
    #expect(throws: PackedError.unknownType(set: "Signal", type: 3)) {
        try Signal.decode([0xC0])
    }
    #expect(throws: PackedError.truncatedMessage(set: "Command", type: 0)) {
        try Command.decode(Array(encoded.prefix(2)))
    }
    #expect(throws: PackedError.trailingBytes(set: "Command", type: 0, count: 1)) {
        try Command.decode(encoded + [0])
    }
    #expect(throws: PackedError.truncatedMessage(set: "Pulse", type: 0)) {
        try Pulse.decode([1, 2])
    }
    #expect(throws: PackedError.trailingBytes(set: "Pulse", type: 0, count: 1)) {
        try Pulse.decode([1, 2, 3, 4, 5])
    }
    // Dial's Severity is bits 10 and 11; 3 has no case.
    var dial = vector("command-dial")
    dial[1] |= 0x0C
    #expect(throws: PackedError.unknownValue(enumeration: "Severity", value: 3)) {
        try Command.decode(dial)
    }
    for (name, command) in commands {
        let bytes = command.encode()
        for length in 0..<bytes.count {
            #expect(throws: PackedError.self, "\(name)") {
                try Command.decode(Array(bytes.prefix(length)))
            }
        }
    }
}

@Test func unknownEnumValueThrows() {
    var bytes = vector("record")
    // id (4 bytes), then the name with its two-byte length, then severity.
    bytes[4 + 2 + record.name.utf8.count] = 9
    #expect(throws: PackedError.unknownValue(enumeration: "Severity", value: 9)) {
        _ = try Record.fromBytes(bytes, 0)
    }
    #expect(throws: PackedError.unknownValue(enumeration: "Signed", value: -1)) {
        _ = try Signed(validating: -1)
    }
}
