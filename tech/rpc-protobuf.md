## golang基于protobuf的rpc使用
基本安装什么的在此不再赘述，不知道的看[这里](http://www.grpc.io/docs/quickstart/go.html)

### proto文件
cat helloworld.proto
```
syntax = "proto3";

option java_multiple_files = true;
option java_package = "io.grpc.examples.helloworld";
option java_outer_classname = "HelloWorldProto";

package helloworld;

// The greeting service definition.
service Greeter {
  // Sends a greeting
  rpc SayHello (HelloRequest) returns (HelloReply) {}
}

// The request message containing the user's name.
message HelloRequest {
  string name = 1;
}

// The response message containing the greetings
message HelloReply {
  string message = 1;
}
```
那几个生成java代码的定义就无视了，关键看service
定义了一个接口，然后定义了两条消息，分别是请求和回复的消息格式。

用一条命令就可以生成一个helloworld.pb.go文件
```
$ protoc -I helloworld/ helloworld/helloworld.proto --go_out=plugins=grpc:helloworld
```
来看看这个文件里有啥：

首先生成了请求和回复的结构体，提供一些方法可以调用
```
// The request message containing the user's name.
type HelloRequest struct {
    Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
}

func (m *HelloRequest) Reset()                    { *m = HelloRequest{} }
func (m *HelloRequest) String() string            { return proto.CompactTextString(m) }
func (*HelloRequest) ProtoMessage()               {}
func (*HelloRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

// The response message containing the greetings
type HelloReply struct {
    Message string `protobuf:"bytes,1,opt,name=message" json:"message,omitempty"`
}
```

重要的：
```
type GreeterClient interface {
    // Sends a greeting
    SayHello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error)
}

type greeterClient struct {
    cc *grpc.ClientConn
}

func NewGreeterClient(cc *grpc.ClientConn) GreeterClient {
    return &greeterClient{cc}
}

func (c *greeterClient) SayHello(ctx context.Context, in *HelloRequest, opts ...grpc.CallOption) (*HelloReply, error) {
    out := new(HelloReply)
    err := grpc.Invoke(ctx, "/helloworld.Greeter/SayHello", in, out, c.cc, opts...)
    if err != nil {
        return nil, err
    }
    return out, nil
}
```
可以看到客户端的代码都生成了，我们直接调用NewGreeterClient就能SayHello了，非常方便。

再看服务端：
```
type GreeterServer interface {
    // Sends a greeting
    SayHello(context.Context, *HelloRequest) (*HelloReply, error)
}

func RegisterGreeterServer(s *grpc.Server, srv GreeterServer) {
    s.RegisterService(&_Greeter_serviceDesc, srv)
}
```
所以我们只需要实现SayHello方法，然后将Server注册进去即可。

### 服务端代码
```
package main

import (
    "log"
    "net"

    "golang.org/x/net/context"
    "google.golang.org/grpc"
    pb "google.golang.org/grpc/examples/helloworld/helloworld"
    "google.golang.org/grpc/reflection"
)

const (
    port = ":50051"
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
    return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func main() {
    lis, err := net.Listen("tcp", port)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    s := grpc.NewServer()
    pb.RegisterGreeterServer(s, &server{})
    // Register reflection service on gRPC server.
    reflection.Register(s)
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
```
可以看到 s.Serve接受一个net.Lister参数，所以通过什么连接来启动rpc由我们来定。可以是tcp或者unix socket
我们自定义的server实现了SayHello具体干什么，然后注册到grpc服务里，最后与创建的监听链接关联起来

### 客户端代码
```
package main

import (
    "log"
    "os"

    "golang.org/x/net/context"
    "google.golang.org/grpc"
    pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
    address     = "localhost:50051"
    defaultName = "world"
)

func main() {
    // Set up a connection to the server.
    conn, err := grpc.Dial(address, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()
    c := pb.NewGreeterClient(conn)

    // Contact the server and print out its response.
    name := defaultName
    if len(os.Args) > 1 {
        name = os.Args[1]
    }
    r, err := c.SayHello(context.Background(), &pb.HelloRequest{Name: name})
    if err != nil {
        log.Fatalf("could not greet: %v", err)
    }
    log.Printf("Greeting: %s", r.Message)
}
```
客户端就简单了，首先与服务建立连接，然后New出client，直接调用生成好的方法即可。

## 总结
使用grpc在服务端进行进程间通信非常方便，生成代码的功能可以减少很多工作量，而且proto的解析速度和json解析不是一个数量级的. 在一些高性能的场景还是优先考虑proto吧。

在看docker源码时是需要知道这个的
