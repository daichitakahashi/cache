package cache

import (
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
)

type file struct {
	filename string
	object   interface{}
	modTime  time.Time
	loadFunc func(r io.Reader) (interface{}, error)
}

// NewFile is
func NewFile(filename string, loadFunc func(r io.Reader) (interface{}, error)) (Cache, error) { // ==============================================================--
	return &file{
		filename: filename,
		loadFunc: loadFunc,
	}, nil
}

func (f *file) Get() interface{} {
	return f.object
}

func (f *file) Reload() error {
	fd, err := os.Open(f.filename)
	if err != nil {
		return errors.Wrap(err, "Reload")
	}
	defer fd.Close()

	f.object, err = f.loadFunc(fd)
	if err != nil {
		return errors.Wrap(err, "Reload")
	}

	fi, err := fd.Stat()
	if err != nil {
		return errors.Wrap(err, "Reload: unknown error")
	}
	f.modTime = fi.ModTime()

	return nil
}

func (f *file) Updated() (bool, error) {
	fi, err := os.Stat(f.filename)
	if err != nil {
		return false, errors.Errorf("Updated: file lost(filename=%s)", f.filename)
	}
	modTime := fi.ModTime()
	return !modTime.Equal(f.modTime), nil

}

func (f *file) Release() {
	f.object = nil
}

var _ Cache = (*file)(nil)
