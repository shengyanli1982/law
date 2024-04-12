package law

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/shengyanli1982/law/internal/utils"
	"github.com/stretchr/testify/assert"
)

type callback struct {
	a0, a1, a2, a3 []string
	e3             []error
}

func (c *callback) OnPushQueue(b []byte) {
	c.a0 = append(c.a0, utils.BytesToString(b))
}

func (c *callback) OnPopQueue(b []byte, _ int64) {
	c.a1 = append(c.a1, utils.BytesToString(b))
}

func (c *callback) OnWriteSuccess(b []byte) {
	c.a2 = append(c.a2, utils.BytesToString(b))
}

func (c *callback) OnWriteFailure(b []byte, err error) {
	c.a3 = append(c.a3, utils.BytesToString(b))
	c.e3 = append(c.e3, err)
}

func TestWriteAsyncer_Standard(t *testing.T) {
	buff := bytes.NewBuffer(make([]byte, 0, 1024))

	w := NewWriteAsyncer(buff, nil)
	defer w.Stop()

	_, err := w.Write([]byte("hello"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("world"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("!!!"))
	assert.Nil(t, err)

	w.cleanQueueToWriter()
	w.bufferedWriter.Flush()

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
}

func TestWriteAsyncer_Callback(t *testing.T) {
	conf := NewConfig().WithCallback(&callback{})

	w := NewWriteAsyncer(os.Stdout, conf)
	defer w.Stop()

	_, err := w.Write([]byte("hello"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("world"))
	assert.Nil(t, err)
	_, err = w.Write([]byte("!!!"))
	assert.Nil(t, err)

	time.Sleep(time.Second)

	assert.Equal(t, []string{"hello", "world", "!!!"}, w.config.callback.(*callback).a0)
	assert.Equal(t, []string{"hello", "world", "!!!"}, w.config.callback.(*callback).a1)
	assert.Equal(t, []string{"hello", "world", "!!!"}, w.config.callback.(*callback).a2)
}
