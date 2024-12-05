package law

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var largeBytes = []byte("#Wqt2ZcQmVzk4KZJPmPamr3cBLNEf5dex2N2RkqhS3E7G6PWnzFEukskx5Z822mZd7")

type callback struct {
	t *testing.T
}

func (c *callback) OnWriteFailed(b []byte, err error) {
	if b != nil {
		fmt.Printf("## callback.OnWriteFailed(%s, %v)\n", b, err)
		assert.Equal(c.t, b, largeBytes, "Expected bytes")
	}
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

func TestWriteAsyncer_EdgeCases(t *testing.T) {
	t.Run("nil writer defaults to stdout", func(t *testing.T) {
		w := NewWriteAsyncer(nil, nil)
		assert.NotNil(t, w)
		w.Stop()
	})

	t.Run("nil content", func(t *testing.T) {
		buff := bytes.NewBuffer(make([]byte, 0))
		w := NewWriteAsyncer(buff, nil)
		defer w.Stop()

		_, err := w.Write(nil)
		assert.ErrorIs(t, err, ErrorWriteContentIsNil)
	})

	t.Run("empty content", func(t *testing.T) {
		buff := bytes.NewBuffer(make([]byte, 0))
		w := NewWriteAsyncer(buff, nil)
		defer w.Stop()

		n, err := w.Write([]byte{})
		assert.Nil(t, err)
		assert.Equal(t, 0, n)
	})

	t.Run("multiple stop calls", func(t *testing.T) {
		buff := bytes.NewBuffer(make([]byte, 0))
		w := NewWriteAsyncer(buff, nil)

		w.Stop()
		w.Stop()
	})
}

func TestWriteAsyncer_Concurrent(t *testing.T) {
	t.Run("concurrent writes", func(t *testing.T) {
		buff := bytes.NewBuffer(make([]byte, 0, 1024))
		w := NewWriteAsyncer(buff, nil)
		defer w.Stop()

		var wg sync.WaitGroup
		writers := 10
		iterations := 100

		wg.Add(writers)
		for i := 0; i < writers; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					content := []byte(fmt.Sprintf("w%d-%d", id, j))
					_, err := w.Write(content)
					assert.Nil(t, err)
				}
			}(i)
		}
		wg.Wait()
		time.Sleep(time.Second)
		assert.Greater(t, buff.Len(), 0)
	})
}

func BenchmarkWriteAsyncer(b *testing.B) {
	b.Run("small writes", func(b *testing.B) {
		buff := bytes.NewBuffer(make([]byte, 0, b.N*10))
		w := NewWriteAsyncer(buff, nil)
		defer w.Stop()

		data := []byte("small")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w.Write(data)
		}
	})

	b.Run("large writes", func(b *testing.B) {
		buff := bytes.NewBuffer(make([]byte, 0, b.N*len(largeBytes)))
		w := NewWriteAsyncer(buff, nil)
		defer w.Stop()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			w.Write(largeBytes)
		}
	})
}

func TestWriteAsyncer_BufferHandling(t *testing.T) {
	t.Run("buffer flush on size exceed", func(t *testing.T) {
		buff := bytes.NewBuffer(make([]byte, 0))
		conf := NewConfig().WithBufferSize(10)
		w := NewWriteAsyncer(buff, conf)
		defer w.Stop()

		_, err := w.Write([]byte("small"))
		assert.Nil(t, err)

		_, err = w.Write([]byte("this is a large content"))
		assert.Nil(t, err)

		time.Sleep(time.Second)
		assert.Contains(t, buff.String(), "small")
	})

	t.Run("buffer flush on idle timeout", func(t *testing.T) {
		buff := bytes.NewBuffer(make([]byte, 0))
		w := NewWriteAsyncer(buff, nil)
		defer w.Stop()

		_, err := w.Write([]byte("test"))
		assert.Nil(t, err)

		time.Sleep(defaultIdleTimeout + time.Second)
		assert.Equal(t, "test", buff.String())
	})
}
