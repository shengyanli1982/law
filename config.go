package law

// DefaultBufferSize 是默认的缓冲区大小。
// DefaultBufferSize is the default buffer size.
const DefaultBufferSize = 2048

// Config 是一个定义了配置行为的结构体。
// Config is a struct that defines the behavior of a configuration.
type Config struct {
	logger   Logger   // logger 是用于记录日志的对象。
	buffsize int      // buffsize 是缓冲区的大小。
	callback Callback // callback 是回调函数的对象。
}

// NewConfig 返回一个拥有默认值的 Config 对象。
// NewConfig returns a Config object with default values.
func NewConfig() *Config {
	return &Config{
		buffsize: DefaultBufferSize,  // 设置缓冲区大小为默认值。
		callback: newEmptyCallback(), // 设置回调函数为一个空的回调。
		logger:   newLogger(),        // 设置日志记录器为新的日志记录器。
	}
}

// DefaultConfig 返回一个拥有默认值的 Config 对象，NewConfig 别名函数。
// DefaultConfig returns a Config object with default values, an alias function of NewConfig.
func DefaultConfig() *Config {
	return NewConfig() // 返回一个新的配置对象，其值为默认值。
}

// WithBufferSize 设置缓冲区大小。
// WithBufferSize sets the buffer size.
func (c *Config) WithBufferSize(size int) *Config {
	c.buffsize = size // 设置缓冲区大小为给定的值。
	return c          // 返回配置对象，以便进行链式调用。
}

// WithCallback 设置回调。
// WithCallback sets the callback.
func (c *Config) WithCallback(cb Callback) *Config {
	c.callback = cb // 设置回调函数为给定的值。
	return c        // 返回配置对象，以便进行链式调用。
}

// WithLogger 设置日志。
// WithLogger sets the logger.
func (c *Config) WithLogger(logger Logger) *Config {
	c.logger = logger // 设置日志记录器为给定的值。
	return c          // 返回配置对象，以便进行链式调用。
}

// isConfigValid 检查配置是否有效。
// isConfigValid checks if the configuration is valid.
func isConfigValid(conf *Config) *Config {
	if conf != nil { // 如果配置对象不为空，则检查其各个字段。
		if conf.buffsize <= 0 { // 如果缓冲区大小小于或等于0，则设置为默认值。
			conf.buffsize = DefaultBufferSize
		}
		if conf.callback == nil { // 如果回调函数为空，则设置为一个空的回调。
			conf.callback = newEmptyCallback()
		}
		if conf.logger == nil { // 如果日志记录器为空，则设置为新的日志记录器。
			conf.logger = newLogger()
		}
	} else { // 如果配置对象为空，则设置为默认配置。
		conf = DefaultConfig()
	}

	return conf // 返回配置对象。
}
