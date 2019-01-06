package main

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"io"
  "os"
	"path"
	"strconv"
	"golang.org/x/crypto/ssh/terminal"
)

// TransparentExpandingReader creates a Reader that transparently decompresses based
// on filename. Supports gzip and bzip2; falls back to assuming uncompressed
func TransparentExpandingReader(key string, source io.ReadCloser) (io.Reader, error) {
	ext := path.Ext(key)
	var reader io.Reader
	var err error
	switch {
	case ext == ".gz":
		reader, err = gzip.NewReader(source)
		if err != nil {
			return nil, err
		}
	case ext == ".bz2":
		reader = bzip2.NewReader(source)
	default:
		reader = bufio.NewReader(source)
	}
	return reader, nil
}

// leftN returns the N leftmost characters of the string when N>0, and the
// whole string otherwise
func leftN(s string, n int) string {
	if n == 0 {
		return s
	}
	l := len(s)
	if n > l {
		return s
	}
	return s[:n]
}

// ttyWidth tries to determine the width in characters of the terminal attached
// to stdin, falling back to the integer value of the COLUMNS environment
// variable, and if that fails, returns 79
func ttyWidth() int {
	// try ioctl (TIOCGWINSZ) approach first
	if width, _, err := terminal.GetSize(0); err == nil {
		return width
	}
	// if that fails, try $COLUMNS environment variable
	c := os.Getenv("COLUMNS")
	if width, err := strconv.ParseInt(c, 10, 16); err == nil && width > 0 {
		return int(width)
	}
	// help. please help
	return 79
}
