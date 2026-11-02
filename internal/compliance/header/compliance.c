// The generated header (messages.h) against the bytes Go wrote to
// ../testdata/vectors.txt, through its C functions: the same samples as
// compliance_test.go builds, encoded, decoded and encoded again, with the
// errors Go reports. TestHeader compiles it as C and as C++ with each host
// compiler and runs it with the vectors' path.
#include "messages.h"

#include <stdio.h>
#include <stdlib.h>

typedef struct Vector {
    char name[40];
    uint8_t bytes[256];
    size_t length;
} Vector;

static Vector vectors[48];
static size_t vectorCount;
static int failures;

static void check(bool condition, const char* name, const char* what) {
    if (!condition) {
        printf("FAIL %s: %s\n", name, what);
        failures++;
    }
}

static int hexDigit(char digit) {
    if (digit >= '0' && digit <= '9') {
        return digit - '0';
    }
    return digit - 'a' + 10;
}

static void load(const char* path) {
    FILE* file = fopen(path, "r");
    if (file == NULL) {
        printf("FAIL cannot open %s\n", path);
        exit(1);
    }
    char name[40];
    char hex[600];
    while (vectorCount < 48 && fscanf(file, "%39s %599s", name, hex) == 2) {
        Vector* vector = &vectors[vectorCount++];
        memcpy(vector->name, name, sizeof name);
        vector->length = strlen(hex) / 2;
        for (size_t index = 0; index < vector->length; index++) {
            vector->bytes[index] = (uint8_t)(hexDigit(hex[2 * index]) * 16 + hexDigit(hex[2 * index + 1]));
        }
    }
    fclose(file);
}

static const Vector* vector(const char* name) {
    for (size_t index = 0; index < vectorCount; index++) {
        if (strcmp(vectors[index].name, name) == 0) {
            return &vectors[index];
        }
    }
    printf("FAIL no vector %s\n", name);
    exit(1);
}

static bool same(const uint8_t* bytes, size_t length, const Vector* expected) {
    return length == expected->length && memcmp(bytes, expected->bytes, length) == 0;
}

static bool equal(const char* text, size_t length, const char* expected) {
    return length == strlen(expected) && memcmp(text, expected, length) == 0;
}

// SET_TEXT points a text field and its length at a C string.
#define SET_TEXT(text, length, value) ((text) = (value), (length) = strlen(value))

