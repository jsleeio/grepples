package main

import (
  "path"
  "bufio"
  "io"
	"compress/bzip2"
	"compress/gzip"
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
