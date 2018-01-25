package easypool

type Pool interface {
	Put(interface{}) error

	Get() (interface{}, error)

	Closable
}

type Closable interface {
	Close() error
}

//PoolConfig 连接池相关配置
type PoolConfig struct {
	//连接池中被初始化后，一直保留连接数
	Core int
	//连接池中拥有的最大的连接数
	Max int
	// 连接池的名字
	Name string
	// 连接池状态阈值
	Threadhold int
	// 需要连续满足Threadhold的次数。即连续满足Threadhold ThreadholdTimes次，机会切换状态
	ThreadholdTimes int
	//生成连接的方法
	Factory func() (Closable, error)
}