// CHECK_STRUCT encodes value, decodes the vector into decoded and encodes
// that again, and checks that every shorter input is truncated and every
// smaller output full.
#define CHECK_STRUCT(Type, name, value, decoded) \
    do { \
        const Vector* expected = vector(name); \
        uint8_t bytes[256]; \
        size_t end = 0; \
        check(Type##Size(&(value)) == expected->length, name, "size"); \
        check(Type##ToBytes(&(value), bytes, sizeof bytes, 0, &end) == PackedErrorNone && same(bytes, end, expected), name, "encoded"); \
        check(Type##FromBytes(&(decoded), expected->bytes, expected->length, 0, &end) == PackedErrorNone && end == expected->length, name, "decoded"); \
        check(Type##Size(&(decoded)) == expected->length, name, "decoded size"); \
        check(Type##ToBytes(&(decoded), bytes, sizeof bytes, 0, &end) == PackedErrorNone && same(bytes, end, expected), name, "encoded again"); \
        for (size_t length = 0; length < expected->length; length++) { \
            Type truncated; \
            memset(&truncated, 0, sizeof truncated); \
            check(Type##FromBytes(&truncated, expected->bytes, length, 0, &end) == PackedErrorTruncated, name, "truncated"); \
            check(Type##ToBytes(&(value), bytes, length, 0, &end) == PackedErrorFull, name, "full"); \
        } \
    } while (0)

// CHECK_SET does the same for a message set, and checks that every shorter
// input fails.
#define CHECK_SET(Type, name, message, decoded) \
    do { \
        const Vector* expected = vector(name); \
        uint8_t bytes[256]; \
        size_t end = 0; \
        check(Type##Size(&(message)) == expected->length, name, "size"); \
        check(Type##Encode(&(message), bytes, sizeof bytes, &end) == PackedErrorNone && same(bytes, end, expected), name, "encoded"); \
        check(Type##Decode(&(decoded), expected->bytes, expected->length) == PackedErrorNone, name, "decoded"); \
        check((decoded).type == (message).type, name, "decoded type"); \
        check(Type##Encode(&(decoded), bytes, sizeof bytes, &end) == PackedErrorNone && same(bytes, end, expected), name, "encoded again"); \
        for (size_t length = 0; length < expected->length; length++) { \
            Type truncated; \
            check(Type##Decode(&truncated, expected->bytes, length) != PackedErrorNone, name, "decoded a shorter input"); \
        } \
    } while (0)

// ZERO clears a local, as C and C++ both understand it.
#define ZERO(value) memset(&(value), 0, sizeof(value))

int main(int argumentCount, char** arguments) {
    if (argumentCount != 2) {
        printf("usage: compliance vectors.txt\n");
        return 2;
    }
    load(arguments[1]);

    Primitives primitives;
    Primitives decodedPrimitives;
    ZERO(primitives);
    ZERO(decodedPrimitives);
    primitives.flag = true;
    primitives.small = -5;
    primitives.byte = 250;
    primitives.short_ = -12345;
    primitives.unsignedShort = 54321;
    primitives.integer = -123456789;
    primitives.unsignedInteger = 3123456789u;
    primitives.long_ = INT64_MIN + 7;
    primitives.unsignedLong = UINT64_MAX - 3;
    primitives.single = -1.25f;
    primitives.double_ = 3.141592653589793;
    primitives.littleInteger = 0x01020304;
    memcpy(primitives.code, "ab\xC2\xB7\0\0", 6);
    CHECK_STRUCT(Primitives, "primitives", primitives, decodedPrimitives);
    check(decodedPrimitives.long_ == INT64_MIN + 7 && decodedPrimitives.double_ == 3.141592653589793, "primitives", "fields");

    Bits bits;
    Bits decodedBits;
    ZERO(bits);
    ZERO(decodedBits);
    bits.active = true;
    bits.mode = 9;
    bits.delta = -11;
    bits.wide = 0x0ABCDEF012345678;
    bits.straddle = -54321;
    bits.tail = 1500;
    bits.inner.a = 5;
    bits.inner.b = -200;
    bits.inner.c = 0xBEEF;
    bits.bigInner.a = 3;
    bits.bigInner.b = 255;
    bits.bigInner.c = 0x1234;
    bits.inners[0].a = 1;
    bits.inners[0].b = -1;
    bits.inners[0].c = 1;
    bits.inners[1].a = 7;
    bits.inners[1].b = -256;
    bits.inners[1].c = 65535;
    CHECK_STRUCT(Bits, "bits", bits, decodedBits);
    check(decodedBits.delta == -11 && decodedBits.straddle == -54321 && decodedBits.wide == 0x0ABCDEF012345678, "bits", "fields");
    check(decodedBits.inners[1].b == -256 && decodedBits.bigInner.c == 0x1234, "bits", "inner fields");

    static const uint8_t first[] = {1, 2, 3};
    static const uint8_t last[] = {255};

    Record record;
    Record decodedRecord;
    ZERO(record);
    ZERO(decodedRecord);
    record.id = 4000000000u;
    SET_TEXT(record.name, record.nameLength, "Pump 3 \xC2\xB7 west \xF0\x9F\x9A\xB0");
    record.severity = SeverityHigh;
    record.signed_ = SignedPositive;
    record.flags.active = true;
    record.flags.level = 6;
    SET_TEXT(record.labels[0].name, record.labels[0].nameLength, "inlet");
    record.labels[0].weight = 0.5f;
    SET_TEXT(record.labels[1].name, record.labels[1].nameLength, "");
    record.labels[1].weight = -3;
    record.labelsCount = 2;
    record.values[0] = -32768;
    record.values[1] = 0;
    record.values[2] = 32767;
    record.valuesCount = 3;
    record.trailer = 0xCAFE;
    SET_TEXT(record.pair[0].text, record.pair[0].length, "left");
    SET_TEXT(record.pair[1].text, record.pair[1].length, "");
    record.nested[0].bytes = first;
    record.nested[0].length = 3;
    record.nested[1].bytes = NULL;
    record.nested[1].length = 0;
    record.nested[2].bytes = last;
    record.nested[2].length = 1;
    record.nestedCount = 3;
    record.severities[0] = SeverityLow;
    record.severities[1] = SeverityMedium;
    record.severities[2] = SeverityHigh;
    record.severitiesCount = 3;
    record.matrix[0][0] = 1;
    record.matrix[0][1] = -2;
    record.matrix[1][0] = -3;
    record.matrix[1][1] = 4;
    CHECK_STRUCT(Record, "record", record, decodedRecord);
    check(equal(decodedRecord.name, decodedRecord.nameLength, "Pump 3 \xC2\xB7 west \xF0\x9F\x9A\xB0"), "record", "name");
    check(decodedRecord.labelsCount == 2 && equal(decodedRecord.labels[0].name, decodedRecord.labels[0].nameLength, "inlet"), "record", "labels");
    check(decodedRecord.labels[1].weight == -3, "record", "label weight");
    check(decodedRecord.valuesCount == 3 && decodedRecord.values[0] == -32768 && decodedRecord.values[2] == 32767, "record", "values");
    check(decodedRecord.nestedCount == 3 && decodedRecord.nested[1].length == 0 && decodedRecord.nested[2].bytes[0] == 255, "record", "nested");
    check(decodedRecord.severitiesCount == 3 && decodedRecord.severities[2] == SeverityHigh, "record", "severities");
    check(decodedRecord.signed_ == SignedPositive, "record", "enums");
    check(equal(decodedRecord.pair[0].text, decodedRecord.pair[0].length, "left") && decodedRecord.matrix[1][0] == -3, "record", "arrays");

    Ping ping;
    Ping decodedPing;
    ZERO(ping);
    ZERO(decodedPing);
    ping.sequence = 0xDEADBEEF;
    CHECK_STRUCT(Ping, "ping", ping, decodedPing);

    static const uint8_t payload[] = {0, 1, 254, 255};
    static const uint8_t seven[] = {7};
    static const uint8_t eightNine[] = {8, 9};
    Blob blob;
    Blob decodedBlob;
    ZERO(blob);
    ZERO(decodedBlob);
    blob.payload = payload;
    blob.payloadLength = 4;
    blob.chunks[0].bytes = seven;
    blob.chunks[0].length = 1;
    blob.chunks[2].bytes = eightNine;
    blob.chunks[2].length = 2;
    blob.chunksCount = 3;
    CHECK_STRUCT(Blob, "blob", blob, decodedBlob);
    check(decodedBlob.payloadLength == 4 && decodedBlob.payload[2] == 254 && decodedBlob.chunks[2].bytes[1] == 9, "blob", "fields");

    static const uint8_t log[] = {'o', 'k', 0, 255};
    Status status;
    Status decodedStatus;
    ZERO(status);
    ZERO(decodedStatus);
    status.identifier = 0x5E;
    status.version = 2;
    status.keys = 0x15;
    status.frames = 0x1234;
    status.log = log;
    status.logCount = 4;
    CHECK_STRUCT(Status, "status", status, decodedStatus);
    check(decodedStatus.keys == 0x15 && decodedStatus.frames == 0x1234 && decodedStatus.log[3] == 255, "status", "fields");
    check(decodedStatus.log == vector("status")->bytes + 6, "status", "log points into the input");

    KeyEvent keys[2];
    ZERO(keys);
    keys[0].key = 1;
    keys[0].kind = KeyKindPress;
    keys[1].key = 0x0203;
    keys[1].kind = KeyKindHold;
    Descriptor descriptor;
    Descriptor decodedDescriptor;
    ZERO(descriptor);
    ZERO(decodedDescriptor);
    descriptor.level = LevelWarning;
    descriptor.message = 0xBEEF;
    descriptor.keys = keys;
    descriptor.keysCount = 2;
    CHECK_STRUCT(Descriptor, "descriptor", descriptor, decodedDescriptor);
    check(decodedDescriptor.level == LevelWarning && decodedDescriptor.keysCount == 2, "descriptor", "fields");
    check(decodedDescriptor.keys[1].key == 0x0203 && decodedDescriptor.keys[1].kind == KeyKindHold, "descriptor", "keys");
    check((const uint8_t*)decodedDescriptor.keys == vector("descriptor")->bytes + 4, "descriptor", "keys point into the input");

    static const Point innerPoints[] = {{1, 2}};
    static const Point rightPoints[] = {{-1, -2}, {300, -300}};
    static const Point aPoints[] = {{11, 12}};
    static const Tagged tags[] = {{true, ModeFast, {-3, 7}}, {false, ModeSlow, {9, 10}}};
    Outer outer;
    Outer decodedOuter;
    ZERO(outer);
    ZERO(decodedOuter);
    SET_TEXT(outer.inner.name, outer.inner.nameLength, "inner");
    outer.inner.points = innerPoints;
    outer.inner.pointsCount = 1;
    SET_TEXT(outer.pair[0].name, outer.pair[0].nameLength, "left");
    SET_TEXT(outer.pair[1].name, outer.pair[1].nameLength, "right");
    outer.pair[1].points = rightPoints;
    outer.pair[1].pointsCount = 2;
    for (int row = 0; row < 2; row++) {
        for (int column = 0; column < 2; column++) {
            outer.grid[row][column].x = (int16_t)(1 + 4 * row + 2 * column);
            outer.grid[row][column].y = (int16_t)(2 + 4 * row + 2 * column);
        }
    }
    outer.tags = tags;
    outer.tagsCount = 2;
    SET_TEXT(outer.names[0].name, outer.names[0].nameLength, "a");
    outer.names[0].points = aPoints;
    outer.names[0].pointsCount = 1;
    SET_TEXT(outer.names[1].name, outer.names[1].nameLength, "bb");
    outer.namesCount = 2;
    outer.bigs[0].value = 0x1234;
    outer.bigs[1].value = 0xABCD;
    outer.bigsCount = 2;
    outer.deep.level.tags = tags;
    outer.deep.level.tagsCount = 1;
    outer.deep.level.mode = ModeSlow;
    CHECK_STRUCT(Outer, "outer", outer, decodedOuter);
    check(equal(decodedOuter.inner.name, decodedOuter.inner.nameLength, "inner") && decodedOuter.inner.points[0].y == 2, "outer", "inner");
    check(decodedOuter.pair[1].pointsCount == 2 && decodedOuter.pair[1].points[1].x == 300 && decodedOuter.pair[1].points[1].y == -300, "outer", "pair");
    check(decodedOuter.grid[1][0].x == 5 && decodedOuter.grid[1][1].y == 8, "outer", "grid");
    check(decodedOuter.tagsCount == 2 && decodedOuter.tags[0].on && decodedOuter.tags[0].mode == ModeFast && decodedOuter.tags[0].at.x == -3, "outer", "tags");
    check(decodedOuter.tags[1].mode == ModeSlow && decodedOuter.tags[1].at.y == 10, "outer", "second tag");
    check(decodedOuter.namesCount == 2 && decodedOuter.names[0].points[0].x == 11 && decodedOuter.names[1].pointsCount == 0, "outer", "names");
    check(decodedOuter.bigsCount == 2 && decodedOuter.bigs[0].value == 0x1234 && decodedOuter.bigs[1].value == 0xABCD, "outer", "bigs");
    check(decodedOuter.deep.level.tagsCount == 1 && decodedOuter.deep.level.tags[0].at.y == 7 && decodedOuter.deep.level.mode == ModeSlow, "outer", "deep");
    size_t tagsAt = (size_t)((const uint8_t*)decodedOuter.tags - vector("outer")->bytes);
    check(tagsAt < vector("outer")->length, "outer", "tags point into the input");
    uint8_t badMode[256];
    memcpy(badMode, vector("outer")->bytes, vector("outer")->length);
    badMode[tagsAt] |= 0x06;
    size_t outerEnd = 0;
    check(OuterFromBytes(&decodedOuter, badMode, vector("outer")->length, 0, &outerEnd) == PackedErrorUnknownValue, "outer", "unknown Mode in a list in place");

    Envelope envelope;
    Envelope decodedEnvelope;
    ZERO(envelope);
    envelope.type = EnvelopeTypeRecord;
    envelope.record = record;
    CHECK_SET(Envelope, "envelope-record", envelope, decodedEnvelope);
    check(decodedEnvelope.record.id == 4000000000u && decodedEnvelope.record.severities[1] == SeverityMedium, "envelope-record", "fields");
    ZERO(envelope);
    envelope.type = EnvelopeTypePing;
    envelope.ping.sequence = 7;
    CHECK_SET(Envelope, "envelope-ping", envelope, decodedEnvelope);

    Command command;
    Command decodedCommand;
    ZERO(command);
    command.type = CommandTypeToggle;
    command.toggle.on = true;
    command.toggle.channel = 9;
    command.toggle.duration = 0xBEEF;
    CHECK_SET(Command, "command-toggle", command, decodedCommand);
    check(decodedCommand.toggle.on && decodedCommand.toggle.channel == 9 && decodedCommand.toggle.duration == 0xBEEF, "command-toggle", "fields");
    check(decodedCommand.toggle.tag == 0, "command-toggle", "tag");
    const Toggle* toggle = ToggleView(vector("command-toggle")->bytes, vector("command-toggle")->length, 0);
    check(toggle != NULL && toggle->on && toggle->channel == 9 && toggle->duration == 0xBEEF, "command-toggle", "view");

    uint8_t prefixed[8] = {0xAB};
    size_t end = 0;
    check(CommandEncode(&command, prefixed + 1, sizeof prefixed - 1, &end) == PackedErrorNone && same(prefixed, end + 1, vector("prefixed-toggle")), "prefixed-toggle", "encoded");

    ZERO(command);
    command.type = CommandTypeRename;
    SET_TEXT(command.rename.name, command.rename.nameLength, "pump");
    CHECK_SET(Command, "command-rename", command, decodedCommand);
    check(equal(decodedCommand.rename.name, decodedCommand.rename.nameLength, "pump") && decodedCommand.rename.tag == 5, "command-rename", "fields");

    ZERO(command);
    command.type = CommandTypeDial;
    command.dial.level = -20;
    command.dial.fine = true;
    command.dial.severity = SeverityHigh;
    CHECK_SET(Command, "command-dial", command, decodedCommand);
    const Dial* dial = DialView(vector("command-dial")->bytes, vector("command-dial")->length, 0);
    check(dial != NULL && dial->level == -20 && dial->fine && dial->severity == SeverityHigh && dial->tag == 6, "command-dial", "view");

    ZERO(command);
    command.type = CommandTypeHalt;
    CHECK_SET(Command, "command-halt", command, decodedCommand);

    Signal signal;
    Signal decodedSignal;
    ZERO(signal);
    signal.type = SignalTypeBeacon;
    signal.beacon.urgent = true;
    signal.beacon.level = -7;
    signal.beacon.signed_ = (uint16_t)SignedPositive;
    signal.beacon.count = 0x1234;
    CHECK_SET(Signal, "signal-beacon", signal, decodedSignal);
    check(decodedSignal.beacon.level == -7 && (Signed)decodedSignal.beacon.signed_ == SignedPositive, "signal-beacon", "fields");
    ZERO(signal);
    signal.type = SignalTypeQuiet;
    CHECK_SET(Signal, "signal-quiet", signal, decodedSignal);

    Pulse pulse;
    Pulse decodedPulse;
    ZERO(pulse);
    pulse.type = PulseTypeHeartbeat;
    pulse.heartbeat.count = 42;
    CHECK_SET(Pulse, "pulse-heartbeat", pulse, decodedPulse);
    const Heartbeat* heartbeat = HeartbeatView(vector("pulse-heartbeat")->bytes, vector("pulse-heartbeat")->length, 0);
    check(heartbeat != NULL && heartbeat->count == 42, "pulse-heartbeat", "view");

    Display display;
    Display decodedDisplay;
    ZERO(display);
    display.type = DisplayTypeSetFrame;
    for (size_t index = 0; index < 192; index++) {
        display.setFrame.pixels[index] = (uint8_t)(index * 3 + 1);
    }
    CHECK_SET(Display, "display-setframe", display, decodedDisplay);
    check(decodedDisplay.setFrame.pixels[191] == (uint8_t)(191 * 3 + 1), "display-setframe", "fields");
    const SetFrame* frame = SetFrameView(vector("display-setframe")->bytes, vector("display-setframe")->length, 1);
    check(frame != NULL && frame->pixels[100] == (uint8_t)301, "display-setframe", "view");
    check(SetFrameView(vector("display-setframe")->bytes, 192, 1) == NULL, "display-setframe", "short view");
    ZERO(display);
    display.type = DisplayTypeRead;
    CHECK_SET(Display, "display-read", display, decodedDisplay);

    // The errors Go's decoders report.
    const uint8_t unknownCommand[] = {0x01};
    const uint8_t unknownSignal[] = {0xC0};
    const uint8_t shortPulse[] = {1, 2};
    const uint8_t longPulse[] = {1, 2, 3, 4, 5};
    uint8_t trailing[8];
    memcpy(trailing, vector("command-toggle")->bytes, 3);
    trailing[3] = 0;
    check(CommandDecode(&decodedCommand, NULL, 0) == PackedErrorEmpty, "errors", "empty");
    check(CommandDecode(&decodedCommand, unknownCommand, 1) == PackedErrorUnknownType, "errors", "unknown Command type");
    check(SignalDecode(&decodedSignal, unknownSignal, 1) == PackedErrorUnknownType, "errors", "unknown Signal type");
    check(CommandDecode(&decodedCommand, vector("command-toggle")->bytes, 2) == PackedErrorTruncated, "errors", "truncated Command");
    check(CommandDecode(&decodedCommand, trailing, 4) == PackedErrorTrailingBytes, "errors", "trailing Command");
    check(PulseDecode(&decodedPulse, shortPulse, 2) == PackedErrorTruncated, "errors", "truncated Pulse");
    check(PulseDecode(&decodedPulse, longPulse, 5) == PackedErrorTrailingBytes, "errors", "trailing Pulse");

    uint8_t badLevel[4];
    memcpy(badLevel, vector("descriptor")->bytes, 4);
    badLevel[0] = 9;
    badLevel[3] = 0;
    check(DescriptorFromBytes(&decodedDescriptor, badLevel, 4, 0, &end) == PackedErrorUnknownValue, "errors", "unknown Level");
    uint8_t badKind[7];
    memcpy(badKind, vector("descriptor")->bytes, 7);
    badKind[3] = 1;
    badKind[6] = 7;
    check(DescriptorFromBytes(&decodedDescriptor, badKind, 7, 0, &end) == PackedErrorUnknownValue, "errors", "unknown KeyKind in a list in place");
    uint8_t badDial[2];
    badDial[0] = vector("command-dial")->bytes[0];
    badDial[1] = (uint8_t)(vector("command-dial")->bytes[1] | 0x0C);
    check(CommandDecode(&decodedCommand, badDial, 2) == PackedErrorUnknownValue, "errors", "unknown Severity bits");

    uint8_t longLog[256];
    uint8_t bytes[300];
    ZERO(longLog);
    status.log = longLog;
    status.logCount = 256;
    check(StatusToBytes(&status, bytes, sizeof bytes, 0, &end) == PackedErrorTooLong, "errors", "too long");

    // Blob's Chunks holds at most 4: an empty payload and small, then 5 chunks.
    const uint8_t fiveChunks[] = {0, 0, 0, 0, 0, 5};
    check(BlobFromBytes(&decodedBlob, fiveChunks, sizeof fiveChunks, 0, &end) == PackedErrorTooLong, "errors", "over the maximum");
    blob.chunksCount = 5;
    check(BlobToBytes(&blob, bytes, sizeof bytes, 0, &end) == PackedErrorTooLong, "errors", "encoding over the maximum");

    if (failures > 0) {
        printf("%d failures\n", failures);
        return 1;
    }
    printf("ok, %d vectors\n", (int)vectorCount);
    return 0;
}
