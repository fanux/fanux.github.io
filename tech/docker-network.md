## 网络概述
1. 端口映射：
```
$ docker run -p 8080:80 nginx:latest
```
如果没有这个-p，会发现启动了nginx但是无法通过宿主机访问到web服务，而使用了-p参数后就可以通过访问主机的8080断开去访问nginx了。
端口映射的原理是作了net转发

2. 共享主机网络:
```
$ docker run --net=host nginx:latest
```
这种容器没有自己的网络，完全共享主机的网络，所以可以通过主机ip直接访问容器服务。 坏处是容器与其它容器端口冲突

3. link网络
```
$ docker run --name mysql mysql:latest
$ docker run --link=mysql nginx:latest
```
这样nginx可以通过容器名去访问mysql，其原理是在nginx容器中的/etc/hosts中加入了mysql主机名解析。这种共享不可跨主机

```
$ docker run --rm -it --name c1 centos:latest /bin/bash
```
```
$ docker run --rm -it --name c2 --link c1  centos:latest /bin/bash
[root@178d290d873c /]# cat /etc/hosts
127.0.0.1    localhost
::1    localhost ip6-localhost ip6-loopback
fe00::0    ip6-localnet
ff00::0    ip6-mcastprefix
ff02::1    ip6-allnodes
ff02::2    ip6-allrouters
172.17.0.4    c1 3b7b15fa7e20   # 看这里
172.17.0.5    178d290d873c
```

4. none模式
容器不创建网络，需要自行定义

5. overlay网络
进群中常用的网络模式，使用vxlan等技术作了一层覆盖，使每个容器有自己独立的ip并可跨主机通信。

6. 共享容器网络

如kubernetes里pod的实现，pod是多个容器的集合，这些容器都共享了同一个容器的网络，那么这些容器就如同在一个host上一样。

### bridge原理
在宿主机上ifconfig:
```
docker0: flags=4163<UP,BROADCAST,RUNNING,MULTICAST>  mtu 1500
        inet 172.17.0.1  netmask 255.255.0.0  broadcast 0.0.0.0
        inet6 fe80::42:a4ff:fe60:b79d  prefixlen 64  scopeid 0x20<link>
        ether 02:42:a4:60:b7:9d  txqueuelen 0  (Ethernet)
        RX packets 23465  bytes 3407255 (3.2 MiB)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 24676  bytes 22031766 (21.0 MiB)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0

vethcd2d45d: flags=4163<UP,BROADCAST,RUNNING,MULTICAST>  mtu 1500
        inet6 fe80::c4d6:dcff:fe7d:5f44  prefixlen 64  scopeid 0x20<link>
        ether c6:d6:dc:7d:5f:44  txqueuelen 0  (Ethernet)
        RX packets 415  bytes 82875 (80.9 KiB)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 372  bytes 379450 (370.5 KiB)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0
```
docker0是一个虚拟网桥，类似一个交换机的存在。 veth开头的网卡就是为容器分配的一个设备,但是要注意这不是容器中的设备。由于linux物理网卡只能出现在一个namespace中，所以只能用虚拟设备给容器创建独立的网卡。

docker network inspect bridge 看一下,这是给容器内部分配的地址：
```
"Containers": {
    "ac8c983592f06d585a75184b4dcd012338645fb7fa60b07c722f59ce43ceb807": {
        "Name": "sick_snyder",
        "EndpointID": "0755707344f30c40d686a2b4fdcabf45d6e1a64f8de8618b9a3a8c8e5b203ddc",
        "MacAddress": "02:42:ac:11:00:02",
        "IPv4Address": "172.17.0.2/16",
        "IPv6Address": ""
    }
}
```

再引入一个概念：linux设备对，类似管道一样，在一端写另一端就可以读,容器内的eth0就与这个veth是一对设备对

```
           docker0         eth0 -> 宿主机
        ---------------    ----
         |          |
        vethx      vethy
        ----       ----
          |          |    ---->设备对
     +----+---+ +----+---+
     |  eth0  | |  eth0  |
     +--------+ +--------+
      容器1       容器2
```
单有这些还不够，还需要iptables对包作一些处理,下文细说。有了这些理论，再去顺着这个思路去读网络模块的代码

### bridge实现源码解析
##### docker0网桥的建立
##### 创建容器时宿主机上的网络操作
##### 创建容器时容器内部的网络操作

### overlay网络代表flannel原理
