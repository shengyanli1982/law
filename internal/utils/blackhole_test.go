package utils

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// 编译期断言：*BlackHoleWriter 实现 io.Writer 接口。
var _ io.Writer = (*BlackHoleWriter)(nil)

// TestBlackHoleWriter_Write 验证 BlackHoleWriter 吞掉全部数据：
// 始终返回 len(p) 与 nil error，覆盖非空/空/nil 三种载荷。
func TestBlackHoleWriter_Write(t *testing.T) {
	w := &BlackHoleWriter{}

	tests := []struct {
		name string
		p    []byte
		want int
	}{
		{name: "non-empty payload", p: []byte("hello"), want: 5},
		{name: "empty payload", p: []byte{}, want: 0},
		{name: "nil payload", p: nil, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := w.Write(tt.p)
			require.NoError(t, err)
			require.Equal(t, tt.want, n)
		})
	}
}
