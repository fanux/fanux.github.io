# 使用kubeadm安装安全高可用kubernetes集群

> 总体流程：
>
> 解压后在master 上 cd shell  && sh init.sh && master.sh （注意因为脚本用的相对路径所以不再当前目录会找不到文件）
>
> 在node上 cd shell && sh init.sh  
>
> 然后在node上执行master输出的join命令即可

## 提前准备

假设构建一个2master+3node的k8s集群，需要5台节点共同的条件如下：（建议做成**模板**以便离线环境安装）

1. 建议安装docker17.x ，cent下安装步骤参考docker官网即可
2. 建议二进制方法安装好docker-compose，步骤参考后文
3. 建议永久关闭selinux和swap以免后续问题
4. 建议停止并关闭firewalld/iptables等防火墙
5. 新的节点启动后记得改网络名 `hostnamectl set-hostname masterX` 
6. 节点之间要能互通内网环境稳定

## 系统架构图

```bash
          kubectl dashboard
                 |
                 V 
     +------------------------+ join
     | LB  10.1.245.94        | <--- Nodes
     +------------------------+
     |                                                   
     |--master1 manager1 schedule1   10.1.245.93                                                
     |--master2 manager2 schedule2   10.1.245.95    =============>  etcd cluster  http://10.1.245.93:2379,http://10.1.245.94:2379,http://10.1.245.95:2379
     |--master3 manager3 schedule3   10.1.245.94                                              

```

## 安装包介绍

解压完之后看到如下目录：

```
├── bin          所需要的k8s相关的bin文件
│   ├── kubeadm
│   ├── kubectl
│   └── kubelet
├── image        依赖的所有镜像包
│   └── images.tar
├── out          所有的配置文件
│   ├── dashboard  dashboard相关配置
│   │   ├── dashboard-admin.yaml
│   │   └── kubernetes-dashboard.yaml
│   ├── etcd  etcd相关配置
│   │   ├── etcd-docker-compose-0.yml
│   │   ├── etcd-docker-compose-1.yml
│   │   └── etcd-docker-compose-2.yml
│   ├── haproxy  haproxy相关配置
│   │   └── haproxy.cfg
│   ├── heapster   heapster相关yaml配置
│   │   ├── influxdb
│   │   │   ├── grafana.yaml
│   │   │   ├── heapster.yaml
│   │   │   └── influxdb.yaml
│   │   └── rbac
│   │       └── heapster-rbac.yaml
│   ├── kube    k8s自身配置
│   │   ├── 10-kubeadm.conf
│   │   ├── config    kubeadm配置
│   │   └── kubelet.service
│   ├── kubeinit.json  忽略
│   └── net  网络相关配置
│       ├── calico.yaml
│       └── calicoctl.yaml
└── shell    初始化脚本
    ├── init.sh   初始化节点,安装bin文件，systemd配置等
    └── master.sh  执行kubeadm init和其它组件

```

## 初始化节点

因为解压后包大小**大约2G**，所以解压时间较长，如果机器性能太弱可以选择先在一台性能好的节点上解压，然后`scp -r xxx root@ip:/root` 的方式分发解压后的包到其他节点。用网络带宽换取时间

**集群中所有节点都需要执行`cd shell && sh init.sh` ,如果单master启动其中master节点还需要执行 `sh master.sh`**  （多master这里不启动，使用后面的 kubeadm init --config config）

<u>有以下需要注意的事项：</u> 

1. 如果同时几个节点加载解压，可能CPU或磁盘跑满，加载镜像时出现这种提示`Failed to execute operation: Connection timed out` .遇到这个提示就**重新执行脚本**
2. 执行`init.sh` 的时间大概要10分钟 （普通双核CPU 4G内存）  
3. 执行`master.sh` 可能会提示`Permission denied` 权限不足。需要chmod +x
4. cgroups驱动需要选择docker17.0x版本，就不需要去调整了，如果是1.1x版本的docker需要手动修改kubelet的启动文件里面的cgroups配置  （修改位置`/etc/systemd/system/kubelet.service.d`）   
5. 提前修改默认的init 或者手动执行`sysctl  -w net.ipv4.ip_forward=1` 不然第七行报错

**执行完成后通过命令查看`kubectl get pod -n kube-system` ,状态全为Running正常**

## 起动etcd集群

在out/etcd目录下有相关模板，启动多个节点时修改成自己的ip地址 其它两个节点照抄，修改ip即可, 镜像名替换成之前导入的，可用docker images查看一下. 应改为  `gcr.io/google_containers/etcd-amd64:3.1.11` ，实际就是版本号改一下即可。

IP修改的地方比较多，建议谨慎一点以免把端口给不小心删了或者写重了，除了最后一行其他的ip换成当前主机ip，最后一行按名字顺序换成对应的ip  

A.使用docker-compose启动，如果没装：

```
$ pip install docker-compose
```

B.使用二进制包启动docker-compose（离线可选）

```bash
$ wget https://github.com/docker/compose/releases/download/1.18.0/docker-compose-Linux-x86_64  #官方推荐是用curl,不建议
$ mv docker-compose-Linux-x86_64 /usr/local/bin/docker-compose && chmod a+x /usr/local/bin/docker-compose  #也有写+x的.
#这样就完成了,测试
$ docker-compose version  #下面是正常输出
docker-compose version 1.18.0, build 8dd22a9
docker-py version: 2.6.1
CPython version: 2.7.13
OpenSSL version: OpenSSL 1.0.1t  3 May 2016
```

三个节点分别启动：

```bash
$ docker-compose -f etcd.yaml up -d  #这里的etcd.yaml是 etcd-xx.yam 的等价意思么？
```

检查是不是安装成功:

