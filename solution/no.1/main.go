package main

import (
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const MaxPoolSize = 8

type WaitGroupWrapper struct {
	sync.WaitGroup
}

func (w *WaitGroupWrapper) Wrap(cb func()) {
	w.Add(1)
	go func() {
		cb()
		w.Done()
	}()
}

type test struct {
	exitChan chan struct{}
	wg       WaitGroupWrapper
	poolSize int32
}

func (t *test) loop() {
	defer log.Println("loop goroutine exit")

	selectNum := 20
	workCh := make(chan string, selectNum)   // 用于分发topic给worker处理
	responseCh := make(chan bool, selectNum) // 用于worker处理完任务后响应
	closeCh := make(chan int)                // 用于通知worker退出
	t.resizePool(MaxPoolSize, workCh, closeCh, responseCh)

	scanTicker := time.NewTicker(1 * time.Second)
	freshTicker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-t.exitChan:
			goto exit
		case <-freshTicker.C:
			t.resizePool((MaxPoolSize + rand.Intn(10)), workCh, closeCh, responseCh)
		case <-scanTicker.C:
		default:
			log.Println("loop goroutine working")
			time.Sleep(time.Second)
		}

		if rand.Intn(100)%2 == 0 {
			workCh <- "Impact-EINTR"
		} else {
			workCh <- "233333"
		}

		if ok := <-responseCh; ok {
			log.Println("正确的结果")
		}

	}

exit:
	close(closeCh)
	scanTicker.Stop()
	freshTicker.Stop()
}

func (t *test) delayLoop(workCh chan string, closeCh chan int, responseCh chan bool) {
	for {
		select {
		case <-closeCh:
			atomic.AddInt32(&t.poolSize, -1)
			return
		case s := <-workCh:
			log.Printf("delay loop goroutine working, consume %s\n", s)
			if s == "Impact-EINTR" {
				responseCh <- true
			} else {
				responseCh <- false
			}
		}
	}
}

func (t *test) resizePool(num int, workCh chan string, closeCh chan int, responseCh chan bool) {
	workerNum := int32(float64(num) * 0.25)
	if workerNum < 1 {
		workerNum = 1
	}

	if workerNum > t.poolSize {
		t.wg.Wrap(func() {
			t.delayLoop(workCh, closeCh, responseCh)
		})
		t.poolSize++
	}

	if workerNum < t.poolSize {
		for i := t.poolSize - workerNum; i > 0; i-- {
			closeCh <- 1
		}
	}
}

func (t *test) exit() {
	close(t.exitChan)
	t.wg.Wait()
}

func main() {
	t := &test{exitChan: make(chan struct{})}

	t.wg.Wrap(t.loop)

	time.Sleep(5 * time.Second)

	t.exit()

	log.Println("main goroutine exit")
}
