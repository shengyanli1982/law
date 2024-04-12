package law

// DefaultBufferSize 是默认的缓冲区大小
// DefaultBufferSize is the default buffer size
const DefaultBufferSize = 2048

// Config 是配置结构体，包含了日志器、缓冲区大小和回调函数
// Config is a structure that contains a logger, buffer size, and a callback function
type Config struct {
	// buffSize 是缓冲区的大小
	// buffSize is the size of the buffer
	buffSize int

	// callback 是回调函数，用于处理特定事件
	// callback is a callback function for handling specific events
	callback Callback
}

// NewConfig 是一个构造函数，用于创建一个新的 Config 实例
// NewConfig is a constructor function for creating a new Config instance
func NewConfig() *Config {
	// 返回一个新的 Config 实例
	// Return a new Config instance
	return &Config{
		// 设置缓冲区大小为默认值
		// Set the buffer size to the default value
		buffSize: DefaultBufferSize,

		// 创建一个空的回调函数
		// Create an empty callback function
		callback: newEmptyCallback(),
	}
}

// DefaultConfig 是一个函数，返回一个新的默认配置实例
// DefaultConfig is a function that returns a new default configuration instance
func DefaultConfig() *Config {
	// 调用 NewConfig 函数创建一个新的配置实例
	// Call the NewConfig function to create a new configuration instance
	return NewConfig()
}

// WithBufferSize 是 Config 结构体的一个方法，用于设置缓冲区大小
// WithBufferSize is a method of the Config structure, used to set the buffer size
func (c *Config) WithBufferSize(size int) *Config {
	// 设置缓冲区大小
	// Set the buffer size
	c.buffSize = size

	// 返回配置实例，以便进行链式调用
	// Return the configuration instance for chaining
	return c
}

// WithCallback 是 Config 结构体的一个方法，用于设置回调函数
// WithCallback is a method of the Config structure, used to set the callback function
func (c *Config) WithCallback(cb Callback) *Config {
	// 设置回调函数
	// Set the callback function
	c.callback = cb

	// 返回配置实例，以便进行链式调用
	// Return the configuration instance for chaining
	return c
}

// isConfigValid 是一个函数，用于检查配置是否有效。如果配置无效，它将使用默认值进行修复。
// isConfigValid is a function to check if the configuration is valid. If the configuration is invalid, it will fix it with default values.
func isConfigValid(conf *Config) *Config {
	// 如果配置不为空
	// If the configuration is not null
	if conf != nil {

		// 如果缓冲区大小小于或等于0，将其设置为默认缓冲区大小
		// If the buffer size is less than or equal to 0, set it to the default buffer size
		if conf.buffSize <= 0 {
			conf.buffSize = DefaultBufferSize
		}

		// 如果回调函数为空，创建一个新的空回调函数
		// If the callback function is null, create a new empty callback function
		if conf.callback == nil {
			conf.callback = newEmptyCallback()
		}

	} else {
		// 如果配置为空，使用默认配置
		// If the configuration is null, use the default configuration
		conf = DefaultConfig()
	}

	// 返回修复后的配置
	// Return the fixed configuration
	return conf
}