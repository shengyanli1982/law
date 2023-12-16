package buffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBytesBuffer_Standard(t *testing.T) {
	p := NewExtraBufferPool(0)
	assert.NotNil(t, p)

	b := p.Get()
	assert.NotNil(t, b)
	assert.NotNil(t, b.Buffer())
	assert.Equal(t, 0, b.Buffer().Len())
	assert.Equal(t, int64(0), b.UpdateAt())

	b.Buffer().Write([]byte("hello"))
	assert.Equal(t, 5, b.Buffer().Len()) // 5 bytes
	assert.Equal(t, int64(0), b.UpdateAt())
	assert.Equal(t, []byte("hello"), b.Buffer().Bytes())

	b.SetUpdateAt(123)
	assert.Equal(t, int64(123), b.UpdateAt())

	p.Put(b)
}
