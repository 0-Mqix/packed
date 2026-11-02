// The C++ layer of the generated header (messages.h): methods, enum
// classes, and text and bytes as the standard library's views, against the
// bytes Go wrote to ../testdata/vectors.txt. compliance.c checks every
// sample through the C functions; this checks that C++ reaches the same
// bytes its own way. TestHeader compiles it with each host C++ compiler and
// runs it with the vectors' path.
#include "messages.h"

#include <stdio.h>
#include <stdlib.h>

#include <span>
#include <string_view>

static uint8_t vectorBytes[256];
static size_t vectorLength;
static int failures;

static void check(bool condition, const char* name, const char* what) {
    if (!condition) {
        printf("FAIL %s: %s\n", name, what);
        failures++;
    }
}

// load reads the vector named name from the file at path into vectorBytes,
// which every load reuses: a struct decoded from it points into it, so it
// is checked before the next load.
static void load(const char* path, const char* name) {
    FILE* file = fopen(path, "r");
    if (file == nullptr) {
        printf("FAIL cannot open %s\n", path);
        exit(1);
    }
    char line[40];
    char hex[600];
    while (fscanf(file, "%39s %599s", line, hex) == 2) {
        if (strcmp(line, name) != 0) {
            continue;
        }
        vectorLength = strlen(hex) / 2;
        for (size_t index = 0; index < vectorLength; index++) {
            unsigned byte = 0;
            sscanf(hex + 2 * index, "%2x", &byte);
            vectorBytes[index] = static_cast<uint8_t>(byte);
        }
        fclose(file);
        return;
    }
    printf("FAIL no vector %s\n", name);
    exit(1);
}

static bool same(const uint8_t* bytes, size_t length) {
    return length == vectorLength && memcmp(bytes, vectorBytes, length) == 0;
}

