module github.com/shengyanli1982/law/examples/log/zap

go 1.19

replace github.com/shengyanli1982/law => ../../../

require (
	github.com/shengyanli1982/law v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.26.0
)

require go.uber.org/multierr v1.10.0 // indirect
