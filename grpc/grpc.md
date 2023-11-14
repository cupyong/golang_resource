1.grpc的框架应用编写（编写一个用户管理系统）
实现目录结构
user-management-demo/
|-- api/
|   |-- user/
|   |   |-- v1/
|   |       |-- user.proto
|
|-- cmd/
|   |-- client/
|   |   |-- main.go
|   |-- server/
|   |   |-- main.go
|
|-- internal/
|   |-- service/
|   |   |-- user.go
|   |   |-- user_test.go
|   |-- middleware/
|   |   |-- logging.go
|   |   |-- authentication.go
|
|-- Makefile
|-- go.mod
|-- go.sum


需要定义grpc服务的.proto出入参文件
```syntax = "proto3";

package api.user.v1;

service UserService {
rpc CreateUser (CreateUserRequest) returns (CreateUserResponse);
rpc GetUser (GetUserRequest) returns (GetUserResponse);
rpc DeleteUser (DeleteUserRequest) returns (DeleteUserResponse);
}

message CreateUserRequest {
string name = 1;
int32 age = 2;
}

message CreateUserResponse {
string id = 1;
}

message GetUserRequest {
string id = 1;
}

message GetUserResponse {
string id = 1;
string name = 2;
int32 age = 3;
}

message DeleteUserRequest {
string id = 1;
}

message DeleteUserResponse {
bool success = 1;
}
```

使用protoco工具，生成客户端和服务端代码（需要安装 protoc 和 protoc-gen-go 插件）
```$ protoc -I api/ --go_out=plugins=grpc:api/ api/user/v1/user.proto```
实现用户服务
```package service

import (
"context"
"errors"

	"github.com/google/uuid"

	pb "user-management-demo/api/user/v1"
)

type User struct {
ID   string
Name string
Age  int32
}

var users = make(map[string]*User)

type UserService struct {
pb.UnimplementedUserServiceServer
}

func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
user := &User{
ID:   uuid.New().String(),
Name: req.Name,
Age:  req.Age,
}
users[user.ID] = user

	return &pb.CreateUserResponse{Id: user.ID}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
user, ok := users[req.Id]
if !ok {
return nil, errors.New("user not found")
}

	return &pb.GetUserResponse{Id: user.ID, Name: user.Name, Age: user.Age}, nil
}

func (s *UserService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
_, ok := users[req.Id]
if !ok {
return nil, errors.New("user not found")
}

	delete(users, req.Id)
	return &pb.DeleteUserResponse{Success: true}, nil
}
```

