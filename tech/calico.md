# calico 网络原理

### node节点 装网络之前路由
```
[root@iZj6c3cqwumhn5jov661z7Z ~]# route -n
Kernel IP routing table
Destination     Gateway         Genmask         Flags Metric Ref    Use Iface
0.0.0.0         172.31.255.253  0.0.0.0         UG    0      0        0 eth0
169.254.0.0     0.0.0.0         255.255.0.0     U     1002   0        0 eth0
172.17.0.0      0.0.0.0         255.255.0.0     U     0      0        0 docker0
172.31.240.0    0.0.0.0         255.255.240.0   U     0      0        0 eth0
```

网卡：
```
[root@iZj6c3cqwumhn5jov661z7Z ~]# ifconfig
docker0: flags=4099<UP,BROADCAST,MULTICAST>  mtu 1500
        inet 172.17.0.1  netmask 255.255.0.0  broadcast 0.0.0.0
        ether 02:42:cb:02:65:a3  txqueuelen 0  (Ethernet)
        RX packets 0  bytes 0 (0.0 B)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 0  bytes 0 (0.0 B)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0

eth0: flags=4163<UP,BROADCAST,RUNNING,MULTICAST>  mtu 1500
        inet 172.31.242.156  netmask 255.255.240.0  broadcast 172.31.255.255
        ether 00:16:3e:00:89:75  txqueuelen 1000  (Ethernet)
        RX packets 72958  bytes 105395247 (100.5 MiB)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 15711  bytes 1583143 (1.5 MiB)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0

lo: flags=73<UP,LOOPBACK,RUNNING>  mtu 65536
        inet 127.0.0.1  netmask 255.0.0.0
        loop  txqueuelen 1  (Local Loopback)
        RX packets 0  bytes 0 (0.0 B)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 0  bytes 0 (0.0 B)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0
```

iptables:
```
[root@iZj6c3cqwumhn5jov661z7Z ~]# iptables -L
Chain INPUT (policy ACCEPT)
target     prot opt source               destination

Chain FORWARD (policy ACCEPT)
target     prot opt source               destination
DOCKER-ISOLATION  all  --  anywhere             anywhere
DOCKER     all  --  anywhere             anywhere
ACCEPT     all  --  anywhere             anywhere             ctstate RELATED,ESTABLISHED
ACCEPT     all  --  anywhere             anywhere
ACCEPT     all  --  anywhere             anywhere

Chain OUTPUT (policy ACCEPT)
target     prot opt source               destination

Chain DOCKER (1 references)
target     prot opt source               destination

Chain DOCKER-ISOLATION (1 references)
target     prot opt source               destination
RETURN     all  --  anywhere             anywhere
```
### master节点安装之前
```
[root@izj6cg11g0cdegoowj058ez ~]# route -n
Kernel IP routing table
Destination     Gateway         Genmask         Flags Metric Ref    Use Iface
0.0.0.0         172.31.255.253  0.0.0.0         UG    0      0        0 eth0
169.254.0.0     0.0.0.0         255.255.0.0     U     1002   0        0 eth0
172.17.0.0      0.0.0.0         255.255.0.0     U     0      0        0 docker0
172.31.240.0    0.0.0.0         255.255.240.0   U     0      0        0 eth0
```

```
[root@izj6cg11g0cdegoowj058ez ~]# ifconfig
docker0: flags=4099<UP,BROADCAST,MULTICAST>  mtu 1500
        inet 172.17.0.1  netmask 255.255.0.0  broadcast 0.0.0.0
        ether 02:42:37:67:91:64  txqueuelen 0  (Ethernet)
        RX packets 0  bytes 0 (0.0 B)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 0  bytes 0 (0.0 B)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0

eth0: flags=4163<UP,BROADCAST,RUNNING,MULTICAST>  mtu 1500
        inet 172.31.242.157  netmask 255.255.240.0  broadcast 172.31.255.255
        ether 00:16:3e:01:aa:f2  txqueuelen 1000  (Ethernet)
        RX packets 169779  bytes 246363004 (234.9 MiB)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 32790  bytes 3034892 (2.8 MiB)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0

lo: flags=73<UP,LOOPBACK,RUNNING>  mtu 65536
        inet 127.0.0.1  netmask 255.0.0.0
        loop  txqueuelen 1  (Local Loopback)
        RX packets 22682  bytes 6946686 (6.6 MiB)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 22682  bytes 6946686 (6.6 MiB)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0
```

