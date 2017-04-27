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
