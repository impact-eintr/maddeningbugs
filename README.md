# maddeningbugs
让我破大防的 bug 们

# 说明
这个仓库用来记录一些让我当时百思不得其解的bug（可能后续也没有头绪），希望以后的我能解开它们（希望不要一直破防到我转行，tnnd，为什么我这么菜）

## Bug No.1

来自 impact-eintr/esq 单机版本 在没有生产者生产消息的时候，消费者直接退出会导致 监听消费端的 goroutine 无法退出 目前使用了一个全局 map[string]chan bool 为每个消费者 goroutine注册一个 exit channel 并在检测到链接断开的时候找到对应的 channel 让然后 close(exitCh)，这样未免太过丑陋，增加不必要的全局变量不说也使得扩展性直线下降(这以后还得同步这个map?)

### 无法解决的问题

为什么同一块内存(至少看上去是 因为地址相同) [mem].(T).func() 调用后实际上无效

### 引申的知识点

优雅停止 goroutine 这算是代码设计的欠缺吗？

- 一种优秀的设计

``` go
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

```
