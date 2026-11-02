// packed's layout in TypeScript: the reader and writer packed's TypeScript
// generator writes its modules against (typescript.go). Copy it into a
// project, or import it from where packed is, as generate.TypeScriptFile's
// runtime says. A reader reads
// straight off its bytes through a DataView, TextDecoder over each
// string's bytes, and throws RangeError past their end; integers of 64 bits
// are BigInts, bit runs BigInt words, bytes a Uint8Array.

const decoder = new TextDecoder('utf-8', { fatal: false });
const encoder = new TextEncoder();

export class PackedReader {
	bytes: Uint8Array;
	view: DataView;
	index: number;
	limit: number;

	// Reads bytes from index to limit.
	constructor(bytes: Uint8Array, index = 0, limit = bytes.length) {
		this.bytes = bytes;
		this.view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
		this.index = index;
		this.limit = limit;
	}

	take(count: number): number {
		const at = this.index;
		if (at + count > this.limit) throw new RangeError(`truncated: ${count} bytes needed, ${this.limit - at} left`);
		this.index = at + count;
		return at;
	}

	boolean(): boolean {
		return this.bytes[this.take(1)] !== 0;
	}

	uint8(): number {
		return this.bytes[this.take(1)];
	}

	int8(): number {
		return this.view.getInt8(this.take(1));
	}

	uint16(littleEndian: boolean): number {
		return this.view.getUint16(this.take(2), littleEndian);
	}

	int16(littleEndian: boolean): number {
		return this.view.getInt16(this.take(2), littleEndian);
	}

	uint32(littleEndian: boolean): number {
		return this.view.getUint32(this.take(4), littleEndian);
	}

	int32(littleEndian: boolean): number {
		return this.view.getInt32(this.take(4), littleEndian);
	}

	uint64(littleEndian: boolean): bigint {
		return this.view.getBigUint64(this.take(8), littleEndian);
	}

	int64(littleEndian: boolean): bigint {
		return this.view.getBigInt64(this.take(8), littleEndian);
	}

	float32(littleEndian: boolean): number {
		return this.view.getFloat32(this.take(4), littleEndian);
	}

	float64(littleEndian: boolean): number {
		return this.view.getFloat64(this.take(8), littleEndian);
	}

	// A bit run's word of size bytes, as a BigInt.
	word(size: number, littleEndian: boolean): bigint {
		const at = this.take(size);
		let value = 0n;
		for (let byte = 0; byte < size; byte++) {
			const shift = BigInt(8 * (littleEndian ? byte : size - 1 - byte));
			value |= BigInt(this.bytes[at + byte]) << shift;
		}
		return value;
	}

	// A length or count of prefix bytes.
	length(prefix: number, littleEndian: boolean): number {
		switch (prefix) {
			case 1:
				return this.uint8();
			case 2:
				return this.uint16(littleEndian);
			case 4:
				return this.uint32(littleEndian);
		}
		return Number(this.uint64(littleEndian));
	}

	text(prefix: number, littleEndian: boolean): string {
		const length = this.length(prefix, littleEndian);
		const at = this.take(length);
		return decoder.decode(this.bytes.subarray(at, at + length));
	}

	bytesOf(prefix: number, littleEndian: boolean): Uint8Array {
		const length = this.length(prefix, littleEndian);
		const at = this.take(length);
		return this.bytes.slice(at, at + length);
	}

	// A string of length bytes, padded with zero bytes.
	fixedText(length: number): string {
		const at = this.take(length);
		let end = at + length;
		while (end > at && this.bytes[end - 1] === 0) end--;
		return decoder.decode(this.bytes.subarray(at, end));
	}
}

export class PackedWriter {
	bytes: Uint8Array;
	view: DataView;
	size: number;

	// Writes after prefix, bytes the caller fills (such as a routing byte).
	constructor(prefix: ArrayLike<number> = []) {
		this.bytes = new Uint8Array(Math.max(64, prefix.length * 2));
		this.view = new DataView(this.bytes.buffer);
		this.bytes.set(prefix);
		this.size = prefix.length;
	}

