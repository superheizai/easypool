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
	Coons       chan Closable // channel for connections
	Mux         sync.RWMutex
	Closed      bool
	closingChan chan interface{}
	current     int        // connections count currently in the pool
	busy        bool       // current status.true, put back can be putinto the pool directly. false, put will check if connection count is greater than core.yes, will discard the connection
	times       chan int64 // times channel, summary the time of every Get, to calculate Get() for every second
	marktime    int64      //current time,time.Unix()
	markCount   int        // Get() executed times at markTime
	markTimes   int        // continuous times over the threadhold or behind the threadhold
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
				// when busy is false and  markCount increases quickly, when markCount is greater than 2 times of threadhold, busy will change to true, will not wait the trigger condition of threadhold and threadholdTimes
				if (!easyPool.busy && easyPool.markCount > 2*easyPool.config.Threadhold) {
					easyPool.busy = true
				}
			} else {
				easyPool.marktime = cur
				lastMarkCount := easyPool.markCount
				easyPool.markCount = 1
				// a new second comes, marktime updated. need to check if fulfill the condition of threadhold and threadholdTimes to trigger busy change
				if (easyPool.busy) {
					if (lastMarkCount < easyPool.config.Threadhold) {
						easyPool.markTimes++
						if (easyPool.markTimes >= easyPool.config.ThreadholdTimes) {
							easyPool.busy = false
							easyPool.markTimes = 0;
						}
					} else {
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
