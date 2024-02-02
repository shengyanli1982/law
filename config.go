package law

// DefaultBufferSize 是默认的缓冲区大小。
// DefaultBufferSize is the default buffer size.
const DefaultBufferSize = 2048

// Config 是一个定义了配置行为的结构体。
// Config is a struct that defines the behavior of a configuration.
type Config struct {
	logger   Logger   // 日志
	buffsize int      // 缓冲区大小
	callback Callback // 回调
}

// NewConfig 返回一个拥有默认值的 Config 对象
// NewConfig returns a Config object with default values.
func NewConfig() *Config {
	return &Config{
		buffsize: DefaultBufferSize,
		callback: newEmptyCallback(),
		logger:   newLogger(),
	}
}

// DefaultConfig 返回一个拥有默认值的 Config 对象，NewConfig 别名函数
// DefaultConfig returns a Config object with default values, an alias function of NewConfig.
func DefaultConfig() *Config {
	return NewConfig()
}

// WithBufferSize 设置缓冲区大小
// WithBufferSize sets the buffer size.
func (c *Config) WithBufferSize(size int) *Config {
	c.buffsize = size
	return c
}

// WithCallback 设置回调
// WithCallback sets the callback.
func (c *Config) WithCallback(cb Callback) *Config {
	c.callback = cb
	return c
}

// WithLogger 设置日志
// WithLogger sets the logger.
func (c *Config) WithLogger(logger Logger) *Config {
	c.logger = logger
	return c
}

// isConfigValid 检查配置是否有效
// isConfigValid checks if the configuration is valid.
func isConfigValid(conf *Config) *Config {
	if conf != nil {
		if conf.buffsize <= 0 {
			conf.buffsize = DefaultBufferSize
		}
		if conf.callback == nil {
			conf.callback = newEmptyCallback()
		}
		if conf.logger == nil {
			conf.logger = newLogger()
		}
	} else {
		conf = DefaultConfig()
	}

	return conf
}
