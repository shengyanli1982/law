package law

import (
	"time"

	lf "github.com/shengyanli1982/law/internal/lockfree"
)

// DefaultBufferSize 默认缓冲区大小
// DefaultBufferSize is the default buffer size
const DefaultBufferSize = 2048

// 默认心跳间隔和空闲超时时间
// Default heartbeat interval and idle timeout duration
const (
	DefaultHeartbeatInterval = 500 * time.Millisecond
	DefaultIdleTimeout       = 5 * time.Second
)

// Config 配置结构体
// Config is the configuration structure
type Config struct {
	buffSize          int           // 缓冲区大小 / buffer size
	callback          Callback      // 回调函数 / callback function
	queue             Queue         // 队列实现 / queue implementation
	heartbeatInterval time.Duration // 心跳间隔 / heartbeat interval
	idleTimeout       time.Duration // 闲置超时 / idle timeout
}

// NewConfig 创建新的配置实例
// NewConfig creates a new configuration instance
func NewConfig() *Config {
	return &Config{
		buffSize:          DefaultBufferSize,
		callback:          newEmptyCallback(),
		queue:             lf.NewLockFreeQueue(),
		heartbeatInterval: DefaultHeartbeatInterval,
		idleTimeout:       DefaultIdleTimeout,
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

// WithHeartbeatInterval 设置心跳间隔
// WithHeartbeatInterval sets the heartbeat interval
func (c *Config) WithHeartbeatInterval(interval time.Duration) *Config {
	c.heartbeatInterval = interval
	return c
}

// WithIdleTimeout 设置闲置超时时间
// WithIdleTimeout sets the idle timeout duration
func (c *Config) WithIdleTimeout(timeout time.Duration) *Config {
	c.idleTimeout = timeout
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
		if conf.heartbeatInterval <= 0 {
			conf.heartbeatInterval = DefaultHeartbeatInterval
		}
		if conf.idleTimeout <= 0 {
			conf.idleTimeout = DefaultIdleTimeout
		}
	} else {
		conf = DefaultConfig()
	}
	return conf
}
