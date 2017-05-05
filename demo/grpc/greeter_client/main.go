package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/containerd/containerd/sys"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	//	address     = "localhost:50051"

	defaultName = "world"
)

func main() {
	socket := filepath.Join("/Users/fanux/", "shim.sock")
	l, err := sys.CreateUnixSocket(socket)
	if err != nil {
		fmt.Println("create unix socket failed", err)
		return
	}
	cmd := exec.Command("../greeter_server/greeter_server")
	//cmd.Dir = "/var/run"
	f, err := l.(*net.UnixListener).File()
	if err != nil {
		fmt.Println("interface transfer failed", err)
		return
	}
	defer f.Close()
	cmd.ExtraFiles = append(cmd.ExtraFiles, f)
	err = cmd.Start()
	if err != nil {
		fmt.Println("start server err", err)
	}

	dialOpts := []grpc.DialOption{grpc.WithInsecure(), grpc.WithTimeout(100 * time.Second)}
	dialOpts = append(dialOpts,
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", socket, timeout)
		}),
		grpc.WithBlock(),
		grpc.WithTimeout(2*time.Second),
	)

	// Set up a connection to the server.
	conn, err := grpc.Dial(fmt.Sprintf("unix://%s", socket), dialOpts...)
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
