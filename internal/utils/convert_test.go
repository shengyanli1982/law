package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvert_BytesToString(t *testing.T) {
	assert.Equal(t, "hello", BytesToString([]byte("hello")))
}

func TestConvert_StringToBytes(t *testing.T) {
	assert.Equal(t, []byte("hello"), StringToBytes("hello"))
}
