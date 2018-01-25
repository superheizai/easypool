package easypool

import (
	"testing"
	"fmt"
	"time"
)

type mockConn struct {
}

func (conn *mockConn) Close() error {
	fmt.Println("conn is closed")
	return nil
}

//func TestInitEasyPool(t *testing.T) {
//	config := PoolConfig{}
//	config.Factory = func() (Closable, error) {
//		return new(mockConn), nil
//	}
//	config.Name = "test"
//	config.Max = 20
//	config.Core = 10
//	config.ThreadholdTimes = 2
//	config.Threadhold = 10
//
//	pool := InitEasyPool(config)
//
//	for i := 0; i < 1000; i++ {
//		go test(pool)
//	}
//	fmt.Println(pool.Size())
//
//	fmt.Println("=====close")
//	pool.Close()
//
//}
//
//func test(pool *EasyPool) {
//	begin := time.Now().UnixNano()
//	conn, _ := pool.Get()
//	pool.Put(conn)
//	end := time.Now().UnixNano()
//	fmt.Println("a tst message ", end-begin, pool.busy, pool.current)
//
//}

func TestEasyPool_Get(t *testing.T) {
	config := PoolConfig{}
	config.Factory = func() (Closable, error) {
		return new(mockConn), nil
	}
	config.Name = "test"
	config.Max = 20
	config.Core = 10
	config.ThreadholdTimes = 2
	config.Threadhold = 10

	pool := InitEasyPool(config)

	for i := 0; i < 1000; i++ {
		go sleep(pool)
	}
	fmt.Println(pool.Size())

	time.Sleep(time.Minute)
	fmt.Println("=====close")
	pool.Close()
}

func sleep(pool *EasyPool) {
	begin := time.Now().Unix()
	conn,err := pool.Get()
	if(err == TimeoutError){
		fmt.Println("timeout")
		return
	}
	end := time.Now().Unix()
	fmt.Println("a tst message ", end-begin, pool.busy, pool.current)
	time.Sleep(10 * time.Millisecond)
	pool.Put(conn)


}
