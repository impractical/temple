package temple_test

import (
	"io"
	"io/fs"
	"time"
)

type staticFS map[string]string

// Open opens the named file.
// When Open returns an error, it should be of type *PathError
// with the Op field set to "open", the Path field set to name,
// and the Err field describing the problem.
//
// Open should reject attempts to open names that do not satisfy
// ValidPath(name), returning a *PathError with Err set to
// ErrInvalid or ErrNotExist.
func (s staticFS) Open(name string) (fs.File, error) {
	val, ok := s[name]
	if !ok {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}
	return &staticFile{
		name:     name,
		contents: []byte(val),
	}, nil
}

type staticFile struct {
	name     string
	contents []byte
	offset   int
}

func (s *staticFile) Stat() (fs.FileInfo, error) {
	return s, nil
}

func (s *staticFile) Read(buf []byte) (int, error) {
	if s.offset >= len(s.contents) {
		return 0, io.EOF
	}
	if s.offset < 0 {
		return 0, &fs.PathError{
			Op:   "read",
			Path: s.name,
			Err:  fs.ErrInvalid,
		}
	}
	n := copy(buf, s.contents[s.offset:])
	s.offset += n
	return n, nil
}

func (*staticFile) Close() error {
	return nil
}

func (s *staticFile) Name() string {
	return s.name
}

func (s *staticFile) Size() int64 {
	return int64(len(s.contents))
}

func (*staticFile) Mode() fs.FileMode {
	return 0400
}

func (*staticFile) ModTime() time.Time {
	return time.Now()
}

func (*staticFile) IsDir() bool {
	return false
}

func (*staticFile) Sys() any {
	return nil
}
