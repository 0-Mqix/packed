package compliance

import java.io.File
import kotlin.math.PI
import kotlin.test.Test
import kotlin.test.assertContentEquals
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith

class ComplianceTest {
    private val vectors: Map<String, ByteArray> =
        File(System.getProperty("packed.vectors")).readLines()
            .filter { it.isNotBlank() }
            .associate { line ->
                val (name, hex) = line.split(" ")
                name to hex.chunked(2).map { it.toInt(16).toByte() }.toByteArray()
            }

    private val primitives = Primitives(
        flag = true,
        small = -5,
        byte = 250u,
        short = -12345,
        unsignedShort = 54321u,
        integer = -123456789,
        unsignedInteger = 3123456789u,
        long = Long.MIN_VALUE + 7,
        unsignedLong = ULong.MAX_VALUE - 3u,
        single = -1.25f,
        double = PI,
        littleInteger = 0x01020304,
        code = "ab·",
    )

    private val bits = Bits(
        active = true,
        mode = 9u,
        delta = -11,
        wide = 0x0ABCDEF012345678uL,
        straddle = -54321,
        tail = 1500u,
        inner = Inner(a = 5u, b = -200, c = 0xBEEFu),
        bigInner = Inner(a = 3u, b = 255, c = 0x1234u),
        inners = listOf(Inner(a = 1u, b = -1, c = 1u), Inner(a = 7u, b = -256, c = 65535u)),
    )

    private val record = Record(
        id = 4000000000u,
        name = "Pump 3 · west 🚰",
        severity = Severity.High,
        signed = Signed.Positive,
        flags = RecordFlags(active = true, level = 6u),
        labels = listOf(Label(name = "inlet", weight = 0.5f), Label(name = "", weight = -3f)),
        values = listOf(-32768, 0, 32767),
        trailer = 0xCAFEu,
        pair = listOf("left", ""),
        nested = listOf(listOf(1u, 2u, 3u), listOf(), listOf(255u)),
        severities = listOf(Severity.Low, Severity.Medium, Severity.High),
        matrix = listOf(listOf(1, -2), listOf(-3, 4)),
    )

    private val ping = Ping(sequence = 0xDEADBEEFu)

    private fun encode(size: Int, toBytes: (ByteArray, Int) -> Int): ByteArray {
        val bytes = ByteArray(size)
        assertEquals(size, toBytes(bytes, 0))
        return bytes
    }

    @Test
    fun primitivesMatchGo() {
        assertContentEquals(vectors.getValue("primitives"), encode(primitives.size(), primitives::toBytes))
        assertEquals(primitives, Primitives.fromBytes(vectors.getValue("primitives"), 0))
    }

    @Test
    fun bitsMatchGo() {
        assertContentEquals(vectors.getValue("bits"), encode(bits.size(), bits::toBytes))
        assertEquals(bits, Bits.fromBytes(vectors.getValue("bits"), 0))
    }

    @Test
    fun recordMatchesGo() {
        assertContentEquals(vectors.getValue("record"), encode(record.size(), record::toBytes))
        assertEquals(record, Record.fromBytes(vectors.getValue("record"), 0))
    }

    @Test
    fun pingMatchesGo() {
        assertContentEquals(vectors.getValue("ping"), encode(ping.size(), ping::toBytes))
        assertEquals(ping, Ping.fromBytes(vectors.getValue("ping"), 0))
    }

    // A data class compares a ByteArray by reference, so the fields are
    // compared by content.
    @Test
    fun blobMatchesGo() {
        val blob = Blob(payload = byteArrayOf(0, 1, -2, -1), small = byteArrayOf(), chunks = listOf(byteArrayOf(7), byteArrayOf(), byteArrayOf(8, 9)))
        assertContentEquals(vectors.getValue("blob"), encode(blob.size(), blob::toBytes))
        val decoded = Blob.fromBytes(vectors.getValue("blob"), 0)
        assertContentEquals(blob.payload, decoded.payload)
        assertContentEquals(blob.small, decoded.small)
        assertEquals(blob.chunks.map { it.toList() }, decoded.chunks.map { it.toList() })
    }

