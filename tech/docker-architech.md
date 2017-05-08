## Docker架构分析
```
[root@docker-build-86-050 ~]# ls /usr/bin |grep docker
docker
docker-compose
docker-containerd
docker-containerd-ctr
docker-containerd-shim
dockerd
docker-proxy
docker-runc
```
大家一定很困惑 dockerd, containerd, ctr,shim, runc,等这几个进程的关系到底是啥

初窥得出的结论是：

* docker是cli没啥可说的
* dockerd是docker engine守护进程，dockerd启动时会启动containerd子进程。
* dockerd与containerd通过rpc进行通信（待验证，可能是通过ctr）
* ctr是containerd的cli
* containerd通过shim操作runc，runc真正控制容器生命周期
* 启动一个容器就会启动一个shim进程，shim与容器中进程是父子或孙等关系(待验证)
* shim直接调用runc的包函数,shim与containerd之前通过rpc通信

以上结论不一定正确，有待验证

```
[root@docker-build-86-050 ~]# ps -aux|grep docker
root      3925  0.0  0.1 2936996 74020 ?       Ssl  3月06  68:14 /usr/bin/dockerd --storage-driver=aufs -H 0.0.0.0:2375 --label ip=10.1.86.50 -H unix:///var/run/docker.sock --insecure-registry 192.168.86.106 --insecure-registry 10.1.86.51 --insecure-registry dev.reg.iflytek.com
root      3939  0.0  0.0 1881796 27096 ?       Ssl  3月06   9:10 docker-containerd -l unix:///var/run/docker/libcontainerd/docker-containerd.sock --shim docker-containerd-shim --metrics-interval=0 --start-timeout 2m --state-dir /var/run/docker/libcontainerd/containerd --runtime docker-runc
root     21238  0.0  0.0 487664  6212 ?        Sl   4月20   0:00 docker-containerd-shim 48119c50a0ca8a53967364f75fb709017cc272ae248b78062e0dafaa22108d21 /var/run/docker/libcontainerd/48119c50a0ca8a53967364f75fb709017cc272ae248b78062e0dafaa22108d21 docker-runc
```

## dockerd 与 containerd 之间的基情
首先dockerd的main函数相信你能找到`cmd/dockerd/docker.go`

其它的先略过,直接进start看一看：
```
err = daemonCli.start(opts)
```
这函数里我们先去关注两件事：

