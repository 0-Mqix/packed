// What a microcontroller does with messages.h, built for an ATtiny88 as C
// with avr-gcc and as C++ with avr-g++ (TestAVR): decode a Display command
// from a receive buffer, view its frame in place, and answer with a Status
// or a Descriptor. It only has to build and fit; the bytes are checked on
// the host by compliance.c.
#include "messages.h"

static uint8_t received[1 + SetFrameFixedSize];
static volatile uint8_t receivedLength;
static uint8_t sent[32];
static volatile uint8_t sentLength;
static volatile uint8_t pixel;
static const uint8_t log[4] = {'o', 'k'};

static void send(PackedError error, size_t end) {
    sentLength = error == PackedErrorNone ? (uint8_t)end : 0;
}

int main(void) {
    uint8_t frames = 0;
    for (;;) {
        size_t length = receivedLength;
        if (length == 0) {
            continue;
        }
        receivedLength = 0;

        // A frame is viewed where it arrived, without a 192-byte copy.
        if (received[0] == (uint8_t)DisplayTypeSetFrame) {
            const SetFrame* frame = SetFrameView(received, length, 1);
            if (frame != NULL && length == 1 + SetFrameFixedSize) {
                pixel = frame->pixels[0];
                frames++;
            }
            continue;
        }

        size_t end = 0;
        Display display;
        PackedError error = DisplayDecode(&display, received, length);
        if (error != PackedErrorNone || display.type != DisplayTypeRead) {
            KeyEvent keys[1];
            keys[0].key = 1;
            keys[0].kind = KeyKindPress;
            Descriptor descriptor;
            descriptor.level = LevelError;
            descriptor.message = (uint16_t)error;
            descriptor.keys = keys;
            descriptor.keysCount = 1;
            send(DescriptorToBytes(&descriptor, sent, sizeof sent, 0, &end), end);
            continue;
        }

        Status status;
        status.identifier = 0x5E;
        status.version = 1;
        status.keys = pixel & 0x1F;
        status.frames = frames;
        status.log = log;
        status.logCount = sizeof log;
        send(StatusToBytes(&status, sent, sizeof sent, 0, &end), end);
    }
}
