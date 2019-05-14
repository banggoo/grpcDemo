package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	pb "simple/helloworld"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	KeepAlive   = flag.Int64("k", 1, "keepalive")                    //长短连接开关，默认开启长连接
	Times       = flag.Int64("n", 0, "times")                        //运行次数，0代表不限制
	Duration    = flag.Int64("d", 0, "duration")                     //运行时间，0代表不限制
	Threads     = flag.Int64("t", 1, "threads")                      //线程数，默认为１
	QPS         = flag.Int64("q", 10, "QPS")                         //发送频率，默认为10
	QueueSize   = flag.Int64("s", 1024, "channel queue Size")        //channel缓冲区长度，默认为1024
	Address     = flag.String("a", "localhost:50051", "address")     //访问地址
	DefaultName = flag.String("defaultName", "world", "defaultNmae") //grpc name

	TaskPoolWaitGroup sync.WaitGroup
	Conn              *grpc.ClientConn = nil
	Jobs              int64            = 0
)

type Task struct {
	job      <-chan int //接收任务的队列
	poolSize int64      //同时运行的协程池
}

func (t *Task) run() {
	var i int64
	start := time.Now()

	for i = 0; i < t.poolSize; i++ {
		TaskPoolWaitGroup.Add(1)

		go func() {
			for j := range t.job {
				_ = j
				justDoIt()
				atomic.AddInt64(&Jobs, 1)

				//如果探测次数达到设定阈值，则退出协程
				if *Times != 0 && Jobs >= *Times {
					break
				}

				//如果运行时间达到阈值，则退出协程
				if *Duration != 0 && int64(time.Since(start)) > *Duration {
					break
				}
			}

			TaskPoolWaitGroup.Done()
		}()
	}

	Fini()
}

func Fini() {
	TaskPoolWaitGroup.Wait()

	if Conn != nil {
		Conn.Close()
	}
}

func justDoIt() {
	if *KeepAlive == 1 {
		testKeepalive()
	} else if *KeepAlive == 0 {
		testNoKeepalive()
	}
}

func invoke(c pb.GreeterClient, name string) {
	r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	_ = r
}

func testKeepalive() {
	var c pb.GreeterClient

	if Conn == nil {
		conn, err := grpc.Dial(*Address, grpc.WithInsecure())
		if err != nil {
			log.Fatalf("did not connect: %v", err)
			return
		}
		Conn = conn
	}
	name := *DefaultName

	c = pb.NewGreeterClient(Conn)
	invoke(c, name)
}

func testNoKeepalive() {
	var c pb.GreeterClient

	conn, err := grpc.Dial(*Address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return
	}
	defer conn.Close()

	name := *DefaultName

	c = pb.NewGreeterClient(Conn)
	invoke(c, name)
}

func jobProducer(job chan int) {
	var i int64

	for {
		start := time.Now()

		//每一次释放固定个数的任务到channel中，如果时间未到1秒，则sleep以达到1s，以此来控制QPS。如果时间超过1秒，则说明任务处理不及时，应该调大线程数。
		for i = 0; i < *QPS; i++ {
			job <- 1
		}

		t := time.Since(start)
		fmt.Printf("Time duration %d, for %d qps.", t, *QPS)

		if t > 1000 {
			//如果打印此行，则需要调整参数以避免出现此种情况。
			fmt.Println(strings.Repeat("!", 100))
		} else {
			time.Sleep((1000 - t*time.Millisecond) * time.Millisecond)
		}
	}
}

func main() {
	flag.Parse()
	jobChan := make(chan int, *QueueSize)

	go jobProducer(jobChan)

	task := Task{jobChan, *Threads}
	task.run()
}
