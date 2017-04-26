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
