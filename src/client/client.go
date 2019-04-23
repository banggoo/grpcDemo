package main

import (
	"fmt"
	"log"
	"sync"
	"time"
	"flag"

	pb "simple/helloworld"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	address     = "localhost:50051"
	defaultName = "world"
)
var (
	Sync = flag.Int("sync", 1, "sync")
	KeepAlive = flag.Int("keepalive", 1, "keepalive")
	Times = flag.Int("times", 10000, "times")
)

func invoke(c pb.GreeterClient, name string) {
	r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	_ = r
}

func invoke2(name string) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return
	}
	defer conn.Close()

	var c pb.GreeterClient = pb.NewGreeterClient(conn)

	r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	_ = r
}

func syncTest(c pb.GreeterClient, name string) {
	fmt.Println("syncTest")
	i := *Times
	t := time.Now().UnixNano()
	for ; i>0; i-- {
		invoke(c, name)
	}
	fmt.Println("took", (time.Now().UnixNano() - t) / 1000000, "ms")
}


func asyncTest(c [20]pb.GreeterClient, name string) {
	fmt.Println("asyncTest")
	var wg sync.WaitGroup
	wg.Add(*Times)

	i := *Times
	t := time.Now().UnixNano()
	for ; i>0; i-- {
		go func() {invoke(c[i % 20], name);wg.Done()}()
	}
	wg.Wait()
	fmt.Println("took", (time.Now().UnixNano() - t) / 1000000, "ms")
}


func testKeepalive() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	var c [20]pb.GreeterClient


	// Contact the server and print out its response.
	name := defaultName

	//warm up
	i := 0
	for ; i < 20; i++ {
		c[i] = pb.NewGreeterClient(conn)
		invoke(c[i], name)
	}

	if *Sync == 1 {
		syncTest(c[0], name)
	} else {
		asyncTest(c, name)
	}
}

func asyncTest2(name string) {
	fmt.Println("asyncTest")
	var wg sync.WaitGroup
	wg.Add(*Times)

	i := *Times
	t := time.Now().UnixNano()
	for ; i>0; i-- {
		go func() {invoke2(name);wg.Done()}()
	}
	wg.Wait()
	fmt.Println("took", (time.Now().UnixNano() - t) / 1000000, "ms")
}
func testNoKeepalive() {
	name := defaultName
	asyncTest2(name)
}

func main() {
	flag.Parse()
	if *KeepAlive == 1 {
		testKeepalive()
	}else if *KeepAlive == 0 {
		testNoKeepalive()
	}
}
