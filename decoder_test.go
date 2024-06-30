package bottledlightning

import (
	"bytes"
	"hash"
	"hash/fnv"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkDecoder(b *testing.B) {
	var (
		buffer bytes.Buffer
		hasher hash.Hash32 = fnv.New32a()

		decoder *Decoder = NewDecoder(&buffer, hasher)

		e error
		i int
	)

	for i = 0; i < 1<<24; i++ {
		_, e = buffer.Write(
			[]byte{
				0b10100000, 0b00000000,
				0, 0,
				0x81, 0x1c, 0x9d, 0xc5,
			},
		)
		if e != nil {
			b.Error(e)
		}
	}

	b.ResetTimer()

	for i = 0; i < b.N; i++ {
		_, _, e = decoder.Decode()
		if e != nil {
			b.Error(e)
		}
	}

	return
}

func TestDecoder(t *testing.T) {
	const (
		keyString = "This is the real secret of life -- to be completely " +
			"engaged with what you are doing in the here and now. And " +
			"instead of calling it work, realize it is play."
		valString = "Alan Watts [1915-1973]"
	)

	var (
		e   error
		key []byte
		val []byte

		buffer bytes.Buffer
		hasher hash.Hash32 = fnv.New32a()

		decoder *Decoder = NewDecoder(&buffer, hasher)
	)

	_, e = buffer.Write([]byte{0b01100000, 0b10011100})
	if e != nil {
		return
	}

	_, e = buffer.Write([]byte{22})
	if e != nil {
		return
	}

	_, e = buffer.WriteString(keyString)
	if e != nil {
		return
	}

	_, e = buffer.WriteString(valString)
	if e != nil {
		return
	}

	_, e = buffer.Write([]byte{0xcb, 0xb7, 0x8d, 0x30})
	if e != nil {
		return
	}

	key, val, e = decoder.Decode()
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, keyString,
		string(key),
	)

	assert.Equal(t, valString,
		string(val),
	)

	_, _, e = decoder.Decode()

	assert.Equal(t, io.EOF,
		e,
	)

	return
}

func TestDecoderReadXCK(t *testing.T) {
	var (
		c bool
		e error
		k int
		x int

		buffer *bytes.Buffer = bytes.NewBuffer([]byte{0b11100001, 0b11111111})

		decoder *Decoder = NewDecoder(buffer, nil)
	)

	x, c, k, e = decoder.readXCK()
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, 3, x)

	assert.Equal(t, true, c)

	assert.Equal(t, 511, k)

	_, e = buffer.Write([]byte{0, 0})
	if e != nil {
		t.Error(e)
	}

	x, c, k, e = decoder.readXCK()
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, 4, x)

	assert.Equal(t, false, c)

	assert.Equal(t, 0, k)

	return
}

func TestDecoderReadV(t *testing.T) {
	var (
		e error
		v int

		buffer *bytes.Buffer = bytes.NewBuffer(
			[]byte{
				0xff,
				0xff, 0xff,
				0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff,
			},
		)

		decoder *Decoder = NewDecoder(buffer, nil)
	)

	v, e = decoder.readV(1)
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, 255, v)

	v, e = decoder.readV(2)
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, 65535, v)

	v, e = decoder.readV(3)
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, 16777215, v)

	v, e = decoder.readV(4)
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, 4294967295, v)

	return
}

func TestDecoderReadKey(t *testing.T) {
	var (
		e   error
		key []byte

		buffer *bytes.Buffer = bytes.NewBuffer([]byte{1, 2, 3, 4, 5})

		decoder *Decoder = NewDecoder(buffer, nil)
	)

	key, e = decoder.readKey(3)
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t,
		[]byte{1, 2, 3},
		key,
	)

	return
}

func TestDecoderReadVal(t *testing.T) {
	var (
		e   error
		val []byte

		buffer *bytes.Buffer = bytes.NewBuffer([]byte{5, 4, 3, 2, 1})

		decoder *Decoder = NewDecoder(buffer, nil)
	)

	val, e = decoder.readVal(3)
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t,
		[]byte{5, 4, 3},
		val,
	)

	return
}

func TestDecoderVerifyChecksum(t *testing.T) {
	var (
		key = []byte("Hello,")
		val = []byte("World!")

		buffer *bytes.Buffer = bytes.NewBuffer([]byte{0x7a, 0xf8, 0xa9, 0xf6})
		hasher hash.Hash32   = fnv.New32a()

		decoder *Decoder = NewDecoder(buffer, hasher)
	)

	assert.NoError(t,
		decoder.verifyChecksum(key, val),
	)

	assert.Error(t,
		decoder.verifyChecksum(val, key),
	)

	return
}
