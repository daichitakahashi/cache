package cache

// Loader is
type Loader func(key string) (Cache, error)

// Cache is
type Cache interface {
	Get() interface{}
	Reload() error
	Updated() (bool, error)
	Release()
}
