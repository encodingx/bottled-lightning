package bottledlightning

import (
	"bytes"
	"hash"
	"hash/fnv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkEncoder(b *testing.B) {
	var (
		buffer bytes.Buffer
		hasher hash.Hash32 = fnv.New32a()

		encoder *Encoder = NewEncoder(&buffer, hasher)

		e   error
		i   int
		key []byte
		val []byte
	)

	buffer.Grow(10)

	b.ResetTimer()

	for i = 0; i < b.N; i++ {
		e = encoder.Encode(key, val)
		if e != nil {
			b.Error(e)
		}

		buffer.Reset()
	}

	return
}

func TestEncoder(t *testing.T) {
	var (
		buffer bytes.Buffer
		hasher hash.Hash32 = fnv.New32a()

		key = []byte("Alan Watts [1915-1973]")
		val = []byte("We are living in a culture entirely hypnotized by the " +
			"illusion of time, in which the so-called present moment is " +
			"felt as nothing but an infintesimal hairline between an " +
			"all-powerfully causative past and an absorbingly important " +
			"future. We have no present. Our consciousness is almost " +
			"completely preoccupied with memory and expectation. We do not " +
			"realize that there never was, is, nor will be any other " +
			"experience than present experience. We are therefore out of " +
			"touch with reality. We confuse the world as talked about, " +
			"described, and measured with the world which actually is. We " +
			"are sick with a fascination for the useful tools of names and " +
			"numbers, of symbols, signs, conceptions and ideas.")

		encoder *Encoder = NewEncoder(&buffer, hasher)

		e    error
		read []byte
	)

	assert.NoError(t,
		encoder.Encode(key, val),
	)

	read = make([]byte, 4)

	_, e = buffer.Read(read)
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t,
		[]byte{
			0b10100000, 0b00010110, // x = 2, c = 1, k = 22
			0b00000010, 0b10110101, // v = 693
		},
		read,
	)

	read = make([]byte, 22)

	_, e = buffer.Read(read)
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, key, read)

	read = make([]byte, 693)

	_, e = buffer.Read(read)
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, val, read)

	read = make([]byte, 4)

	_, e = buffer.Read(read)
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t,
		[]byte{0xdb, 0x1a, 0x20, 0x3e},
		read,
	)

	assert.Equal(t, 0,
		buffer.Len(),
	)

	return
}

func TestEncoderValidateLens(t *testing.T) {
	var (
		buffer bytes.Buffer
		key    []byte
		val    []byte

		encoder *Encoder = NewEncoder(&buffer, nil)
	)

	key = make([]byte, 512)

	assert.Error(t,
		encoder.validateLens(key, val),
	)

	key = make([]byte, 511)

	assert.NoError(t,
		encoder.validateLens(key, val),
	)

	val = make([]byte, 4294967296)

	assert.NoError(t,
		encoder.validateLens(key, val),
	)

	val = make([]byte, 4294967297)

	assert.Error(t,
		encoder.validateLens(key, val),
	)

	return
}

func TestEncoderWriteXCK(t *testing.T) {
	var (
		buffer bytes.Buffer
		key    = make([]byte, 341)
		val    = make([]byte, 65536)

		encoder *Encoder = NewEncoder(&buffer, nil)
	)

	assert.NoError(t,
		encoder.writeXCK(key, val),
	)

	assert.Equal(t, []byte{0b11000001, 0b01010101},
		buffer.Bytes(),
	)

	buffer.Reset()

	key = make([]byte, 170)
	val = make([]byte, 16777216)

	encoder = NewEncoder(&buffer,
		fnv.New32a(),
	)

	assert.NoError(t,
		encoder.writeXCK(key, val),
	)

	assert.Equal(t, []byte{0b00100000, 0b10101010},
		buffer.Bytes(),
	)

	return
}

func TestEncoderWriteV(t *testing.T) {
	var (
		buffer bytes.Buffer
		val    = make([]byte, 1)

		encoder *Encoder = NewEncoder(&buffer, nil)
	)

	assert.NoError(t,
		encoder.writeV(val),
	)

	assert.Equal(t, []byte{1},
		buffer.Bytes(),
	)

	buffer.Reset()

	val = make([]byte, 256)

	assert.NoError(t,
		encoder.writeV(val),
	)

	assert.Equal(t, []byte{1, 0},
		buffer.Bytes(),
	)

	buffer.Reset()

	val = make([]byte, 65536)

	assert.NoError(t,
		encoder.writeV(val),
	)

	assert.Equal(t, []byte{1, 0, 0},
		buffer.Bytes(),
	)

	buffer.Reset()

	val = make([]byte, 16777216)

	assert.NoError(t,
		encoder.writeV(val),
	)

	assert.Equal(t, []byte{1, 0, 0, 0},
		buffer.Bytes(),
	)

	return
}

func TestEncoderWriteChecksum(t *testing.T) {
	var (
		buffer bytes.Buffer

		encoder *Encoder = NewEncoder(&buffer, fnv.New32a())
		key              = []byte("@url")
		val              = []byte(
			"https://github.com/cute-capacitor/bottled-lightning")
	)

	assert.NoError(t,
		encoder.writeChecksum(key, val),
	)

	assert.Equal(t, []byte{0x7c, 0xe5, 0x87, 0x76},
		buffer.Bytes(),
	)

	return
}

func TestFindX(t *testing.T) {
	var (
		s []byte
	)

	assert.Equal(t, 1,
		findX(s),
	)

	s = make([]byte, 255)

	assert.Equal(t, 1,
		findX(s),
	)

	s = make([]byte, 256)

	assert.Equal(t, 2,
		findX(s),
	)

	s = make([]byte, 65535)

	assert.Equal(t, 2,
		findX(s),
	)

	s = make([]byte, 65536)

	assert.Equal(t, 3,
		findX(s),
	)

	s = make([]byte, 16777215)

	assert.Equal(t, 3,
		findX(s),
	)

	s = make([]byte, 16777216)

	assert.Equal(t, 4,
		findX(s),
	)

	s = make([]byte, 4294967295)

	assert.Equal(t, 4,
		findX(s),
	)

	s = make([]byte, 4294967296)

	assert.Panics(t,
		func() { findX(s) },
	)

	return
}