    @Test
    fun statusMatchesGo() {
        val status = Status(identifier = 0x5Eu, version = 2u, keys = 0x15u, frames = 0x1234u, log = listOf(0x6Fu, 0x6Bu, 0u, 255u))
        assertContentEquals(vectors.getValue("status"), encode(status.size(), status::toBytes))
        assertEquals(status, Status.fromBytes(vectors.getValue("status"), 0))
    }

    @Test
    fun descriptorMatchesGo() {
        val descriptor = Descriptor(
            level = Level.Warning,
            message = 0xBEEFu,
            keys = listOf(KeyEvent(key = 1u, kind = KeyKind.Press), KeyEvent(key = 0x0203u, kind = KeyKind.Hold)),
        )
        assertContentEquals(vectors.getValue("descriptor"), encode(descriptor.size(), descriptor::toBytes))
        assertEquals(descriptor, Descriptor.fromBytes(vectors.getValue("descriptor"), 0))
    }

    // Structs nested every way, as outer() in compliance_test.go builds them.
    @Test
    fun outerMatchesGo() {
        val tag = Tagged(on = true, mode = Mode.Fast, at = Point(x = -3, y = 7))
        val outer = Outer(
            inner = Named(name = "inner", points = listOf(Point(x = 1, y = 2))),
            pair = listOf(
                Named(name = "left", points = listOf()),
                Named(name = "right", points = listOf(Point(x = -1, y = -2), Point(x = 300, y = -300))),
            ),
            grid = listOf(
                listOf(Point(x = 1, y = 2), Point(x = 3, y = 4)),
                listOf(Point(x = 5, y = 6), Point(x = 7, y = 8)),
            ),
            tags = listOf(tag, Tagged(on = false, mode = Mode.Slow, at = Point(x = 9, y = 10))),
            names = listOf(Named(name = "a", points = listOf(Point(x = 11, y = 12))), Named(name = "bb", points = listOf())),
            bigs = listOf(Big(value = 0x1234u), Big(value = 0xABCDu)),
            deep = Deep(level = Deeper(tags = listOf(tag), mode = Mode.Slow)),
        )
        assertContentEquals(vectors.getValue("outer"), encode(outer.size(), outer::toBytes))
        assertEquals(outer, Outer.fromBytes(vectors.getValue("outer"), 0))
    }

    // Pixel i is 3i+1, as frame() in compliance_test.go builds it.
    @Test
    fun displaysMatchGo() {
        val frame = SetFrame(pixels = List(192) { (it * 3 + 1).toUByte() })
        assertContentEquals(vectors.getValue("display-setframe"), frame.encode())
        assertContentEquals(vectors.getValue("display-read"), Read().encode())
        assertEquals(frame, Display.decode(vectors.getValue("display-setframe")))
        assertEquals(Read(), Display.decode(vectors.getValue("display-read")))
    }

    @Test
    fun envelopesMatchGo() {
        assertContentEquals(vectors.getValue("envelope-record"), record.encode())
        assertContentEquals(vectors.getValue("envelope-ping"), Ping(sequence = 7u).encode())
        assertEquals(record, Envelope.decode(vectors.getValue("envelope-record")))
        assertEquals(Ping(sequence = 7u), Envelope.decode(vectors.getValue("envelope-ping")))
    }

    @Test
    fun envelopeRejectsBadInput() {
        val encoded = record.encode()
        assertFailsWith<IllegalArgumentException> { Envelope.decode(byteArrayOf()) }
        assertFailsWith<IllegalArgumentException> { Envelope.decode(byteArrayOf(99, 1, 2)) }
        assertFailsWith<IllegalArgumentException> { Envelope.decode(encoded.copyOf(encoded.size - 3)) }
        assertFailsWith<IllegalArgumentException> { Envelope.decode(encoded + 0) }
    }

