## 容器信号使用
我们跑在容器中的程序通常想在容器退出之前做一些清理操作，比较常用的方式是监听一个信号，延迟关闭容器。

docker提供了这样的功能：
```
╰─➤  docker stop --help

Usage:  docker stop [OPTIONS] CONTAINER [CONTAINER...]

Stop one or more running containers

Options:
      --help       Print usage
  -t, --time int   Seconds to wait for stop before killing it (default 10)
```

docker 1.13以上版本在创建容器时可直接指定STOP_TIMEOUT 和STOP_SIGNAL参数:
```
$ docker run --help
...
--stop-signal string                    Signal to stop a container, SIGTERM by default (default "SIGTERM")
--stop-timeout int                      Timeout (in seconds) to stop a container
...
```

但是。。。

我们测试一个：
```
package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    fmt.Println("signal test")
    go func() {
        for {
            c := make(chan os.Signal, 1)
            signal.Notify(c, syscall.SIGTERM)
            s := <-c
            fmt.Println("Got signal:", s)
        }
    }()
    time.Sleep(time.Second * 100)
}
```

Dockerfile:
```
FROM dev.reg.iflytek.com/base/golang:1.8.0
COPY main.go .
RUN go build -o signal && cp signal $GOPATH/bin
CMD signal  
```
构建：
```
docker build -t dev.reg.iflytek.com/test/signal:latest .
```
运行：
```
docker run --name signal dev.reg.iflytek.com/test/signal:latest
```
再开一终端，运行：
```
docker stop -t 10 signal
```
发现并没有打印出Got signal:... 监听信号失败。

问题再于：我们docker inspect signal看一下
可以看到
```
Path:/bin/sh
Args:[
  -c,
  signal
]
```
或者docker exec signal ps 看一下可以看到pid为1的进程并不是signal, 而是shell.

所以原因找到了，是因为docker engine只给pid为1的进程发送信号，sh收到了信号而我们想要的signal进程没有收到信号

解决办法：
```
FROM dev.reg.iflytek.com/base/golang:1.8.0
COPY main.go .
RUN go build -o signal && cp signal $GOPATH/bin
CMD ["signal"]  # 不能写成 CMD signal, 这会直接exec，否则会以shell的方式派生子进程。
```
