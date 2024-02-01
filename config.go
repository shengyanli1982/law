package law

const DefaultBufferSize = 2048 // 默认缓冲区大小 (default buffer size)

type Config struct {
	buffsize int      // 缓冲区大小 (Buffer size)
	callback Callback // 回调函数 (Callback function)
}

// 创建一个配置对象 (Create a new Config object)
func NewConfig() *Config {
	return &Config{buffsize: DefaultBufferSize, callback: newEmptyCallback()}
}

func DefaultConfig() *Config {
	return NewConfig()
}

// 设置缓冲区大小 (Set the buffer size)
func (c *Config) WithBufferSize(size int) *Config {
	c.buffsize = size
	return c
}

// 设置回调函数 (Set the callback function)
func (c *Config) WithCallback(cb Callback) *Config {
	c.callback = cb
	return c
}

func isConfigValid(conf *Config) *Config {
	if conf != nil {
		if conf.buffsize <= 0 {
			conf.buffsize = DefaultBufferSize
		}
		if conf.callback == nil {
			conf.callback = newEmptyCallback()
		}
	} else {
		conf = DefaultConfig()
	}

	return conf
}
