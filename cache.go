package cache

// Factory is
type Factory func(key string) (Cache, error)

// Cache is
type Cache interface {
	Get() interface{}
	Reload() error
	Updated() (bool, error)
	Replace(interface{}) error
	Release()
}
