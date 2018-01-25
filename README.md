# easypool

connection pool in Go, MIT license.
If any problem, welcome to contact me with email: superheizai@aliyun.com. And welcome to fork

## Intro
easypool is a connection pool in Go, which referenced the implementation of https://github.com/fatih/pool

## Core features:
 * with core connections, but initial size is 0, will be increased by Get,and will not increse to core if the frequence is not
high
 * has status change 
   1.init status is busy with true ,which means the pool will not destory the connection when put the connection back to pool  
   2.when the get opeartion is less than PoolConfig.Threadhold for PoolConfig.ThreadholdTimes, the status will change busy to false  
   3.when status is false, when the get opeartion is more than PoolConfig.Threadhold for PoolConfig.ThreadholdTimes, the status will change busy to true  
   4.when status is false, when get opeartion is is more than 2 times of PoolConfig.Threadhold, the busy will change to busy immediately  
  
## Usage

You can use like this

Firstly, you need a Connection which is a Closable interface implemenation, which need to implement Close() error .
```
type Closable interface {   
	Close() error   
  }    
 ```

Secondly, you need to config file

```
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

```
Thirdly, init the pool

```
	pool := InitEasyPool(config)
	```
	
Now, you can use like this

  ```
  	conn,err := pool.Get()
	if(err == TimeoutError){
		fmt.Println("timeout")
		return
	}
	pool.Put(conn)
    ```
 At last, please remember to close the pool
 
  ```
   pool.Close()
    ```
    
    
