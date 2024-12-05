package law

import lf "github.com/shengyanli1982/law/internal/lockfree"

// DefaultBufferSize 默认缓冲区大小
// DefaultBufferSize is the default buffer size
const DefaultBufferSize = 2048

// Config 配置结构体
// Config is the configuration structure
type Config struct {
	buffSize int      // 缓冲区大小 / buffer size
	callback Callback // 回调函数 / callback function
	queue    Queue    // 队列实现 / queue implementation
}

// NewConfig 创建新的配置实例
// NewConfig creates a new configuration instance
func NewConfig() *Config {
	return &Config{
		buffSize: DefaultBufferSize,
		callback: newEmptyCallback(),
		queue:    lf.NewLockFreeQueue(),
	}
}

// DefaultConfig 返回默认配置
// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return NewConfig()
}

// WithBufferSize 设置缓冲区大小
// WithBufferSize sets the buffer size
func (c *Config) WithBufferSize(size int) *Config {
	c.buffSize = size
	return c
}

// WithCallback 设置回调函数
// WithCallback sets the callback function
func (c *Config) WithCallback(cb Callback) *Config {
	c.callback = cb
	return c
}

// WithQueue 设置队列实现
// WithQueue sets the queue implementation
func (c *Config) WithQueue(q Queue) *Config {
	c.queue = q
	return c
}

// isConfigValid 验证并修正配置
// isConfigValid validates and corrects the configuration
func isConfigValid(conf *Config) *Config {
	if conf != nil {
		if conf.buffSize <= 0 {
			conf.buffSize = DefaultBufferSize
		}
		if conf.callback == nil {
			conf.callback = newEmptyCallback()
		}
		if conf.queue == nil {
			conf.queue = lf.NewLockFreeQueue()
		}
	} else {
		conf = DefaultConfig()
	}
	return conf
}
