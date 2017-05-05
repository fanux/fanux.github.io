## 基于Unixsocket的父子进程之间的grpc通信
本demo 基于grpc的helloworld例子修改而来，主要修改的地方在于建立链接的部分。
client启动时自动启动server, 通过给server传Unixsocket文件参数，而不需要真正建立一个真的文件。

这是docker containerd 与shim的通信模型。

## 使用方法
到greeter_client 和greeter_server 目录分别执行go build
然后直接运行client即可。

## 相关链接
通信Demo地址
[Containerd 与 shim关系](http://lameleg.com/tech/docker-architech.html)
