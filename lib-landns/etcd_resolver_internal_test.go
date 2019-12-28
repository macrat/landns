package landns

import (
	"testing"
)

func TestCompileGlob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Glob string
		Str  string
		Exp  bool
	}{
		{`ab*def`, `abcdef`, true},
		{`ab.*f`, `abcdef`, false},
		{`ab.*f`, `ab.cdef`, true},
		{`[0-9]*(a+b*)`, `[0-9]---(a+b=====)`, true},
		{`^abc$`, `^abc$`, true},
		{`\.*`, `\.abc`, true},
		{`cd`, `abcdef`, false},
		{`*cd*`, `abcdef`, true},
		{`*.example.com.`, `abc.example.com.`, true},
		{`*.example.com.`, `.example.com.`, true},
		{`*.example.com.`, `abc*example*com*`, false},
	}

	for _, tt := range tests {
		glob, err := compileGlob(tt.Glob)
		if err != nil {
			t.Errorf("failed to compile glob: %#v: %s", tt.Glob, err)
			continue
		}

		if glob(tt.Str) != tt.Exp {
			t.Errorf("failed to test glob: %#v <- %#v: expected %v but got %v", tt.Glob, tt.Str, tt.Exp, glob(tt.Str))
		}
	}
}