int main(int argumentCount, char** arguments) {
    if (argumentCount != 2) {
        printf("usage: compliance vectors.txt\n");
        return 2;
    }
    const char* path = arguments[1];
    uint8_t bytes[256];
    size_t end = 0;

    KeyEvent keys[2] = {};
    keys[0].key = 1;
    keys[0].kind = KeyKind::Press;
    keys[1].key = 0x0203;
    keys[1].kind = KeyKind::Hold;
    Descriptor descriptor{};
    descriptor.level = Level::Warning;
    descriptor.message = 0xBEEF;
    descriptor.keys = keys;
    descriptor.keysCount = 2;
    load(path, "descriptor");
    check(descriptor.size() == vectorLength, "descriptor", "size");
    check(descriptor.toBytes(bytes, sizeof bytes, 0, end) == PackedErrorNone && same(bytes, end), "descriptor", "encoded");
    Descriptor decodedDescriptor{};
    check(decodedDescriptor.fromBytes(vectorBytes, vectorLength, 0, end) == PackedErrorNone && end == vectorLength, "descriptor", "decoded");
    check(decodedDescriptor.level == Level::Warning && decodedDescriptor.keys[1].kind == KeyKind::Hold, "descriptor", "fields");
    uint32_t keySum = 0;
    for (const KeyEvent& event : std::span(decodedDescriptor.keys, decodedDescriptor.keysCount)) {
        keySum += event.key;
    }
    check(keySum == 1 + 0x0203, "descriptor", "keys as a span");
    const KeyEvent* key = KeyEvent::view(vectorBytes, vectorLength, 4);
    check(key != nullptr && key->kind == KeyKind::Press, "descriptor", "view");

    static const uint8_t log[] = {'o', 'k', 0, 255};
    Status status{};
    status.identifier = 0x5E;
    status.version = 2;
    status.keys = 0x15;
    status.frames = 0x1234;
    status.log = log;
    status.logCount = 4;
    load(path, "status");
    check(status.toBytes(bytes, sizeof bytes, 0, end) == PackedErrorNone && same(bytes, end), "status", "encoded");
    Status decodedStatus{};
    check(decodedStatus.fromBytes(vectorBytes, vectorLength, 0, end) == PackedErrorNone, "status", "decoded");
    check(decodedStatus.log[1] == 'k' && decodedStatus.logCount == 4, "status", "log");

    static const uint8_t first[] = {1, 2, 3};
    static const uint8_t last[] = {255};
    Record record{};
    record.id = 4000000000u;
    record.name = "Pump 3 \xC2\xB7 west \xF0\x9F\x9A\xB0";
    record.nameLength = strlen(record.name);
    record.severity = Severity::High;
    record.signed_ = Signed::Positive;
    record.flags.active = true;
    record.flags.level = 6;
    record.labels[0] = Label{"inlet", 5, 0.5f};
    record.labels[1] = Label{"", 0, -3};
    record.labelsCount = 2;
    record.values[0] = -32768;
    record.values[1] = 0;
    record.values[2] = 32767;
    record.valuesCount = 3;
    record.trailer = 0xCAFE;
    record.pair[0] = PackedText{"left", 4};
    record.pair[1] = PackedText{"", 0};
    record.nested[0] = PackedBytes{first, 3};
    record.nested[1] = PackedBytes{nullptr, 0};
    record.nested[2] = PackedBytes{last, 1};
    record.nestedCount = 3;
    record.severities[0] = Severity::Low;
    record.severities[1] = Severity::Medium;
    record.severities[2] = Severity::High;
    record.severitiesCount = 3;
    record.matrix[0][0] = 1;
    record.matrix[0][1] = -2;
    record.matrix[1][0] = -3;
    record.matrix[1][1] = 4;

    Envelope envelope{};
    envelope.type = Envelope::Type::Record;
    envelope.record = record;
    load(path, "envelope-record");
    check(envelope.size() == vectorLength, "envelope-record", "size");
    check(envelope.encode(bytes, sizeof bytes, end) == PackedErrorNone && same(bytes, end), "envelope-record", "encoded");
    Envelope decodedEnvelope{};
    check(Envelope::decode(vectorBytes, vectorLength, decodedEnvelope) == PackedErrorNone, "envelope-record", "decoded");
    const Record& decoded = decodedEnvelope.record;
    check(decodedEnvelope.type == EnvelopeType::Record && decoded.signed_ == Signed::Positive, "envelope-record", "type");
    check(std::string_view(decoded.name, decoded.nameLength) == "Pump 3 \xC2\xB7 west \xF0\x9F\x9A\xB0", "envelope-record", "name");
    check(decoded.labels[1].weight == -3 && decoded.labels[0].name[4] == 't', "envelope-record", "labels");
    std::string_view left = decoded.pair[0];
    check(left == "left", "envelope-record", "text as a string_view");
    std::span<const uint8_t> inner = decoded.nested[2];
    check(inner.size() == 1 && inner[0] == 255, "envelope-record", "bytes as a span");
    size_t valueCount = 0;
    for (int16_t value : std::span(decoded.values, decoded.valuesCount)) {
        valueCount += value != 1;
    }
    check(valueCount == 3 && decoded.severities[1] == Severity::Medium, "envelope-record", "lists");

    Display display{};
    display.type = Display::Type::SetFrame;
    for (size_t index = 0; index < SetFrame::fixedSize; index++) {
        display.setFrame.pixels[index] = static_cast<uint8_t>(index * 3 + 1);
    }
    load(path, "display-setframe");
    check(display.encode(bytes, sizeof bytes, end) == PackedErrorNone && same(bytes, end), "display-setframe", "encoded");
    Display decodedDisplay{};
    check(Display::decode(vectorBytes, vectorLength, decodedDisplay) == PackedErrorNone, "display-setframe", "decoded");
    check(decodedDisplay.type == Display::Type::SetFrame && decodedDisplay.setFrame.pixels[2] == 7, "display-setframe", "fields");
    const SetFrame* frame = SetFrame::view(vectorBytes, vectorLength, 1);
    check(frame != nullptr && frame->pixels[191] == static_cast<uint8_t>(191 * 3 + 1), "display-setframe", "view");
    size_t pixels = 0;
    for (uint8_t pixel : frame->pixels) {
        pixels += pixel == static_cast<uint8_t>(pixels * 3 + 1);
    }
    check(pixels == 192, "display-setframe", "range-for over a fixed array");

    load(path, "outer");
    Outer decodedOuter{};
    check(decodedOuter.fromBytes(vectorBytes, vectorLength, 0, end) == PackedErrorNone && end == vectorLength, "outer", "decoded");
    int modes = 0;
    for (const Tagged& tag : std::span(decodedOuter.tags, decodedOuter.tagsCount)) {
        modes += static_cast<int>(tag.mode);
    }
    check(modes == static_cast<int>(Mode::Fast) + static_cast<int>(Mode::Slow), "outer", "tags as a span");
    check(std::string_view(decodedOuter.pair[1].name, decodedOuter.pair[1].nameLength) == "right", "outer", "pair");
    check(decodedOuter.pair[1].points[1].x == 300 && decodedOuter.grid[1][1].y == 8, "outer", "nested values");
    check(decodedOuter.deep.level.mode == Mode::Slow && decodedOuter.deep.level.tags[0].at.x == -3, "outer", "deep");
    check(decodedOuter.toBytes(bytes, sizeof bytes, 0, end) == PackedErrorNone && same(bytes, end), "outer", "encoded again");

    Command command{};
    command.type = CommandType::Dial;
    command.dial.level = -20;
    command.dial.fine = true;
    command.dial.severity = Severity::High;
    load(path, "command-dial");
    check(command.encode(bytes, sizeof bytes, end) == PackedErrorNone && same(bytes, end), "command-dial", "encoded");
    const Dial* dial = Dial::view(vectorBytes, vectorLength);
    check(dial != nullptr && dial->severity == Severity::High && dial->level == -20, "command-dial", "view");

    if (failures > 0) {
        printf("%d failures\n", failures);
        return 1;
    }
    printf("ok\n");
    return 0;
}
