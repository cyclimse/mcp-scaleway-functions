package std

import (
	"context"
	"io"
)

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (int, error) { return rf(p) }

// Shamelessly stolen from this Gist: https://gist.github.com/dillonstreator/3e9162e6e0d0929a6543a64f4564b604
func Copy(ctx context.Context, dst io.Writer, src io.Reader) error {
	_, err := io.Copy(dst, readerFunc(func(p []byte) (int, error) {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
			return src.Read(p)
		}
	}))

	//nolint:wrapcheck // fine to preserve the original error.
	return err
}