```
[root@izj6cg11g0cdegoowj058ez ~]# iptables -L
Chain INPUT (policy ACCEPT)
target     prot opt source               destination
KUBE-SERVICES  all  --  anywhere             anywhere             /* kubernetes service portals */
KUBE-FIREWALL  all  --  anywhere             anywhere

Chain FORWARD (policy ACCEPT)
target     prot opt source               destination
DOCKER-ISOLATION  all  --  anywhere             anywhere
DOCKER     all  --  anywhere             anywhere
ACCEPT     all  --  anywhere             anywhere             ctstate RELATED,ESTABLISHED
ACCEPT     all  --  anywhere             anywhere
ACCEPT     all  --  anywhere             anywhere

Chain OUTPUT (policy ACCEPT)
target     prot opt source               destination
KUBE-SERVICES  all  --  anywhere             anywhere             /* kubernetes service portals */
KUBE-FIREWALL  all  --  anywhere             anywhere

Chain DOCKER (1 references)
target     prot opt source               destination

Chain DOCKER-ISOLATION (1 references)
target     prot opt source               destination
RETURN     all  --  anywhere             anywhere

Chain KUBE-FIREWALL (2 references)
target     prot opt source               destination
DROP       all  --  anywhere             anywhere             /* kubernetes firewall for dropping marked packets */ mark match 0x8000/0x8000

Chain KUBE-SERVICES (2 references)
target     prot opt source               destination
REJECT     udp  --  anywhere             10.96.0.10           /* kube-system/kube-dns:dns has no endpoints */ udp dpt:domain reject-with icmp-port-unreachable
REJECT     tcp  --  anywhere             10.96.0.10           /* kube-system/kube-dns:dns-tcp has no endpoints */ tcp dpt:domain reject-with icmp-port-unreachable
```

### 启动完一个pod
```
[root@izj6cg11g0cdegoowj058ez ~]# kubectl get pod --all-namespaces -o wide
NAMESPACE     NAME                                              READY     STATUS    RESTARTS   AGE       IP               NODE
kube-system   calico-etcd-gqqv2                                 1/1       Running   0          22m       172.31.242.157   izj6cg11g0cdegoowj058ez
kube-system   calico-kube-controllers-55449f8d88-pmh9h          1/1       Running   0          22m       172.31.242.157   izj6cg11g0cdegoowj058ez
kube-system   calico-node-77hm6                                 2/2       Running   0          22m       172.31.242.157   izj6cg11g0cdegoowj058ez
kube-system   calico-node-c6jx5                                 2/2       Running   0          9m        172.31.242.156   izj6c3cqwumhn5jov661z7z
kube-system   etcd-izj6cg11g0cdegoowj058ez                      1/1       Running   0          28m       172.31.242.157   izj6cg11g0cdegoowj058ez
kube-system   kube-apiserver-izj6cg11g0cdegoowj058ez            1/1       Running   1          27m       172.31.242.157   izj6cg11g0cdegoowj058ez
kube-system   kube-controller-manager-izj6cg11g0cdegoowj058ez   1/1       Running   1          28m       172.31.242.157   izj6cg11g0cdegoowj058ez
kube-system   kube-dns-545bc4bfd4-c9vsc                         3/3       Running   0          27m       192.168.83.129   izj6c3cqwumhn5jov661z7z
kube-system   kube-proxy-4btpd                                  1/1       Running   0          27m       172.31.242.157   izj6cg11g0cdegoowj058ez
kube-system   kube-proxy-cdvvf                                  1/1       Running   0          9m        172.31.242.156   izj6c3cqwumhn5jov661z7z
kube-system   kube-scheduler-izj6cg11g0cdegoowj058ez            1/1       Running   1          28m       172.31.242.157   izj6cg11g0cdegoowj058ez
sock-shop     carts-794f6cc876-8lfj8                            1/1       Running   0          5m        192.168.83.131   izj6c3cqwumhn5jov661z7z
sock-shop     carts-db-787f4b7896-v8qbn                         1/1       Running   0          5m        192.168.83.130   izj6c3cqwumhn5jov661z7z
```

```
[root@izj6cg11g0cdegoowj058ez ~]# route -n
Kernel IP routing table
Destination     Gateway         Genmask         Flags Metric Ref    Use Iface
0.0.0.0         172.31.255.253  0.0.0.0         UG    0      0        0 eth0
169.254.0.0     0.0.0.0         255.255.0.0     U     1002   0        0 eth0
172.17.0.0      0.0.0.0         255.255.0.0     U     0      0        0 docker0
172.31.240.0    0.0.0.0         255.255.240.0   U     0      0        0 eth0
192.168.83.128  172.31.242.156  255.255.255.192 UG    0      0        0 tunl0
192.168.179.0   0.0.0.0         255.255.255.192 U     0      0        0 *
```

```
[root@iZj6c3cqwumhn5jov661z7Z ~]# route -n
Kernel IP routing table
Destination     Gateway         Genmask         Flags Metric Ref    Use Iface
0.0.0.0         172.31.255.253  0.0.0.0         UG    0      0        0 eth0
169.254.0.0     0.0.0.0         255.255.0.0     U     1002   0        0 eth0
172.17.0.0      0.0.0.0         255.255.0.0     U     0      0        0 docker0
172.31.240.0    0.0.0.0         255.255.240.0   U     0      0        0 eth0
192.168.83.128  0.0.0.0         255.255.255.192 U     0      0        0 *
192.168.83.129  0.0.0.0         255.255.255.255 UH    0      0        0 cali513e7f6c556
192.168.83.130  0.0.0.0         255.255.255.255 UH    0      0        0 cali85f45a254f4
192.168.83.131  0.0.0.0         255.255.255.255 UH    0      0        0 cali65bda6d8589
192.168.179.0   172.31.242.157  255.255.255.192 UG    0      0        0 tunl0
```
