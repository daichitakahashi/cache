package cache

import (
	"io"
	"os"
	"time"
)

type (
	// Unmarshaler :
	Unmarshaler func(io.Reader) (interface{}, error)

	// Marshaler :
	Marshaler func(interface{}, io.Writer) error
)

// FromFile :
func FromFile(unmarshal Unmarshaler, marshal Marshaler) Factory {
	return func(key string) (Cache, error) {
		file := &file{
			filename:  key,
			unmarshal: unmarshal,
			marshal:   marshal,
		}
		/*
			err := file.Reload()
			if err != nil {
				return nil, err
			}
		*/
		return file, nil
	}
}

type file struct {
	filename  string
	object    interface{}
	modTime   time.Time
	unmarshal Unmarshaler
	marshal   Marshaler
}

func (f *file) Get() interface{} {
	return f.object
}

func (f *file) Reload() error {
	fd, err := os.Open(f.filename)
	if err != nil {
		return err
	}
	defer fd.Close()

	f.object, err = f.unmarshal(fd)
	if err != nil {
		return err
	}

	fi, err := fd.Stat()
	if err != nil {
		return err
	}
	f.modTime = fi.ModTime()

	return nil
}

func (f *file) Updated() (bool, error) {
	fi, err := os.Stat(f.filename)
	if err != nil {
		return false, &OpError{"check update", err}
	}
	modTime := fi.ModTime()
	return !modTime.Equal(f.modTime), nil

}

func (f *file) Reset(v interface{}) error {
	if f.marshal != nil {
		fd, err := os.Create(f.filename)
		if err != nil {
			return err
		}
		defer fd.Close()
		err = f.marshal(f.object, fd)
		if err != nil {
			return err
		}
	}
	f.object = v
	return nil
}

func (f *file) Release() {
	f.object = nil
}

var _ Cache = (*file)(nil)
