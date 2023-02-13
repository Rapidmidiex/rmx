package rmx

type Broker[K, V any] interface {
	Load(key K) (value V, ok bool)
	LoadOrStore(key K, value V) (actual V, loaded bool)
	Store(key K, value V)
	Delete(key K)
	LoadAndDelete(key K) (value V, loaded bool)
}
