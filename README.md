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


- 一个更加综合的案例 配合任务池

``` go
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

```

## Bug No.2

这个 Bug 其实是我之前解决的,算是一种设计思路 场景是 使用 gin 进行路由处理的时候需要给处理方法传递一些数据

``` go
r := gin.Default()

r.GET("/test", func(c *gin.Context){
	// TODO
})
```


但这样无法直接传递(不使用全局变量等方法的话)

### 解决

``` go
r := gin.Default

s := "test"

r.GET("/test", func(val string) gin.Handlefunc {
	return func(c *gin.Context) {
		fmt.Println(val)
	}
}(s))


```


### Notice No.3

- 在for-select中，break只会影响到select，不会影响到for
- 单独在select中是不能使用continue，会编译错误，只能用在for-select中。continue的语义就类似for中的语义，select后的代码不会被执行到。

### Bug No.4
- for select 中想要关闭管道来停止当前循环 应该再开一个goroutine？

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

```

### Bug No.5
- http 客户端不应该使用默认配置
- 客户端的 resp.Body 一定要关掉! 否则会引发 `accept4: too many open files; retrying in 5m`

``` go
func Ping(url string) (bool) {
    // create a new instance of http client struct, with a timeout of 2sec
    client := http.Client{ Timeout: time.Second * 2 }

    // simple GET request on given URL
    res, err := client.Get(url)
    if err != nil {
        // if unable to GET given URL, then ping must fail
        return false
    }

    // always close the response-body, even if content is not required
    defer res.Body.Close()

    // is the page status okay?
    return res.StatusCode == http.StatusOK
}
```

### TIPs NO.6
替换多个文件中某个字符串

-i 表示inplace edit，就地修改文件
-r 表示搜索子目录
-l 表示输出匹配的文件名

``` sh
sed -i "s/entry/Entry/g" `grep entry -rl .`
```


### Note No.7

切片将第一个元素放到最后

``` go
var client []*Client

	c := clients[0]
	if len(clients) > 1 {
		// 已处理过的消息客户端重新放在最后
		clients = append(clients[1:], c)
	}

```
