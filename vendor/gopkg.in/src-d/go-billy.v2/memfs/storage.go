package memfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/src-d/go-billy.v2"
)

type storage struct {
	files    map[string]*file
	children map[string]map[string]*file
}

func newStorage() *storage {
	return &storage{
		files:    make(map[string]*file, 0),
		children: make(map[string]map[string]*file, 0),
	}
}

func (s *storage) Has(path string) bool {
	path = filepath.Clean(path)

	_, ok := s.files[path]
	return ok
}

func (s *storage) New(path string, mode os.FileMode, flag int) (*file, error) {
	path = filepath.Clean(path)
	if s.Has(path) {
		if !s.MustGet(path).mode.IsDir() {
			return nil, fmt.Errorf("file already exists %q", path)
		}

		return nil, nil
	}

	f := &file{
		BaseFile: billy.BaseFile{
			BaseFilename: filepath.Base(path),
		},
		content: &content{},
		mode:    mode,
		flag:    flag,
	}

	s.files[path] = f
	s.createParent(path, mode, f)
	return f, nil
}

func (s *storage) createParent(path string, mode os.FileMode, f *file) error {
	base := filepath.Dir(path)
	base = filepath.Clean(base)
	if f.Filename() == string(separator) {
		return nil
	}

	if _, err := s.New(base, mode.Perm()|os.ModeDir, 0); err != nil {
		return err
	}

	if _, ok := s.children[base]; !ok {
		s.children[base] = make(map[string]*file, 0)
	}

	s.children[base][f.Filename()] = f
	return nil
}

func (s *storage) Children(path string) []*file {
	path = filepath.Clean(path)

	l := make([]*file, 0)
	for _, f := range s.children[path] {
		l = append(l, f)
	}

	return l
}

func (s *storage) MustGet(path string) *file {
	f, ok := s.Get(path)
	if !ok {
		panic(fmt.Errorf("couldn't find %q", path))
	}

	return f
}

func (s *storage) Get(path string) (*file, bool) {
	path = filepath.Clean(path)
	if !s.Has(path) {
		return nil, false
	}

	file, ok := s.files[path]
	return file, ok
}

func (s *storage) Rename(from, to string) error {
	from = filepath.Clean(from)
	to = filepath.Clean(to)

	if !s.Has(from) {
		return os.ErrNotExist
	}

	move := [][2]string{{from, to}}

	for pathFrom := range s.files {
		if pathFrom == from || !filepath.HasPrefix(pathFrom, from) {
			continue
		}

		rel, _ := filepath.Rel(from, pathFrom)
		pathTo := filepath.Join(to, rel)

		move = append(move, [2]string{pathFrom, pathTo})
	}

	for _, ops := range move {
		from := ops[0]
		to := ops[1]

		if err := s.move(from, to); err != nil {
			return err
		}
	}

	return nil
}

func (s *storage) move(from, to string) error {
	s.files[to] = s.files[from]
	s.files[to].BaseFilename = filepath.Base(to)
	s.children[to] = s.children[from]

	delete(s.children, from)
	delete(s.files, from)

	return s.createParent(to, 0644, s.files[to])
}

func (s *storage) Remove(path string) error {
	path = filepath.Clean(path)

	f, has := s.Get(path)
	if !has {
		return os.ErrNotExist
	}

	if f.mode.IsDir() && len(s.children[path]) != 0 {
		return fmt.Errorf("dir: %s contains files", path)
	}

	base, file := filepath.Split(path)
	base = filepath.Clean(base)

	delete(s.children[base], file)
	delete(s.files, path)
	return nil
}

type content struct {
	bytes []byte
}

func (c *content) WriteAt(p []byte, off int64) (int, error) {
	prev := len(c.bytes)

	diff := int(off) - prev
	if diff > 0 {
		c.bytes = append(c.bytes, make([]byte, diff)...)
	}

	c.bytes = append(c.bytes[:off], p...)
	if len(c.bytes) < prev {
		c.bytes = c.bytes[:prev]
	}

	return len(p), nil
}

func (c *content) ReadAt(b []byte, off int64) (int, error) {
	size := int64(len(c.bytes))
	if off >= size {
		return 0, io.EOF
	}

	l := int64(len(b))
	if off+l > size {
		l = size - off
	}

	n := copy(b, c.bytes[off:off+l])
	return n, nil
}
