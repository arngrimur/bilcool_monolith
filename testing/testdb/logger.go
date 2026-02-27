package testdb

import (
	"io"
	"testing"
)

// NewWriteLogger returns a writer that behaves like w except
// that it logs (using t.Logf).
func NewWriteLogger(t *testing.T, w io.Writer) io.Writer {
	return &writeLogger{t, w}
}

type writeLogger struct {
	t *testing.T
	w io.Writer
}

func (l *writeLogger) Write(p []byte) (n int, err error) {
	n, err = l.w.Write(p)
	if err != nil {
		l.t.Logf("%s: %v", p[0:n], err)
		return n, err
	}
	l.t.Logf("%s", p[0:n])
	return n, nil
}
