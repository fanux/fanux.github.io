一些工具安装：
yum install bridge-utils  # 如果brctl不能用

yum install net-tools     # 如果route命令不能用

# 单节点上使用ovs vlan划分网络
启动四个容器：
```
docker run -itd --name con1 ubuntu:14.04 /bin/bash
docker run -itd --name con2 ubuntu:14.04 /bin/bash
docker run -itd --name con3 ubuntu:14.04 /bin/bash
docker run -itd --name con4 ubuntu:14.04 /bin/bash
```
创建ovs网桥并绑定端口
```
pipework ovs0 con1 192.168.0.1/24 @100
pipework ovs0 con2 192.168.0.2/24 @100

pipework ovs0 con3 192.168.0.3/24 @200
pipework ovs0 con4 192.168.0.4/24 @200
```
这样con1 和 con2是通的，con3和con4是通的，这个比较简单。pipework干的具体的事是：
```
ovs-vsctl add-port ovs0 [容器的虚拟网卡设备] tag=100
```
ovs划分vlan处理的原理也非常简单，包进入到switch时打上tag，发出去时去掉tag，发出去的端口与包的tag不匹配时不处理，这便实现了二层隔离。

access端口与trunk端口的区别是，trunk端口可接受多个tag。

# 跨主机vlan
准备两个主机，在host1上：
```
docker run -itd --name con1 ubuntu:14.04 /bin/bash
docker run -itd --name con2 ubuntu:14.04 /bin/bash
pipework ovs0 con1 192.168.0.1/24 @100
pipework ovs0 con2 192.168.0.2/24 @200
```
如果是单张网卡的话，把eth0桥接到switch上时会造成网络中断，所以以下几步不要通过ssh操作：
如果非得ssh去操作的话把以下命令放在一条命令中执行（用&&连接各个命令）
```
ovs-vsctl add-port ovs0 eth0
ifconfig ovs0 10.1.86.201 netmask 255.255.255.0   # 这里地址和掩码与eth0的配置一致
ifconfig ovs0 up
ifconfig eth0 0.0.0.0
route add default gw 10.1.86.1  # 执行之前看看eth0的gw是什么，保持一致，这样eth0就桥接到ovs0上去了。
```
查看switch端口：
```
[root@dev-86-204 ~]# ovs-vsctl show
c5ddf9e8-daac-4ed2-80f5-16e6365425fa
    Bridge "ovs0"
        Port "veth1pl41885"
            tag: 100
            Interface "veth1pl41885"
        Port "ovs0"
            Interface "ovs0"
                type: internal
        Port "eth0"
            Interface "eth0"
        Port "veth1pl41805"
            tag: 200
            Interface "veth1pl41805"
    ovs_version: "2.5.1"
```

在host2上：
```
docker run -itd --name con3 ubuntu:14.04 /bin/bash
docker run -itd --name con4 ubuntu:14.04 /bin/bash
pipework ovs0 con3 192.168.0.3/24 @100
pipework ovs0 con4 192.168.0.4/24 @200
```
同样要桥接eth0到ovs0上,同host1的操作，然后con1与con3可通，con2与con4可通.

# GRE实现overlay网络
linux内核需要3.11以上，本尊在3.10内核上实践失败，在虚拟机中升级内核时虚拟机启动不了，CPU飙到100%，以后再试。

# Vxlan实现跨主机通信
```
   host1:10.1.86.203 
   ovs0
    |
    |-veth1 <-------> eth1 192.168.0.3  con3
    |
    |-vxlan1-------------+
    |                    |
                         |
    host2:10.1.86.204    |
    ovs0                 |
     |                   |
     |-vxlan1------------+
     |
     |-veth1 <--------> eth1 192.168.0.4 con4
     |
```
可以看到con3和con4在搭建vxlan之前是无法通信的。

在host1上：
```
[root@dev-86-203 ~]# docker run --name con3 -itd ubuntu:14.04 /bin/bash
[root@dev-86-203 ~]# ovs-vsctl add-br ovs0
[root@dev-86-203 ~]# pipework ovs0 con3 192.168.0.3/24   # 给容器分配地址并挂到ovs0上

[root@dev-86-203 ~]# ovs-vsctl add-port ovs0 vxlan1 -- set interface vxlan1 type=vxlan options:remote_ip=10.1.86.204 options:key=flow # 创建vxlan

[root@dev-86-203 ~]# ovs-vsctl show
5e371797-db70-451c-a0f2-d70c6d00cd05
    Bridge "ovs0"
        Port "veth1pl3342"
            Interface "veth1pl3342"
        Port "ovs0"
            Interface "ovs0"
                type: internal
        Port "vxlan1"
            Interface "vxlan1"
                type: vxlan
                options: {key=flow, remote_ip="10.1.86.204"}
    ovs_version: "2.5.1"
```
host2上同理：
```
[root@dev-86-204 ~]# docker run --name con4 -itd ubuntu:14.04 /bin/bash
[root@dev-86-204 ~]# ovs-vsctl add-br ovs0
[root@dev-86-204 ~]# pipework ovs0 con4 192.168.0.4/24   # 给容器分配地址并挂到ovs0上

[root@dev-86-204 ~]# ovs-vsctl add-port ovs0 vxlan1 -- set interface vxlan1 type=vxlan options:remote_ip=10.1.86.203 options:key=flow # 创建vxlan

[root@dev-86-204 ~]# ovs-vsctl show
c5ddf9e8-daac-4ed2-80f5-16e6365425fa
    Bridge "ovs0"
        Port "ovs0"
            Interface "ovs0"
                type: internal
        Port "veth1pl52846"
            Interface "veth1pl52846"
        Port "vxlan1"
            Interface "vxlan1"
                type: vxlan
                options: {key=flow, remote_ip="10.1.86.203"}
    ovs_version: "2.5.1"
```
验证：
```
[root@dev-86-204 ~]# docker exec con4 ping 192.168.0.3  # con4容器中ping con3的地址，可通
PING 192.168.0.3 (192.168.0.3) 56(84) bytes of data.
64 bytes from 192.168.0.3: icmp_seq=1 ttl=64 time=0.251 ms
64 bytes from 192.168.0.3: icmp_seq=2 ttl=64 time=0.170 ms
```