实现grpc服务器
```package main

import (
"log"
"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"user-management-demo/internal/middleware"
	"user-management-demo/internal/service"

	pb "user-management-demo/api/user/v1"
)

const (
port = ":50051"
)

func main() {
lis, err := net.Listen("tcp", port)
if err != nil {
log.Fatalf("failed to listen: %v", err)
}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.LoggingInterceptor,
			middleware.AuthenticationInterceptor,
		),
	)

	userService := &service.UserService{}
	pb.RegisterUserServiceServer(grpcServer, userService)
	reflection.Register(grpcServer)

	log.Printf("gRPC server is running on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

```
实现grpc客户端
```
package main

import (
"context"
"log"
"time"

	"google.golang.org/grpc"

	pb "user-management-demo/api/user/v1"
)

const (
address = "localhost:50051"
)

func main() {
conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
if err != nil {
log.Fatalf("did not connect: %v", err)
}
defer conn.Close()

	client := pb.NewUserServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Create a new user
	createResp, err := client.CreateUser(ctx, &pb.CreateUserRequest{Name: "Alice", Age: 30})
	if err != nil {
		log.Fatalf("could not create user: %v", err)
	}
	userId := createResp.Id
	log.Printf("User created: %s", userId)

	// Get the user
	getResp, err := client.GetUser(ctx, &pb.GetUserRequest{Id: userId})
	if err != nil {
		log.Fatalf("could not get user: %v", err)
	}
	log.Printf("User: ID=%s, Name=%s, Age=%d", getResp.Id, getResp.Name, getResp.Age)

	// Delete the user
	_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: userId})
	if err != nil {
		log.Fatalf("could not delete user: %v", err)
	}
	log.Printf("User deleted: %s", userId)
}

```
实现日志和认证中间件
// internal/middleware/logging.go
```package middleware

import (
"context"
"log"
"time"

	"google.golang.org/grpc"
)

func LoggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
start := time.Now()
resp, err = handler(ctx, req)
log.Printf("Request - Method: %s, Duration: %s, Error: %v", info.FullMethod, time.Since(start), err)
return resp, err
}

// internal/middleware/authentication.go
package middleware

import (
"context"
"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func AuthenticationInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
if md, ok := metadata.FromIncomingContext(ctx); ok {
if token, ok := md["authorization"]; ok && len(token) > 0 {
if validateToken(token[0]) {
return handler(ctx, req)
}
}
}

	return nil, errors.New("unauthorized")
}

func validateToken(token string) bool {
// Replace this with your actual token validation logic.
return token == "your-valid-token"
}

```
测试grpc服务
```
package service

import (
"context"
"testing"

	pb "user-management-demo/api/user/v1"
)

func TestUserService(t *testing.T) {
service := UserService{}

	// Test CreateUser
	createResp, err := service.CreateUser(context.Background(), &pb.CreateUserRequest{Name: "Alice", Age: 30})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	userId := createResp.Id

	// Test GetUser
	getResp, err := service.GetUser(context.Background(), &pb.GetUserRequest{Id: userId})
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if getResp.Name != "Alice" || getResp.Age != 30 {
		t.Errorf("GetUser returned incorrect data: got %v, want Name=Alice, Age=30", getResp)
	}

	// Test DeleteUser
	deleteResp, err := service.DeleteUser(context.Background(), &pb.DeleteUserRequest{Id: userId})
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	if !deleteResp.Success {
		t.Errorf("DeleteUser failed: got %v, want Success=true", deleteResp)
	}
}

```
运行测试
```$ go test ./internal/service```
编译运行
```$ go build -o server ./cmd/server```
```$ ./server```
2.使用Go-kit 来封装grpc
添加 go-kit 相关依赖：
$ go get github.com/go-kit/kit/endpoint
$ go get github.com/go-kit/kit/log
$ go get github.com/go-kit/kit/transport/grpc
创建一个用户服务接口（internal/service/user.go）：
```package service

type User struct {
ID   string
Name string
Age  int32
}

type UserService interface {
CreateUser(name string, age int32) (id string, err error)
GetUser(id string) (user User, err error)
DeleteUser(id string) (success bool, err error)
}
```
创建用户服务的具体实现（internal/service/user_impl.go）：
```package service

import (
"errors"

	"github.com/google/uuid"
)

var users = make(map[string]*User)

type UserServiceImpl struct{}

func (s *UserServiceImpl) CreateUser(name string, age int32) (string, error) {
user := &User{
ID:   uuid.New().String(),
Name: name,
Age:  age,
}
users[user.ID] = user
return user.ID, nil
}

func (s *UserServiceImpl) GetUser(id string) (User, error) {
user, ok := users[id]
if !ok {
return User{}, errors.New("user not found")
}

	return *user, nil
}

func (s *UserServiceImpl) DeleteUser(id string) (bool, error) {
_, ok := users[id]
if !ok {
return false, errors.New("user not found")
}

	delete(users, id)
	return true, nil
}
```
创建用户服务的 go-kit Endpoints（internal/endpoint/user_endpoint.go）
```
package endpoint

import (
"context"

	"github.com/go-kit/kit/endpoint"

	"user-management-demo/internal/service"
)

type UserEndpoints struct {
CreateUserEndpoint  endpoint.Endpoint
GetUserEndpoint     endpoint.Endpoint
DeleteUserEndpoint  endpoint.Endpoint
}

func MakeUserEndpoints(s service.UserService) UserEndpoints {
return UserEndpoints{
CreateUserEndpoint:  makeCreateUserEndpoint(s),
GetUserEndpoint:     makeGetUserEndpoint(s),
DeleteUserEndpoint:  makeDeleteUserEndpoint(s),
}
}

func makeCreateUserEndpoint(s service.UserService) endpoint.Endpoint {
return func(ctx context.Context, request interface{}) (interface{}, error) {
req := request.(CreateUserRequest)
id, err := s.CreateUser(req.Name, req.Age)
return CreateUserResponse{Id: id}, err
}
}

func makeGetUserEndpoint(s service.UserService) endpoint.Endpoint {
return func(ctx context.Context, request interface{}) (interface{}, error) {
req := request.(GetUserRequest)
user, err := s.GetUser(req.Id)
return GetUserResponse{
Id:   user.ID,
Name: user.Name,
Age:  user.Age,
}, err
}
}

func makeDeleteUserEndpoint(s service.UserService) endpoint.Endpoint {
return func(ctx context.Context, request interface{}) (interface{}, error) {
req := request.(DeleteUserRequest)
success, err := s.DeleteUser(req.Id)
return DeleteUserResponse{Success: success}, err
}
}

type CreateUserRequest struct {
Name string
Age  int32
}

type CreateUserResponse struct {
Id string
}

type GetUserRequest struct {
Id string
}

type GetUserResponse struct {
Id   string
Name string
Age  int32
}

type DeleteUserRequest struct {
Id string
}

type DeleteUserResponse struct {
Success bool
}
```
创建 gRPC 传输层适配器（internal/transport/grpc/user_grpc.go）：
```
package grpc

import (
"context"

	"github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"

	"user-management-demo/api/user/v1"
	"user-management-demo/internal/endpoint"
)

type grpcServer struct {
createUser  grpc.Handler
getUser     grpc.Handler
deleteUser  grpc.Handler
}

func (s *grpcServer) CreateUser(ctx context.Context, req *v1.CreateUserRequest) (*v1.CreateUserResponse, error) {
_, resp, err := s.createUser.ServeGRPC(ctx, req)
if err != nil {
return nil, err
}
return resp.(*v1.CreateUserResponse), nil
}

func (s *grpcServer) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.GetUserResponse, error) {
_, resp, err := s.getUser.ServeGRPC(ctx, req)
if err != nil {
return nil, err
}
return resp.(*v1.GetUserResponse), nil
}

func (s *grpcServer) DeleteUser(ctx context.Context, req *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error) {
_, resp, err := s.deleteUser.ServeGRPC(ctx, req)
if err != nil {
return nil, err
}
return resp.(*v1.DeleteUserResponse), nil
}

func NewUserGRPCServer(endpoints endpoint.UserEndpoints, serverOptions []grpc.ServerOption) v1.UserServiceServer {
return &grpcServer{
createUser: grpc.NewServer(
endpoints.CreateUserEndpoint,
decodeCreateUserRequest,
encodeCreateUserResponse,
serverOptions...,
),
getUser: grpc.NewServer(
endpoints.GetUserEndpoint,
decodeGetUserRequest,
encodeGetUserResponse,
serverOptions...,
),
deleteUser: grpc.NewServer(
endpoints.DeleteUserEndpoint,
decodeDeleteUserRequest,
encodeDeleteUserResponse,
serverOptions...,
),
}
}

func decodeCreateUserRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
req := grpcReq.(*v1.CreateUserRequest)
return endpoint.CreateUserRequest{Name: req.Name, Age: req.Age}, nil
}

func encodeCreateUserResponse(_ context.Context, response interface{}) (interface{}, error) {
resp := response.(endpoint.CreateUserResponse)
return &v1.CreateUserResponse{Id: resp.Id}, nil
}

func decodeGetUserRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
req := grpcReq.(*v1.GetUserRequest)
return endpoint.GetUserRequest{Id: req.Id}, nil
}

func encodeGetUserResponse(_ context.Context, response interface{}) (interface{}, error) {
resp := response.(endpoint.GetUserResponse)
return &v1.GetUserResponse{Id: resp.Id, Name: resp.Name, Age: resp.Age}, nil
}

func decodeDeleteUserRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
req := grpcReq.(*v1.DeleteUserRequest)
return endpoint.DeleteUserRequest{
req := grpcReq.(*v1.DeleteUserRequest)
return endpoint.DeleteUserRequest{Id: req.Id}, nil
}
}
func encodeDeleteUserResponse(_ context.Context, response interface{}) (interface{}, error) {
resp := response.(endpoint.DeleteUserResponse)
return &v1.DeleteUserResponse{Success: resp.Success}, nil
}
```
实现 gRPC 服务器（cmd/server/main.go）：
```
package main

import (
"log"
"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"user-management-demo/api/user/v1"
	"user-management-demo/internal/endpoint"
	"user-management-demo/internal/service"
	"user-management-demo/internal/transport/grpc"
)

const (
port = ":50051"
)

func main() {
lis, err := net.Listen("tcp", port)
if err != nil {
log.Fatalf("failed to listen: %v", err)
}

	userService := &service.UserServiceImpl{}
	userEndpoints := endpoint.MakeUserEndpoints(userService)

	grpcServer := grpc.NewServer()
	v1.RegisterUserServiceServer(grpcServer, grpc_transport.NewUserGRPCServer(userEndpoints, nil))
	reflection.Register(grpcServer)

	log.Printf("gRPC server is running on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
```
3.go-kit解耦原理
gokit的作用
在 Go-kit 中，Endpoints（端点）是一种抽象，它用于将服务的业务逻辑与传输层（如 HTTP、gRPC 等）解耦。简而言之，Endpoints 是一种用于表示服务方法的通用接口，它定义了服务方法的输入和输出，并处理底层通信细节。

