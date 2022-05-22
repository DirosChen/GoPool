package gopool

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Gopool struct {
	PoolSize    int64
	queue       []*buffer
	lock        *sync.Mutex
	notify      chan interface{}
	rootContext *context.Context
	rootCancle  *context.CancelFunc
	current     int64
}

type buffer struct {
	runnable interface{}
	timeout  *context.Context
	ct       *context.Context
	cancle   *context.CancelFunc
}

func MakePool(size int64) *Gopool {
	pool := &Gopool{
		PoolSize: size,
		notify:   make(chan interface{}, size),
		queue:    make([]*buffer, 0),
		lock:     new(sync.Mutex),
	}
	root, cancle := context.WithCancel(context.Background())
	pool.rootContext = &root
	pool.rootCancle = &cancle
	pool.worker()
	return pool
}

func (pool *Gopool) Submit(runnable func()) *context.CancelFunc {
	atomic.AddInt64(&pool.current, 1)
	task, cancle := context.WithCancel(*pool.rootContext)
	if pool.current > pool.PoolSize {
		pool.queue = append(pool.queue, &buffer{runnable, nil, &task, &cancle})
	} else {
		pool.exec(runnable, task, nil, nil)
	}
	return &cancle
}

func (pool *Gopool) SubmitDelay(runnable func(), duration time.Duration) *context.CancelFunc {
	var cancale *context.CancelFunc
	time.AfterFunc(duration, func() {
		cancale = pool.Submit(runnable)
	})
	return cancale
}

func (pool *Gopool) SubmitTimeOut(runnable func(ctx context.Context), duration time.Duration) *context.CancelFunc {
	atomic.AddInt64(&pool.current, 1)
	task, cancle := context.WithCancel(*pool.rootContext)
	timeout, _ := context.WithTimeout(task, duration)
	if pool.current > pool.PoolSize {
		pool.queue = append(pool.queue, &buffer{runnable, &timeout, &task, &cancle})
	} else {
		pool.exec(runnable, task, timeout, nil)
	}
	return &cancle
}

func (pool *Gopool) Shutdown() {
	(*pool.rootCancle)()
}

func (pool *Gopool) exec(runnable interface{}, ctx context.Context, timeout context.Context, duration *time.Duration) {
	go func() {
		select {
		case <-ctx.Done():
			log.Println("done")
			break
		default:
			switch run := runnable.(type) {
			case func():
				run()
				break
			case func(ctx2 context.Context):
				run(timeout)
				break

			}
			atomic.AddInt64(&pool.current, -1)
			pool.notify <- 0
			break
		}
	}()
}

func (pool *Gopool) worker() {
	go func() {
		for {
			select {
			case <-pool.notify:
				result := pool.syncRead()
				if result != nil {
					pool.syncRemove()
					if result != nil {
						go func() {
							select {
							case <-(*result.ct).Done():
								break
							default:
								switch run := result.runnable.(type) {
								case func():
									run()
									break
								case func(ctx2 context.Context):
									run(*result.timeout)
									break
								}
								pool.notify <- 0
								break
							}
						}()
					}
				}
			}
		}
	}()
}

func (pool *Gopool) syncRead() *buffer {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	if len(pool.queue) > 0 {
		ta := pool.queue[0]
		return ta
	}
	return nil
}

func (pool *Gopool) syncRemove() {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	length := len(pool.queue)
	if length == 1 {
		pool.queue = make([]*buffer, 0)
	} else if length == 0 {
	} else {
		pool.queue = pool.queue[1:]
	}
	atomic.AddInt64(&pool.current, -1)
}
