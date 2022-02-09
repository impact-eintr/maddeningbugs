package main

import (
	"log"
	"sync"
	"time"
)

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
}

func (t *test) loop() {
	defer log.Println("io goroutine exiting...")
	for {
		select {
		case <-t.exitChan:
			return
		default:
			log.Println("loop goroutine working")
			time.Sleep(time.Second)
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

	t.wg.Wrap(func() {
		defer log.Println("ticker goroutine exiting...")
		c := time.NewTimer(3 * time.Second)
		for {
			select {
			case <-t.exitChan:
				return
			case <-c.C:
				go t.exit()
			}
		}
	})

	for i := 2; i < 10; i++ {
		t.wg.Wrap(func() {
			defer log.Println("test goroutine exiting...")
			for {
				select {
				case <-t.exitChan:
					return
				}
			}
		})
	}

	time.Sleep(5 * time.Second)

	log.Println("main goroutine exit")

}