在 Go-kit 中，Endpoint 是一个函数，它的签名如下：

type Endpoint func(ctx context.Context, request interface{}) (response interface{}, err error)
Endpoint 接受一个 context 和一个请求对象，并返回一个响应对象和一个错误。这使得 Endpoint 可以用于表示任何服务方法，无论其参数和返回值的结构如何。

当你使用 Go-kit 编写微服务时，你会为服务的每个方法创建一个 Endpoint。然后，你可以将这些 Endpoints 与不同的传输层（如 HTTP、gRPC 等）集成，让它们处理实际的请求和响应的序列化和反序列化。这使得你可以更轻松地在不同传输协议之间切换，同时保持业务逻辑的一致性。

此外，Go-kit 还提供了一组中间件，可以用于在 Endpoints 上实现通用的功能，如熔断、限流、监控、日志记录等。这使得你可以为微服务添加这些功能，而无需对底层服务实现进行修改。

为了解决什么
相当于，把grpc的相关接口，接入到端点，进行端点注册反射即可，就不用去手动创建grpc的服务拨号请求之类的繁琐，所有的都归类到框架特定的地方

Go-kit 的目标之一就是帮助开发者将业务逻辑与底层通信协议（如 gRPC、HTTP 等）解耦。通过使用 Endpoints，你可以将服务的业务逻辑与实际的传输层实现分离，这使得在不同协议之间切换更加容易。

在使用 Go-kit 和 gRPC 时，你会创建 Endpoints，它们会将服务方法与 gRPC 传输层适配器连接。这样，你就可以专注于实现业务逻辑，而无需关心 gRPC 的具体实现细节。同时，通过使用 Endpoints，你还可以轻松地在其他传输协议（如 HTTP）之间切换，只需实现相应的适配器即可。

此外，Go-kit 还提供了一组中间件，可以用于在 Endpoints 上实现通用的功能，如熔断、限流、监控、日志记录等。这使得你可以为微服务添加这些功能，而无需对底层服务实现进行修改。这种架构方式帮助你更好地组织代码，使得代码更易于维护和扩展。

