// The generated TypeScript (output.ts) against the bytes Go wrote to
// ../testdata/vectors.txt: the same samples as compliance_test.go,
// kotlin/ComplianceTest.kt and the Swift tests build, encoded and decoded.
// TestTypeScript runs it with node --test, which strips output.ts's types.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';
import { Beacon, Bits, Blob, Command, Descriptor, Display, Envelope, Label, Outer, Ping, Primitives, Pulse, Record, Signal, Status } from './output.ts';
import { PackedReader, PackedWriter } from '../../../typescript/packed.ts';

const vectors = new Map(
	readFileSync(new URL('../testdata/vectors.txt', import.meta.url), 'utf8')
		.trim()
		.split('\n')
		.map((line) => {
			const [name, hex] = line.split(' ');
			return [name, Uint8Array.from(hex.match(/../g) ?? [], (byte) => parseInt(byte, 16))];
		}),
);

const primitives = {
	flag: true,
	small: -5,
	byte: 250,
	short: -12345,
	unsignedShort: 54321,
	integer: -123456789,
	unsignedInteger: 3123456789,
	long: -(2n ** 63n) + 7n,
	unsignedLong: 2n ** 64n - 1n - 3n,
	single: -1.25,
	double: Math.PI,
	littleInteger: 0x01020304,
	code: 'ab·',
};

const bits = {
	active: true,
	mode: 9,
	delta: -11,
	wide: 0x0abcdef012345678n,
	straddle: -54321,
	tail: 1500,
	inner: { a: 5, b: -200, c: 0xbeef },
	bigInner: { a: 3, b: 255, c: 0x1234 },
	inners: [
		{ a: 1, b: -1, c: 1 },
		{ a: 7, b: -256, c: 65535 },
	],
};

const record = {
	id: 4000000000,
	name: 'Pump 3 · west 🚰',
	severity: 'High',
	signed: 'Positive',
	flags: { active: true, level: 6 },
	labels: [
		{ name: 'inlet', weight: 0.5 },
		{ name: '', weight: -3 },
	],
	values: [-32768, 0, 32767],
	trailer: 0xcafe,
	pair: ['left', ''],
	nested: [Uint8Array.of(1, 2, 3), Uint8Array.of(), Uint8Array.of(255)],
	severities: ['Low', 'Medium', 'High'],
	matrix: [
		[1, -2],
		[-3, 4],
	],
};

const ping = { sequence: 0xdeadbeef };

const blob = { payload: Uint8Array.of(0, 1, 254, 255), small: Uint8Array.of(), chunks: [Uint8Array.of(7), Uint8Array.of(), Uint8Array.of(8, 9)] };

const status = { identifier: 0x5e, version: 2, keys: 0x15, frames: 0x1234, log: Uint8Array.of(0x6f, 0x6b, 0, 255) };

const descriptor = {
	level: 'Warning',
	message: 0xbeef,
	keys: [
		{ key: 1, kind: 'Press' },
		{ key: 0x0203, kind: 'Hold' },
	],
};

// Structs nested every way, as outer() in compliance_test.go builds them.
const point = (x, y) => ({ x, y });
const named = (name, ...points) => ({ name, points });
const tag = { on: true, mode: 'Fast', at: point(-3, 7) };
const outer = {
	inner: named('inner', point(1, 2)),
	pair: [named('left'), named('right', point(-1, -2), point(300, -300))],
	grid: [
		[point(1, 2), point(3, 4)],
		[point(5, 6), point(7, 8)],
	],
	tags: [tag, { on: false, mode: 'Slow', at: point(9, 10) }],
	names: [named('a', point(11, 12)), named('bb')],
	bigs: [{ value: 0x1234 }, { value: 0xabcd }],
	deep: { level: { tags: [tag], mode: 'Slow' } },
};

// Pixel i is 3i+1, as frame() in compliance_test.go builds it.
const frame = { pixels: Array.from({ length: 192 }, (_, index) => (index * 3 + 1) & 0xff) };

function encode(type, value) {
	const writer = new PackedWriter();
	type.write(writer, value);
	return writer.written();
}

function decode(type, bytes) {
	const reader = new PackedReader(bytes);
	const value = type.read(reader);
	assert.equal(reader.index, bytes.length);
	return value;
}

for (const [name, type, value] of [
	['primitives', Primitives, primitives],
	['bits', Bits, bits],
	['record', Record, record],
	['ping', Ping, ping],
	['blob', Blob, blob],
	['status', Status, status],
	['descriptor', Descriptor, descriptor],
	['outer', Outer, outer],
]) {
	test(`${name} matches Go`, () => {
		assert.deepEqual([...encode(type, value)], [...vectors.get(name)]);
		assert.deepEqual(decode(type, vectors.get(name)), value);
	});
}

test('envelopes match Go', () => {
	assert.deepEqual([...Envelope.encode({ variant: 'Record', message: record })], [...vectors.get('envelope-record')]);
	assert.deepEqual([...Envelope.encode({ variant: 'Ping', message: { sequence: 7 } })], [...vectors.get('envelope-ping')]);
	assert.deepEqual(Envelope.decode(vectors.get('envelope-record')), { variant: 'Record', message: record });
	assert.deepEqual(Envelope.decode(vectors.get('envelope-ping')), { variant: 'Ping', message: { sequence: 7 } });
});

