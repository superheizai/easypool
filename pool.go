package easypool

type Pool interface {
	Put(interface{}) error

	Get() (interface{}, error)

	Closable
}

type Closable interface {
	Close() error
}

type PoolConfig struct {
	// core pool size
	Core int
	//max pool size
	Max int
	// Name the pool
	Name string
	// Threadhold for change status
	Threadhold int
	// times the Threadhold should be fulfilled to trigger status change
	ThreadholdTimes int
	// Factory method to get connection
	Factory func() (Closable, error)
}

