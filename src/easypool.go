package easypool

import (
	"sync"
	"time"
	"log"
	"errors"
)

var (
	TimeoutError               = errors.New("get connection timeout")
	OperationAtClosedPoolError = errors.New("pool has been shutdown")
)

type EasyPool struct {
	config      PoolConfig
	Coons       chan Closable // 连接池
	Mux         sync.RWMutex  //锁
	Closed      bool
	closingChan chan interface{}
	current     int        // 当前分配出的连接数量，包括conns里面的和在使用未归还的
	busy        bool       // current status, 记录当前操作频繁与否;true时候，put操作直接入队；false时候，先看是否小于core，再入队，大于core，直接回收了
	times       chan int64 // times channel,每个get请求会把自己的请求时间放到channel里面，统计每个时间点的请求信息
	marktime    int64      //此时在统计的标记时间点,秒级时间，time.unix()返回
	markCount   int        // markTime时间点上，被mark的次数
	markTimes   int        // 现在已经因为超过或者低于阈值，而累积的次数。当超过durationtimes，会触发busy值的改变
}

func InitEasyPool(config PoolConfig) *EasyPool {
	easyPool := new(EasyPool)
	easyPool.config = config
	easyPool.busy = true
	easyPool.Coons = make(chan Closable, config.Max)
	easyPool.Closed = false
	easyPool.closingChan = make(chan interface{}, 1)
	easyPool.busy = true
	easyPool.times = make(chan int64, 1000)
	easyPool.marktime = time.Now().Unix()
	easyPool.markCount = 0
	easyPool.markTimes = 0
	easyPool.current = 0
	go easyPool.switchStatus()
	return easyPool
}

func (easyPool *EasyPool) switchStatus() {

	for ; ; {
		if (!easyPool.Closed) {
			cur := <-easyPool.times
			if easyPool.marktime == cur {
				easyPool.markCount++
				//在空闲时候，如果操作次数大于2倍阈值的时候，会直接转为忙状态，不会等待这一秒的统计周期结束
				if (!easyPool.busy && easyPool.markCount > 2*easyPool.config.Threadhold) {
					easyPool.busy = true
				}
			} else {
				easyPool.marktime = cur
				lastMarkCount := easyPool.markCount
				easyPool.markCount = 1
				//新的时间窗到来，根据之前是忙还是空，来判断状态
				if (easyPool.busy) {
					if (lastMarkCount < easyPool.config.Threadhold) {
						easyPool.markTimes++
						if (easyPool.markTimes >= easyPool.config.ThreadholdTimes) {
							easyPool.busy = false
							easyPool.markTimes = 0;
						}
					} else {
						//新的时间窗到来，但是其阈值没有超过设定阈值
						easyPool.markTimes = 0
					}
				} else {

					if (lastMarkCount >= easyPool.config.Threadhold) {
						easyPool.markTimes++
						if (easyPool.markTimes >= easyPool.config.ThreadholdTimes) {
							easyPool.busy = true
							easyPool.markTimes = 0;
						}
					} else {
						//新的时间窗到来，但是其阈值没有超过设定阈值
						easyPool.markTimes = 0
					}
				}

			}

		} else {
			log.Fatal("switchStatus check exit for pool has been closed")
			return
		}
	}

}

func (easyPool *EasyPool) pinPoint(cur int64) {
	easyPool.times <- cur
}

func (easyPool *EasyPool) Put(conn Closable) error {

	if (easyPool.Closed) {
		log.Fatal("easyPool has been Closed")
		return OperationAtClosedPoolError

	}

	easyPool.Mux.Lock()
	defer easyPool.Mux.Unlock()

	if (easyPool.busy) {
		easyPool.Coons <- conn
	} else {
		if len(easyPool.Coons) < easyPool.config.Core {
			easyPool.Coons <- conn
		} else {
			easyPool.current--
			conn.Close()
		}
	}

	return nil
}

func (easyPool *EasyPool) Get() (Closable, error) {
	if (easyPool.Closed) {
		log.Fatal("easyPool has been Closed")
		return nil, OperationAtClosedPoolError

	}

	go easyPool.pinPoint(time.Now().Unix())
	select {
	case conn := <-easyPool.Coons:
		return conn, nil
	default:

		if (easyPool.current < easyPool.config.Max) {
			easyPool.Mux.RLock()
			defer easyPool.Mux.RUnlock()
			conn, err := easyPool.config.Factory()
			if (err != nil) {
				log.Fatal(err)
				return nil, err
			}
			easyPool.current++
			return conn, nil
		} else {
			for {
				select {
				case <-time.After(5 * time.Second):
					//log.Fatal("can't get connection %s after 5S", easyPool.config.Name)
					return nil, TimeoutError
				case conn := <-easyPool.Coons:
					return conn, nil

				case <-easyPool.closingChan:
					easyPool.Mux.RLock()
					defer easyPool.Mux.RUnlock()
					easyPool.current--
					if easyPool.current >= 0 {
						easyPool.closingChan <- ""
					}
					return nil, OperationAtClosedPoolError
				}
			}
		}
	}

}

func (easyPool *EasyPool) Close() {
	if (easyPool.Closed) {
		return
	}
	easyPool.Mux.Lock()
	easyPool.closingChan <- ""
	close(easyPool.Coons)
	defer easyPool.Mux.Unlock()
	easyPool.Closed = true
	for closable := range easyPool.Coons {
		closable.Close()
	}

}

func (easyPool *EasyPool) Size() (int, error) {
	if (easyPool.Closed) {
		log.Fatal("easyPool has been Closed")
		return 0, OperationAtClosedPoolError

	}
	return easyPool.current, nil

}