	reserve(count: number): number {
		if (this.size + count <= this.bytes.length) return this.size;
		const bytes = new Uint8Array(Math.max(this.bytes.length * 2, this.size + count));
		bytes.set(this.bytes.subarray(0, this.size));
		this.bytes = bytes;
		this.view = new DataView(bytes.buffer);
		return this.size;
	}

	// Makes room for count bytes and returns where they go. Callers take
	// the place before they touch bytes or view, which a growth replaces.
	put(count: number): number {
		const at = this.reserve(count);
		this.size += count;
		return at;
	}

	boolean(value: boolean): void {
		const at = this.put(1);
		this.bytes[at] = value ? 1 : 0;
	}

	uint8(value: number): void {
		const at = this.put(1);
		this.view.setUint8(at, value);
	}

	int8(value: number): void {
		const at = this.put(1);
		this.view.setInt8(at, value);
	}

	uint16(value: number, littleEndian: boolean): void {
		const at = this.put(2);
		this.view.setUint16(at, value, littleEndian);
	}

	int16(value: number, littleEndian: boolean): void {
		const at = this.put(2);
		this.view.setInt16(at, value, littleEndian);
	}

	uint32(value: number, littleEndian: boolean): void {
		const at = this.put(4);
		this.view.setUint32(at, value, littleEndian);
	}

	int32(value: number, littleEndian: boolean): void {
		const at = this.put(4);
		this.view.setInt32(at, value, littleEndian);
	}

	uint64(value: bigint | number, littleEndian: boolean): void {
		const at = this.put(8);
		this.view.setBigUint64(at, BigInt(value), littleEndian);
	}

	int64(value: bigint | number, littleEndian: boolean): void {
		const at = this.put(8);
		this.view.setBigInt64(at, BigInt(value), littleEndian);
	}

	float32(value: number, littleEndian: boolean): void {
		const at = this.put(4);
		this.view.setFloat32(at, value, littleEndian);
	}

	float64(value: number, littleEndian: boolean): void {
		const at = this.put(8);
		this.view.setFloat64(at, value, littleEndian);
	}

	// A bit run's word of size bytes, from a BigInt.
	word(value: bigint, size: number, littleEndian: boolean): void {
		const at = this.put(size);
		for (let byte = 0; byte < size; byte++) {
			const shift = BigInt(8 * (littleEndian ? byte : size - 1 - byte));
			this.bytes[at + byte] = Number((value >> shift) & 0xffn);
		}
	}

	length(length: number, prefix: number, littleEndian: boolean): void {
		if (prefix < 8 && length > 2 ** (8 * prefix) - 1) throw new RangeError(`${length} does not fit a ${prefix}-byte length prefix`);
		switch (prefix) {
			case 1:
				return this.uint8(length);
			case 2:
				return this.uint16(length, littleEndian);
			case 4:
				return this.uint32(length, littleEndian);
		}
		this.uint64(length, littleEndian);
	}

	text(value: string, prefix: number, littleEndian: boolean): void {
		const encoded = encoder.encode(value);
		this.length(encoded.length, prefix, littleEndian);
		const at = this.put(encoded.length);
		this.bytes.set(encoded, at);
	}

	bytesOf(value: Uint8Array, prefix: number, littleEndian: boolean): void {
		this.length(value.length, prefix, littleEndian);
		const at = this.put(value.length);
		this.bytes.set(value, at);
	}

	// A string in length bytes, cut or padded with zero bytes.
	fixedText(value: string, length: number): void {
		const encoded = encoder.encode(value).subarray(0, length);
		const at = this.put(length);
		this.bytes.fill(0, at, at + length);
		this.bytes.set(encoded, at);
	}

	// The bytes written so far, a view of the writer's own.
	written(): Uint8Array {
		return this.bytes.subarray(0, this.size);
	}
}

// An enum's value name by its value, as the generated modules read it.
export function enumName<Name extends string>(names: readonly Name[], value: number, enumeration: string): Name {
	const name = names[value];
	if (name === undefined) throw new Error(`packed: unknown ${enumeration} ${value}`);
	return name;
}

// An enum's value by its name.
export function enumValue<Name extends string>(names: readonly Name[], name: Name): number {
	const value = names.indexOf(name);
	if (value < 0) throw new RangeError(`${name} is none of ${names.join(', ')}`);
	return value;
}
