package cache

import (
	"io"
	"os"
	"sync/atomic"
	"time"
)

type (
	// Unmarshaler :
	Unmarshaler func(io.Reader) (interface{}, error)

	// Marshaler :
	Marshaler func(interface{}, io.Writer) error
)

// FromFile :
// updateLatency is used for setting interval of os.Stat.
func FromFile(unmarshal Unmarshaler, marshal Marshaler, updateLatency time.Duration) Factory {

	return func(key string) (Cache, error) {
		var update int32 = 1
		file := &file{
			filename:  key,
			unmarshal: unmarshal,
			marshal:   marshal,
		}
		if updateLatency > 0 {
			file.update = &update
			file.latency = updateLatency
		}
		return file, nil
	}
}

type file struct {
	filename  string
	object    interface{}
	modTime   time.Time
	unmarshal Unmarshaler
	marshal   Marshaler
	update    *int32
	latency   time.Duration
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
	// レイテンシの設定がない、あるいはチェックすべきタイミングであるとき
	if f.update == nil || f.shouldCheckUpdate() {
		fi, err := os.Stat(f.filename)
		if err != nil {
			return false, &OpError{"check update", err}
		}
		modTime := fi.ModTime()
		return !modTime.Equal(f.modTime), nil
	}
	return false, nil
}

func (f *file) wait() {
	time.Sleep(f.latency)
	atomic.StoreInt32(f.update, 1)
}

func (f *file) shouldCheckUpdate() bool {
	shouldUpdate := atomic.SwapInt32(f.update, 0)
	if shouldUpdate == 1 {
		go f.wait()
		return true
	}
	return false
}

func (f *file) Replace(v interface{}) error {
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
