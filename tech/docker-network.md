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

## network namespace实践
使用ip命令，如果没有的话安装一下：`yum install net-tools`

基本命令：
```
ip netns add nstest  # 创建一个net namespace
ip netns list        # 查看net namespace列表
ip netns delete nstest # 删除
ip netns exec [ns name] command # 到对应的ns里去执行命令
ip netns exec [ns name] bash # 在ns中使用bash,需要要ns中做一系列操作时方便
```

开启ns中的回环设备,以创建的nstest为例
```
ip netns exec nstest ip link set dev lo up
```
在主机上创建两个虚拟网卡两张网卡是linux设备对
```
ip link set add veth-a type veth peer name veth-b
```
添加veth-b到nstest中
```
ip link set veth-b netns nstest
```
验证：
```
[root@dev-86-208 ~]# ip netns exec nstest ip link
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN mode DEFAULT
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
251: veth-b@if252: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN mode DEFAULT qlen 1000
    link/ether aa:0a:7d:01:06:d3 brd ff:ff:ff:ff:ff:ff link-netnsid 0
```
为网卡设置ip并启动：
```
[root@dev-86-208 ~]# ip addr add 10.0.0.1/24 dev veth-a
[root@dev-86-208 ~]# ip link set dev veth-a up

[root@dev-86-208 ~]# ip netns exec nstest ip addr add 10.0.0.2/24 dev veth-b
[root@dev-86-208 ~]# ip netns exec nstest ip link set dev veth-b up

设置完ip，自动添加了这个路由
[root@dev-86-208 ~]# route
Kernel IP routing table
Destination     Gateway         Genmask         Flags Metric Ref    Use Iface
default         10.1.86.1       0.0.0.0         UG    100    0        0 eth0
10.0.0.0        0.0.0.0         255.255.255.0   U     0      0        0 veth-a # 目的地址是10.0.0.0/24的就从这张网卡发出
10.1.86.0       0.0.0.0         255.255.255.0   U     100    0        0 eth0
172.17.0.0      0.0.0.0         255.255.0.0     U     0      0        0 docker0
172.18.0.0      0.0.0.0         255.255.0.0     U     0      0        0 br-4b03f208bc30

ns里面的路由表
[root@dev-86-208 ~]# ip netns exec nstest ip route
10.0.0.0/24 dev veth-b  proto kernel  scope link  src 10.0.0.2
```
验证相互ping：
```
[root@dev-86-208 ~]# ip netns exec nstest ping 10.0.0.1
PING 10.0.0.1 (10.0.0.1) 56(84) bytes of data.
64 bytes from 10.0.0.1: icmp_seq=1 ttl=64 time=0.043 ms
64 bytes from 10.0.0.1: icmp_seq=2 ttl=64 time=0.032 ms

[root@dev-86-208 ~]# ping 10.0.0.2
PING 10.0.0.2 (10.0.0.2) 56(84) bytes of data.
64 bytes from 10.0.0.2: icmp_seq=1 ttl=64 time=0.069 ms
64 bytes from 10.0.0.2: icmp_seq=2 ttl=64 time=0.024 ms
```
### Docker bridge的网络
我们去创建两个ns（ns1 与 ns2）模拟两个容器，创建四张网卡（两对设备对）模仿容器网卡。
```
 brtest
   |          +-------------+
   |-veth1 <--|--> eth1 ns1 |  
   |          |-------------+
   |-veth2 <--|--> eth1 ns2 |  
   |          +-------------+
````
再在宿主机上创建一个网桥brtest模拟docker0网桥，将veth1和veth2桥接到上面。

添加namespace:
```
[root@dev-86-208 ~]# ip netns add ns1
[root@dev-86-208 ~]# ip netns add ns2
[root@dev-86-208 ~]# ip netns list
ns2
ns1
test1 (id: 3)
nstest (id: 2)

[root@dev-86-208 ~]# ip netns exec ns1 ip link set dev lo up
[root@dev-86-208 ~]# ip netns exec ns2 ip link set dev lo up
```

添加网卡对：
```
[root@dev-86-208 ~]# ip link add veth1 type veth peer name eth1
[root@dev-86-208 ~]# ip link set eth1 netns ns1
[root@dev-86-208 ~]# ip link add veth2 type veth peer name eth1
[root@dev-86-208 ~]# ip link set eth1 netns ns2

[root@dev-86-208 ~]# ip netns exec ns1 ip link
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN mode DEFAULT
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
255: eth1@if256: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN mode DEFAULT qlen 1000
    link/ether ae:93:ba:2c:54:93 brd ff:ff:ff:ff:ff:ff link-netnsid 0
[root@dev-86-208 ~]# ip netns exec ns2 ip link
257: eth1@if258: <BROADCAST,MULTICAST> mtu 1500 qdisc noop state DOWN mode DEFAULT qlen 1000
    link/ether 3a:a6:f3:27:9d:83 brd ff:ff:ff:ff:ff:ff link-netnsid 0
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN mode DEFAULT
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
```

配置地址：
```
[root@dev-86-208 ~]# ip netns exec ns1 ip addr add 172.17.1.1/24 dev eth1
[root@dev-86-208 ~]# ip netns exec ns2 ip addr add 172.17.1.2/24 dev eth1

[root@dev-86-208 ~]# ip netns exec ns1 ip link set dev eth1 up
[root@dev-86-208 ~]# ip netns exec ns2 ip link set dev eth1 up
```

创建网桥：
```
[root@dev-86-208 ~]# brctl addbr brtest
[root@dev-86-208 ~]# ifconfig brtest
brtest: flags=4098<BROADCAST,MULTICAST>  mtu 1500
        ether 1e:60:eb:c1:e6:d0  txqueuelen 0  (Ethernet)
        RX packets 0  bytes 0 (0.0 B)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 0  bytes 0 (0.0 B)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0

[root@dev-86-208 ~]# brctl addif brtest veth1
[root@dev-86-208 ~]# brctl addif brtest veth2

[root@dev-86-208 ~]# ifconfig brtest up
[root@dev-86-208 ~]# ifconfig veth1 up      # 主机上这两张网卡工作在数据链路层，因此不需要设置ip也能通
[root@dev-86-208 ~]# ifconfig veth2 up
```
恭喜两个eth1之间可以通了：
```
[root@dev-86-208 ~]# ip netns exec ns1 ping 172.17.1.2
PING 172.17.1.2 (172.17.1.2) 56(84) bytes of data.
64 bytes from 172.17.1.2: icmp_seq=1 ttl=64 time=0.063 ms
64 bytes from 172.17.1.2: icmp_seq=2 ttl=64 time=0.022 ms

[root@dev-86-208 ~]# ip netns exec ns2 ping 172.17.1.1
PING 172.17.1.1 (172.17.1.1) 56(84) bytes of data.
64 bytes from 172.17.1.1: icmp_seq=1 ttl=64 time=0.038 ms
64 bytes from 172.17.1.1: icmp_seq=2 ttl=64 time=0.041 ms
```
当然想在主机上能ping通容器的话需要给brtest加ip：
```
[root@dev-86-208 ~]# ip addr add 172.17.1.254/24 dev brtest
[root@dev-86-208 ~]# ping 172.17.1.1
PING 172.17.1.1 (172.17.1.1) 56(84) bytes of data.
64 bytes from 172.17.1.1: icmp_seq=1 ttl=64 time=0.046 ms
64 bytes from 172.17.1.1: icmp_seq=2 ttl=64 time=0.030 ms
```
以上操作就是docker bridge模式的模型
