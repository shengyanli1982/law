package law

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var largeBytes = []byte("#Wqt2ZcQmVzk4KZJPmPamr3cBLNEf5dex2N2RkqhS3E7G6PWnzFEukskx5Z822mZd7")

type callback struct {
	t *testing.T
}

func (c *callback) OnWriteFailed(b []byte, err error) {
	fmt.Printf("## callback.OnWriteFailed(%s, %v)\n", b, err)
	assert.Equal(c.t, b, largeBytes, "Expected bytes")
	assert.ErrorIs(c.t, err, errorWriteFailed, "Expected error")
}

var errorWriteFailed = errors.New("write context failed")

type faultyWriter struct{}

func (fw *faultyWriter) Write(p []byte) (n int, err error) {
	fmt.Printf("!! faultyWriter.Write(%s)\n", p)
	return 0, errorWriteFailed
}

func TestWriteAsyncer_Standard(t *testing.T) {
	buff := bytes.NewBuffer(make([]byte, 0, 1024))

	w := NewWriteAsyncer(buff, nil)

	_, err := w.Write([]byte("hello"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("world"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("!!!"))
	assert.Nil(t, err)

	w.Stop()

	assert.Equal(t, "helloworld!!!", buff.String())
}

func TestWriteAsyncer_WaitForIdleSync(t *testing.T) {
	buff := bytes.NewBuffer(make([]byte, 0, 1024))

	w := NewWriteAsyncer(buff, nil)
	defer w.Stop()

	_, err := w.Write([]byte("hello"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("world"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("!!!"))
	assert.Nil(t, err)

	time.Sleep(time.Second * 6)

	assert.Equal(t, "helloworld!!!", buff.String())
}

func TestWriteAsyncer_EarlyShutdown(t *testing.T) {
	buff := bytes.NewBuffer(make([]byte, 0, 1024))

	w := NewWriteAsyncer(buff, nil)

	_, err := w.Write([]byte("hello"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("world"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("!!!"))
	assert.Nil(t, err)

	w.Stop()

	assert.Equal(t, "helloworld!!!", buff.String())

	_, err = w.Write([]byte("stop"))

	assert.ErrorIs(t, err, ErrorWriteAsyncerIsClosed, "Expected error")
	assert.Equal(t, "helloworld!!!", buff.String())
}

func TestWriteAsyncer_OnWriteFailed(t *testing.T) {

	// Will print callback.OnWriteFailed 10 times, faultyWriter.Write 1 time, buff is 66 bytes
	t.Run("Message large than bufferSize", func(t *testing.T) {
		conf := NewConfig().WithCallback(&callback{t: t}).WithBufferSize(60)

		w := NewWriteAsyncer(&faultyWriter{}, conf)
		defer w.Stop()

		for i := 0; i < 10; i++ {
			_, err := w.Write(largeBytes)
			assert.Nil(t, err)
		}

		fmt.Println(">>> Error return by bufio Write method")

		time.Sleep(time.Second)
	})

	// Will print callback.OnWriteFailed 1 times, faultyWriter.Write 1 time, buff is 594 bytes (Because of the buffer size is 600)
	t.Run("Message less than bufferSize", func(t *testing.T) {
		conf := NewConfig().WithCallback(&callback{t: t}).WithBufferSize(600)

		w := NewWriteAsyncer(&faultyWriter{}, conf)
		defer w.Stop()

		for i := 0; i < 10; i++ {
			_, err := w.Write(largeBytes)
			assert.Nil(t, err)
		}

		fmt.Println(">>> Error return by bufio Flush method")

		time.Sleep(time.Second)
	})
}
