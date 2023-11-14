package main

import "grpc/proto"

//go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
//go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
//protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/hello.proto
import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"net"
)

type server struct {
	proto.UnimplementedHelloServer
}

func (s *server) Say(ctx context.Context, req *proto.SayRequest) (*proto.SayResponse, error) {
	fmt.Println("request:", req.Name)
	return &proto.SayResponse{Message: "Hello " + req.Name}, nil
}

func main() {
	listen, err := net.Listen("tcp", ":8001")
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
		return
	}
	s := grpc.NewServer()
	proto.RegisterHelloServer(s, &server{})
	//reflection.Register(s)
	defer func() {
		s.Stop()
		listen.Close()
	}()

	fmt.Println("Serving 8001...")
	err = s.Serve(listen)
	if err != nil {
		fmt.Printf("failed to serve: %v", err)
		return
	}
}