```bash
$ docker exec etcd_infra0 etcdctl member list  #master上可能运行报错容易正在重启。。node上可以
#成功应该是类似显示
5ded6dd284b89d31: name=infra1 peerURLs=http://10.230.204.153:2380 clientURLs=http://10.230.204.153:2379 isLeader=true
6d4b5eee32c1497a: name=infra0 peerURLs=http://10.230.204.150:2380 clientURLs=http://10.230.204.150:2379 isLeader=false
729d9cd56debb1a1: name=infra2 peerURLs=http://10.230.204.154:2380 clientURLs=http://10.230.204.154:2379 isLeader=false
```

## kubeadm配置

修改配置 `out/kube/config `文件 

```yaml
apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
apiServerCertSANs:    #此处填所有的masterip和lbip和其它你可能需要通过它访问apiserver的地址和域名或者主机名等，如阿里fip，证书中会允许这些ip
- 172.31.244.231
- 172.31.244.232
- 172.31.244.233
- 172.31.244.234
- master1
- master2
- master3
- node1
- 47.75.1.72

etcd:
  endpoints:   #这里填之前安装的etcd集群地址列表，修改IP地址
  - http://172.31.244.232:2379
  - http://172.31.244.233:2379
  - http://172.31.244.234:2379

apiServerExtraArgs:
  endpoint-reconciler-type: lease

networking:
  podSubnet: 192.168.0.0/16  #不用改
kubernetesVersion: v1.9.2  #不用改
featureGates:  #不用改
   CoreDNS: true
```

然后执行：

```bash
$ kubeadm init --config config
```

把成功后的kubeadm join命令存在文件里，那东西不能丢了

PS：这有个疑问，先执行master init再执行这个会报错冲突  前面的不执行也不行。。都会报错,更新如果直接搭多master之前不执行`master.sh` 

## 启动calico等

`mkdir ~/.kube && cp /etc/kubernetes/admin.conf ~/.kube/config ` （如果已经存在请校验一下是否相同,不确定建议删掉重新cp过去）

```bash
kubectl apply -f out/net/calico.yaml
kubectl apply -f out/heapster/influxdb
kubectl apply -f out/heapster/rbac
kubectl apply -f out/dashboard
```

然后访问https://master0IP:32000端口即可，启动后会发现heapster组件没有启动因为此时没有node工作节点加入，调度器还无法安排

## 启动多个master

第一个master我们称之为master0, 现在在master1上同样拷贝压缩包执行 `cd shell && sh init.sh`

别的master节点初始化好之后，把第一个master的/etc/kubernetes/pki目录拷贝到别的master节点上

```bash
$ scp -r /etc/kubernetes/pki root@10.1.245.93:/etc/kubernetes/
```

删除pki目录下的apiserver.crt 和 apiserver.key文件，注意如果不删除会只能看到一个master，是不正常的。

同样使用master0上的out/kube/config文件，复制内容，拷贝到master1上，`scp /out/kube/config  root@10.230.204.151:/root/` 执行`kubeadm init --config config`

master2节点同master1

## 启动loadbalance

我比较推荐使用四层代理 HAproxy配置out/haproxy目录:

`vi out/haproxy/haproxy.cfg` 

```
global
  daemon
  log 127.0.0.1 local0
  log 127.0.0.1 local1 notice
  maxconn 4096

defaults
  log               global
  retries           3
  maxconn           2000
  timeout connect   5s
  timeout client    50s
  timeout server    50s

frontend k8s
  bind *:6444
  mode tcp
  default_backend k8s-backend

backend k8s-backend
  balance roundrobin
  mode tcp
  server k8s-1 10.1.245.93:6443 check #替换成三个master的地址
  server k8s-1 10.1.245.94:6443 check
  server k8s-2 10.1.245.95:6443 check
```

把这个文件拷贝在`cp out/haproxy/haproxy.cfg /etc/haproxy/haproxy.cfg`

```bash
$ docker run --net=host -v /etc/haproxy:/usr/local/etc/haproxy --name ha -d haproxy:1.7
```

## 修改kubeproxy配置

```bash
$ kubectl -n kube-system edit configmap kube-proxy
```

找到master地址，修改成LB地址。6444端口  （疑问：到底是修改哪个参数的。。是cluster下的server参数么？）

## join node节点

还是在node节点执行第一个master输出的命令，就是上面haproxy的地址（ha的地址是master1的？）

```bash
$ kubeadm join --token <token> 10.1.245.94:6443 --discovery-token-ca-cert-hash sha256:<hash>
```

## 修改node节点kubelet配置

`vi /etc/kubernetes/kubelet.conf ` 同样把地址修改成LB地址,如：`10.1.245.94:6444`

## kubectl配置

修改`~/.kube/config`文件,ip改成LB的ip `10.1.245.94:6444` 

或者通过命令修改：

```bash
$ kubectl config set-cluster kubernetes --server=https://47.52.227.242:6444 --kubeconfig=$HOME/.kube/config
```

### 启动多DNS副本

```bash
$ kubectl edit deploy coredns -n kube-system
```

replicas: 3

```bash
[root@master1 ~]$ kubectl get pod -n kube-system -o wide|grep core
coredns-65dcdb4cf-4j5s8                  1/1       Running   0          39m       192.168.137.65    master1
coredns-65dcdb4cf-ngx4h                  1/1       Running   0          38s       192.168.180.1     master2
coredns-65dcdb4cf-qbsr6                  1/1       Running   0          38s       192.168.166.132   node1
```

这样，启动了三个dns

### 验证与测试

```bash
$ kubectl run test --image busybox sleep 10000
$ kubectl exec your-busybox-pod-name nslookup kubernetes
```

杀非LB的master，多次测试看创建pod与dns是否正常，还可以telnet 10.96.0.1 443 去验证clusterip是否正常