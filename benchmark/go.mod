module github.com/shengyanli1982/law/benchmark

go 1.19

replace github.com/shengyanli1982/law => ../

require (
	github.com/shengyanli1982/law v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.27.0
)

require go.uber.org/multierr v1.11.0 // indirect
