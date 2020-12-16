package utils

import "io"

func CloseSilently(closer io.Closer) {
	_ = closer.Close()
}