    private val toggle = Toggle(on = true, channel = 9u, duration = 0xBEEFu)

    // The messages of the sets with TagBits, as taggedSamples() in
    // compliance_test.go builds them.
    private val commands: Map<String, Command> = mapOf(
        "command-toggle" to toggle,
        "command-rename" to Rename(name = "pump"),
        "command-dial" to Dial(level = -20, fine = true, severity = Severity.High),
        "command-halt" to Halt(),
    )

    private val beacon = Beacon(urgent = true, level = -7, signed = Signed.Positive, count = 0x1234u)

    private val heartbeat = Heartbeat(count = 42u)

    @Test
    fun tagBitsMatchGo() {
        for ((name, command) in commands) {
            assertContentEquals(vectors.getValue(name), command.encode(), name)
            assertEquals(command, Command.decode(vectors.getValue(name)), name)
        }
        assertContentEquals(vectors.getValue("signal-beacon"), beacon.encode())
        assertEquals(beacon, Signal.decode(vectors.getValue("signal-beacon")))
        assertContentEquals(vectors.getValue("signal-quiet"), Quiet().encode())
        assertEquals(Quiet(), Signal.decode(vectors.getValue("signal-quiet")))
        assertContentEquals(vectors.getValue("pulse-heartbeat"), heartbeat.encode())
        assertEquals(heartbeat, Pulse.decode(vectors.getValue("pulse-heartbeat")))
    }

    @Test
    fun prefixAndIndexMatchGo() {
        val prefixed = toggle.encode(byteArrayOf(0xAB.toByte()))
        assertContentEquals(vectors.getValue("prefixed-toggle"), prefixed)
        assertEquals(toggle, Command.decode(prefixed, 1))
        assertContentEquals(vectors.getValue("envelope-ping"), Ping(sequence = 7u).encode(byteArrayOf(1, 2)).copyOfRange(2, 7))
        assertEquals(Ping(sequence = 7u), Envelope.decode(byteArrayOf(1, 2) + vectors.getValue("envelope-ping"), 2))
    }

    @Test
    fun tagBitsRejectBadInput() {
        val encoded = toggle.encode()
        assertFailsWith<IllegalArgumentException> { Command.decode(byteArrayOf()) }
        assertFailsWith<IllegalArgumentException> { Command.decode(encoded, encoded.size) }
        assertFailsWith<IllegalArgumentException> { Command.decode(byteArrayOf(1)) }
        assertFailsWith<IllegalArgumentException> { Signal.decode(byteArrayOf(0xC0.toByte())) }
        assertFailsWith<IllegalArgumentException> { Command.decode(encoded.copyOf(2)) }
        assertFailsWith<IllegalArgumentException> { Command.decode(encoded + 0) }
        assertFailsWith<IllegalArgumentException> { Pulse.decode(byteArrayOf(1, 2)) }
        assertFailsWith<IllegalArgumentException> { Pulse.decode(byteArrayOf(1, 2, 3, 4, 5)) }
        // Dial's Severity is bits 10 and 11; 3 has no value.
        val dial = vectors.getValue("command-dial").copyOf()
        dial[1] = (dial[1].toInt() or 0x0C).toByte()
        assertFailsWith<IllegalArgumentException> { Command.decode(dial) }
        for ((name, command) in commands) {
            val bytes = command.encode()
            for (length in 0 until bytes.size) {
                assertFailsWith<IllegalArgumentException>(name) { Command.decode(bytes.copyOf(length)) }
            }
        }
    }

    @Test
    fun lengthPrefixOverflowFails() {
        val label = Label(name = "x".repeat(256), weight = 0f)
        assertFailsWith<IllegalArgumentException> { label.toBytes(ByteArray(300), 0) }
    }
}
