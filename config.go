package law

const DefaultBufferSize = 2048

type Config struct {
	logger Logger

	buffsize int

	callback Callback
}

func NewConfig() *Config {

	return &Config{

		buffsize: DefaultBufferSize,

		callback: newEmptyCallback(),

		logger: newLogger(),
	}

}

func DefaultConfig() *Config {

	return NewConfig()

}

func (c *Config) WithBufferSize(size int) *Config {

	c.buffsize = size

	return c

}

func (c *Config) WithCallback(cb Callback) *Config {

	c.callback = cb

	return c

}

func (c *Config) WithLogger(logger Logger) *Config {

	c.logger = logger

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

		if conf.logger == nil {

			conf.logger = newLogger()

		}

	} else {

		conf = DefaultConfig()

	}

	return conf

}
