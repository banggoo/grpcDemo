package main

import (
    "context"
    "log"

    "google.golang.org/grpc"

    pb "simple/helloworld"
)

const (
    address     = "localhost:50051"
    defaultName = "world"
)

func main() {
    conn, err := grpc.Dial(address, grpc.WithInsecure())
    if err != nil {
        panic(err)
    }

    defer conn.Close()

    c := pb.NewGreeterClient(conn)
    r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: defaultName})
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("r.Message = %+v\n", r.Message)
}