1. 创建了多个Hosts，这是给client去连接的，dockerd启动时用-H参数指定，可以是多个，如指定一个tcp 指定一个unix sock( -H unix:///var/run/docker.sock)
2. 创建了containerd子进程

这个New很重要
```
containerdRemote, err := libcontainerd.New(cli.getLibcontainerdRoot(), cli.getPlatformRemoteOptions()...)
```
进去看看：
```
...
    err := r.runContainerdDaemon(); 
...
    conn, err := grpc.Dial(r.rpcAddr, dialOpts...)
    if err != nil {
        return nil, fmt.Errorf("error connecting to containerd: %v", err)
    }

    r.rpcConn = conn
    r.apiClient = containerd.NewAPIClient(conn)
...
```
启动了一个containerd进程，并与之建立连接。通过protobuf进行rpc通信， grpc相关介绍看[这里](http://lameleg.com/tech/rpc-protobuf.html)

具体如何创建containerd进程的可以进入runContainerDaemon里细看
```
    cmd := exec.Command(containerdBinary, args...)
    // redirect containerd logs to docker logs
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.SysProcAttr = setSysProcAttr(true)
    cmd.Env = nil
    // clear the NOTIFY_SOCKET from the env when starting containerd
    for _, e := range os.Environ() {
        if !strings.HasPrefix(e, "NOTIFY_SOCKET") {
            cmd.Env = append(cmd.Env, e)
        }
    }
    if err := cmd.Start(); err != nil {
        return err
    }
```
看不明白的话，去标准库里恶补一下cmd怎么用。 cmd.Start()异步创建进程，创建完直接返回

所以创建一个协程等待子进程退出
```
    go func() {
        cmd.Wait()
        close(r.daemonWaitCh)
    }() // Reap our child when needed

```

## docker-containerd-shim是何方神圣 与containerd和runc又有什么关系？
代码中的一句话解释：`shim for container lifecycle and reconnection`, 容器生命周期和重连, 所以可以顺着这个思路去看。

先看containerd/linux/runtime.go里的一段代码：
Runtime 的Create方法里有这一行,这里的Runtime对象也是注册到register里面的，可以看init函数，然后containerd进程启动时去加载了这个Runtime
```
s, err := newShim(path, r.remote)
```
缩减版内容:
```
func newShim(path string, remote bool) (shim.ShimClient, error) {
    l, err := sys.CreateUnixSocket(socket) //创建了一个UnixSocket
    cmd := exec.Command("containerd-shim")
    f, err := l.(*net.UnixListener).File()
    cmd.ExtraFiles = append(cmd.ExtraFiles, f) //留意一下这个，非常非常重要，不知道这个原理可能就看不懂shim里面的代码了
    if err := reaper.Default.Start(cmd); err != nil { //启动了一个shim进程
    }
    return connectShim(socket) // 这里返回了与shim进程通信的客户端
}
```

再去看看shim的代码：
shim进程启动干的最主要的一件事就是启动一个grpc server:
```
if err := serve(server, "shim.sock"); err != nil {
```
进去一探究竟：
```
func serve(server *grpc.Server, path string) error {
    l, err := net.FileListener(os.NewFile(3, "socket"))
    logrus.WithField("socket", path).Debug("serving api on unix socket")
    go func() {
        if err := server.Serve(l); err != nil &&
        }
    }()
}
```
我曾经因为这个`os.NewFile(3, "socket")`看了半天看不懂，为啥是3？联系`cmd.ExtraFiles = append(cmd.ExtraFiles, f)` 创建shim进程时的这句，问题解决了。

这个3的文件描述符，就是containerd用于创建UnixSocket的文件，这样containerd的client刚好与这边启动的 grpc server连接上了，可以远程调用其接口了：
```
type ContainerServiceClient interface {
    Create(ctx context.Context, in *CreateRequest, opts ...grpc.CallOption) (*CreateResponse, error)
    Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*google_protobuf.Empty, error)
    Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error)
    Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*containerd_v1_types1.Container, error)
    List(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*ListResponse, error)
    Kill(ctx context.Context, in *KillRequest, opts ...grpc.CallOption) (*google_protobuf.Empty, error)
    Events(ctx context.Context, in *EventsRequest, opts ...grpc.CallOption) (ContainerService_EventsClient, error)
    Exec(ctx context.Context, in *ExecRequest, opts ...grpc.CallOption) (*ExecResponse, error)
    Pty(ctx context.Context, in *PtyRequest, opts ...grpc.CallOption) (*google_protobuf.Empty, error)
    CloseStdin(ctx context.Context, in *CloseStdinRequest, opts ...grpc.CallOption) (*google_protobuf.Empty, error)
}

```
[containerd与shim通信模型介绍](https://github.com/fanux/fanux.github.io/tree/master/demo/grpc)

再看shim与runc的关系，这个比较简单了，直接进入shim service 实现的Create方法即可
```
sv = shim.New(path)
```
```
func (s *Service) Create(ctx context.Context, r *shimapi.CreateRequest) (*shimapi.CreateResponse, error) {
    process, err := newInitProcess(ctx, s.path, r)
    return &shimapi.CreateResponse{
        Pid: uint32(pid),
    }, nil
}
```
进入到newInitProcess里面：
```
func newInitProcess(context context.Context, path string, r *shimapi.CreateRequest) (*initProcess, error) {
    runtime := &runc.Runc{
        Command:      r.Runtime,
        Log:          filepath.Join(path, "log.json"),
        LogFormat:    runc.JSON,
        PdeathSignal: syscall.SIGKILL,
    }
    p := &initProcess{
        id:     r.ID,
        bundle: r.Bundle,
        runc:   runtime,
    }
  
    if err := p.runc.Create(context, r.ID, r.Bundle, opts); err != nil {
        return nil, err
    }
    return p, nil
}
```
可以看到，在这里调用了runc的API去真正执行创建容器的操作。其本质是调用了`runc create --bundle [bundle] [containerid]` 命令,在此不多作介绍了

## docker-containerd-ctr 与 docker-containerd 
ctr 是一个containerd的client，之间通过proto rpc通信, containerd监听了unix:///run/containerd/containerd.sock。
```
[root@dev-86-201 ~]# docker-containerd --help
NAME:
   containerd - High performance container daemon

USAGE:
   docker-containerd [global options] command [command options] [arguments...]

VERSION:
   0.2.0 commit: 0ac3cd1be170d180b2baed755e8f0da547ceb267

COMMANDS:
   help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug                            enable debug output in the logs
   --state-dir "/run/containerd"                runtime state directory
   --metrics-interval "5m0s"                    interval for flushing metrics to the store
   --listen, -l "unix:///run/containerd/containerd.sock"    proto://address on which the GRPC API will listen
   --runtime, -r "runc"                        name or path of the OCI compliant runtime to use when executing containers
   --runtime-args [--runtime-args option --runtime-args option]    specify additional runtime args
   --shim "containerd-shim"                    Name or path of shim
   --pprof-address                         http address to listen for pprof events
   --start-timeout "15s"                    timeout duration for waiting on a container to start before it is killed
   --retain-count "500"                        number of past events to keep in the event log
   --graphite-address                         Address of graphite server
   --help, -h                            show help
   --version, -v                        print the version
```
```
[root@dev-86-201 ~]# docker-containerd-ctr --help
NAME:
   ctr - High performance container daemon cli

USAGE:
   docker-containerd-ctr [global options] command [command options] [arguments...]

VERSION:
   0.2.0 commit: 0ac3cd1be170d180b2baed755e8f0da547ceb267

COMMANDS:
   checkpoints    list all checkpoints
   containers    interact with running containers
   events    receive events from the containerd daemon
   state    get a raw dump of the containerd state
   version    return the daemon version
   help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug                        enable debug output in the logs
   --address "unix:///run/containerd/containerd.sock"    proto://address of GRPC API
   --conn-timeout "1s"                    GRPC connection timeout
   --help, -h                        show help
   --version, -v                    print the version
```

## 容器创建过程分析

## 网络模块分析

## 镜像模块分析
