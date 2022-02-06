package consolefile

import (
	"bytes"
	"context"

	"github.com/containerd/console"
)

type tflogFunc func(ctx context.Context, msg string, args ...interface{})

// WithPrefix wraps File so that all content gets written with tflog
func WithPrefix(ctx context.Context, f console.File, fun tflogFunc) console.File {

	return &fileWithPrefix{
		ctx: ctx,
		f:   f,
		fun: fun,
	}
}

type fileWithPrefix struct {
	ctx context.Context
	f   console.File
	fun tflogFunc
	buf bytes.Buffer // reuse buffer to save allocations
}

func (f *fileWithPrefix) Close() error {
	if f.buf.Len() != 0 {
		f.fun(f.ctx, f.buf.String())
		f.buf.Reset() // clear the buffer
	}
	return f.f.Close()
}

func (f *fileWithPrefix) Fd() uintptr {
	return f.f.Fd()
}

func (f *fileWithPrefix) Name() string {
	return f.f.Name()
}

func (f *fileWithPrefix) Read(p []byte) (n int, err error) {
	return f.f.Read(p)
}

func (f *fileWithPrefix) Write(p []byte) (n int, err error) {

	for _, b := range p {
		if b == '\n' {
			f.fun(f.ctx, f.buf.String())
			f.buf.Reset() // clear the buffer
			continue
		}

		f.buf.WriteByte(b)
	}

	// return original length to satisfy io.Writer interface
	return len(p), nil
}
