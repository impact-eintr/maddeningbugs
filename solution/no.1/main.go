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
	for {
		select {
		case <-t.exitChan:
			log.Println("loop goroutine exit")
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

	time.Sleep(3 * time.Second)

	t.exit()

	log.Println("main goroutine exit")
}