test('displays match Go', () => {
	assert.deepEqual([...Display.encode({ variant: 'SetFrame', message: frame })], [...vectors.get('display-setframe')]);
	assert.deepEqual([...Display.encode({ variant: 'Read', message: {} })], [...vectors.get('display-read')]);
	assert.deepEqual(Display.decode(vectors.get('display-setframe')), { variant: 'SetFrame', message: frame });
	assert.deepEqual(Display.decode(vectors.get('display-read')), { variant: 'Read', message: {} });
});

test('an envelope rejects bad input', () => {
	const encoded = Envelope.encode({ variant: 'Record', message: record });
	assert.throws(() => Envelope.decode(Uint8Array.of()), /empty Envelope message/);
	assert.throws(() => Envelope.decode(Uint8Array.of(99, 1, 2)), /unknown Envelope type 99/);
	assert.throws(() => Envelope.decode(encoded.slice(0, encoded.length - 3)), /truncated Envelope type 7/);
	assert.throws(() => Envelope.decode(Uint8Array.from([...encoded, 0])), /Envelope type 7 has 1 trailing bytes/);
});

const toggle = { on: true, channel: 9, duration: 0xbeef };

// The messages of the sets with TagBits, as taggedSamples() builds them.
const commands = {
	'command-toggle': ['Toggle', toggle],
	'command-rename': ['Rename', { name: 'pump' }],
	'command-dial': ['Dial', { level: -20, fine: true, severity: 'High' }],
	'command-halt': ['Halt', {}],
};

const beacon = { urgent: true, level: -7, signed: 'Positive', count: 0x1234 };

test('tag bits match Go', () => {
	for (const [name, [variant, message]] of Object.entries(commands)) {
		assert.deepEqual([...Command.encode({ variant, message })], [...vectors.get(name)], name);
		assert.deepEqual(Command.decode(vectors.get(name)), { variant, message }, name);
	}
	assert.deepEqual([...Signal.encode({ variant: 'Beacon', message: beacon })], [...vectors.get('signal-beacon')]);
	assert.deepEqual(Signal.decode(vectors.get('signal-beacon')), { variant: 'Beacon', message: beacon });
	assert.deepEqual([...Signal.encode({ variant: 'Quiet', message: {} })], [...vectors.get('signal-quiet')]);
	assert.deepEqual(Signal.decode(vectors.get('signal-quiet')), { variant: 'Quiet', message: {} });
	assert.deepEqual([...Pulse.encode({ variant: 'Heartbeat', message: { count: 42 } })], [...vectors.get('pulse-heartbeat')]);
	assert.deepEqual(Pulse.decode(vectors.get('pulse-heartbeat')), { variant: 'Heartbeat', message: { count: 42 } });
});

test('a prefix and an index match Go', () => {
	const prefixed = Command.encode({ variant: 'Toggle', message: toggle }, [0xab]);
	assert.deepEqual([...prefixed], [...vectors.get('prefixed-toggle')]);
	assert.deepEqual(Command.decode(prefixed, 1), { variant: 'Toggle', message: toggle });
	assert.deepEqual([...Envelope.encode({ variant: 'Ping', message: { sequence: 7 } }, [1, 2]).subarray(2)], [...vectors.get('envelope-ping')]);
	assert.deepEqual(Envelope.decode(Uint8Array.from([1, 2, ...vectors.get('envelope-ping')]), 2), { variant: 'Ping', message: { sequence: 7 } });
});

test('tag bits reject bad input', () => {
	const encoded = Command.encode({ variant: 'Toggle', message: toggle });
	assert.throws(() => Command.decode(Uint8Array.of()), /empty Command message/);
	assert.throws(() => Command.decode(encoded, encoded.length), /empty Command message/);
	assert.throws(() => Command.decode(Uint8Array.of(1)), /unknown Command type 1/);
	assert.throws(() => Signal.decode(Uint8Array.of(0xc0)), /unknown Signal type 3/);
	assert.throws(() => Command.decode(encoded.slice(0, 2)), /truncated Command type 0/);
	assert.throws(() => Command.decode(Uint8Array.from([...encoded, 0])), /Command type 0 has 1 trailing bytes/);
	assert.throws(() => Pulse.decode(Uint8Array.of(1, 2)), /truncated Pulse type 0/);
	assert.throws(() => Pulse.decode(Uint8Array.of(1, 2, 3, 4, 5)), /Pulse type 0 has 1 trailing bytes/);
	// Dial's Severity is bits 10 and 11; 3 has no value.
	const dial = vectors.get('command-dial').slice();
	dial[1] |= 0x0c;
	assert.throws(() => Command.decode(dial), /unknown Severity 3/);
	for (const [name, [variant, message]] of Object.entries(commands)) {
		const bytes = Command.encode({ variant, message });
		for (let length = 0; length < bytes.length; length++) assert.throws(() => Command.decode(bytes.slice(0, length)), Error, name);
	}
});

test('a length prefix that overflows fails', () => {
	assert.throws(() => encode(Label, { name: 'x'.repeat(256), weight: 0 }), RangeError);
});
