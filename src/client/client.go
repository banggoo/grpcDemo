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
	Duration    = flag.Int64("d", 0, "duration")                     //运行时间，0代表不限制。单位为秒。
	Timeout     = flag.Int64("e", 5, "grpc timeout")                 //grpc连接超时时间，默认5秒。
	Threads     = flag.Int64("t", 1, "threads")                      //线程数，默认为１
	QPS         = flag.Int64("q", 10, "QPS")                         //发送频率，默认为10
	QueueSize   = flag.Int64("s", 1024, "channel queue Size")        //channel缓冲区长度，默认为1024
	Address     = flag.String("a", "localhost:50051", "address")     //访问地址
	DefaultName = flag.String("defaultName", "world", "defaultNmae") //grpc name

	TaskPoolWaitGroup sync.WaitGroup
	StatisticsChan	  chan int
	Conn              *grpc.ClientConn = nil
	Jobs              int64            = 0
	TimeAve           int              = 0
)

type Task struct {
	job      <-chan int //接收任务的队列
	poolSize int64      //同时运行的协程池
}

func (t *Task) run() {
	var i int64
	start := time.Now()
	fmt.Println("Start Time: ", start)

	for i = 0; i < t.poolSize; i++ {
		TaskPoolWaitGroup.Add(1)

		go func() {
			for j := range t.job {
				_ = j

				jobStart := time.Now()
				justDoIt()
				//fmt.Println(time.Since(jobStart))
				StatisticsChan <- int(time.Since(jobStart)/1.0e+3)

				atomic.AddInt64(&Jobs, 1)

				//如果探测次数达到设定阈值，则退出协程
				if *Times != 0 && Jobs >= *Times {
					break
				}

				//如果运行时间达到阈值，则退出协程
				if *Duration != 0 && int64(time.Since(start)/1.0e+9) > *Duration {
					break
				}
			}

			TaskPoolWaitGroup.Done()
		}()
	}

	Fini()

	fmt.Println("End Time: ", time.Now())
	fmt.Println(strings.Repeat("#", 100))

	fmt.Println("Run Times:           ", Jobs)
	//fmt.Println("Run Time Duration:   ", int(time.Since(start)/1.0e+9))
	fmt.Println("Run Time Duration:   ", time.Since(start))
	fmt.Println("Run Time Average(ms):", TimeAve/1.0e+3)
	//fmt.Println("Run Time Average(us):", TimeAve)
}

func Fini() {
	TaskPoolWaitGroup.Wait()
	close(StatisticsChan)

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*Timeout)*time.Second)
	defer cancel()

	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
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

	c = pb.NewGreeterClient(Conn)
	invoke(c, *DefaultName)
}

func testNoKeepalive() {
	var c pb.GreeterClient

	conn, err := grpc.Dial(*Address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return
	}
	defer conn.Close()

	c = pb.NewGreeterClient(conn)
	invoke(c, *DefaultName)
}

func jobProducer(job chan int) {
	var i int64

	for {
		start := time.Now()

		//每一次释放固定个数的任务到channel中，如果时间未到1秒，则sleep以达到1s，以此来控制QPS。如果时间超过1秒，则说明任务处理不及时，应该调大线程数。
		for i = 0; i < *QPS; i++ {
			job <- 1
		}

		t := time.Since(start)/1.0e+6

		if t > 1000 {
			//如果打印此行，则需要调整参数以避免出现此种情况。
			fmt.Println(strings.Repeat("!", 100))
		} else {
			time.Sleep((1000 - t) * time.Millisecond)
		}
	}
}

func Statistics(tlist chan int) {
	for j := range tlist {
		TimeAve = (TimeAve + j)/2.0
	}
}

func main() {
	flag.Parse()
	jobChan := make(chan int, *QueueSize)
	StatisticsChan = make(chan int, *QueueSize)

	go jobProducer(jobChan)
	go Statistics(StatisticsChan)

	task := Task{jobChan, *Threads}
	task.run()
}
