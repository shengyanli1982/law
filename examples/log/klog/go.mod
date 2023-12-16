module github.com/shengyanli1982/law/examples/http/server/syncwriter

go 1.19

require (
	github.com/shengyanli1982/law v0.0.0-00010101000000-000000000000
	k8s.io/klog/v2 v2.110.1
)

require github.com/go-logr/logr v1.3.0 // indirect

replace github.com/shengyanli1982/law => ../../../
