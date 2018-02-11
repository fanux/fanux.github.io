# 使用kubeadm安装安全高可用kubernetes集群
[安装包地址](https://market.aliyun.com/products/57742013/cmxz025618.html?spm=5176.730005.0.0.TFKV5K#sku=yuncode1961800000) 如非高可用安装请忽略此教程，直接看产品页的三步安装。

> **单个master流程：**

 1. 解压后在master 上 cd shell  && sh init.sh ,然后sh master.sh（注意因为脚本用的相对路径所以不再当前目录会找不到文件）
 2. 在node上 cd shell && sh init.sh  。然后在node上执行master输出的join命令即可

> **高可用如下**

## 提前准备

假设构建一个3master+2node的k8s集群，需要5台节点共同的条件如下：

1. （`yum install -y docker是1.12.6版本需要改cg`）
   17.06安装教程：
   ```bash
   #0.删除老旧的
   $ yum remove -y docker*  #如果默认之前yum安装的1.12版本,可以这样删没装可以跳过此步
   #1.安装需要的包
   $ yum install -y yum-utils \
     device-mapper-persistent-data \
     lvm2
     
   #2.添加源,不然默认的找不到
   $ yum-config-manager \
       --add-repo \
       https://download.docker.com/linux/centos/docker-ce.repo
       
   #3.根据实际查找当前版本 (可选)
   $ yum list docker-ce --showduplicates | sort -r
   #4.如果确定了版本,直接安装,如果要装17。03直接修改下面数字即可
   $ yum install  docker-ce-17.06.1.ce  # 主意版本填写包名的格式.
   #5.开启docker服务,和开机启动
   $ systemctl start docker && systemctl enable docker
   ```

2. 建议二进制方法提前部署好docker-compose，步骤参考后文

3. 建议永久关闭selinux和swap以免后续问题

4. 建议停止并关闭firewalld/iptables等防火墙

5. 新的节点启动后记得改网络名 `hostnamectl set-hostname masterX` 

6. 节点之间要能互通内网环境稳定

7. 安装中出了问题要看日志journalctl -n 10 ,运行中的日志查看`tail -f 10 /var/log/messages` 

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
因为解压后包,然后`scp -r xxx root@ip:/root` 的方式分发解压后的包到其他节点

**集群中所有节点都需要执行`cd shell && sh init.sh` （如果只跑单个master那么还需要执行 `sh master.sh`** ，多master勿跑 ）

> 有以下需要注意的事项：  
1. 修改init.sh脚本在后面添加,如果二进制程序没可执行权限`chmod +x  /usr/bin/kube*` 
2. cgroups驱动需要选择docker17.0x版本，就不需要去调整了，如果是1.1x版本的docker需要**手动修改**kubelet的启动文件里面的cgroups配置为`systemd`   （修改位置`/etc/systemd/system/kubelet.service.d`）   与 docker info|grep Cg一致
3. 提前修改默认的init 或者手动执行`sysctl  -w net.ipv4.ip_forward=1` 不然第七行报错

**执行完成后通过命令查看`kubectl get pod -n kube-system` ,状态全为Running正常**

## 起动etcd集群

etcd集群安装使用docker-compose方式部署

A.使用docker-compose启动，如果没装：

```bash
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

在out/etcd目录下有相关模板`etcd-docker-compose-x.yam`，启动多个节点时修改成自己的ip地址 其它两个节点照抄，修改ip即可, image那行 应改为  `gcr.io/google_containers/etcd-amd64:3.1.11` ，实际就是版本号改一下即可。

```yaml
#需要修改所有含有ip的地方，下面的9，10，11，12行改为当前节点ip，15行三个ip顺序改为etcd集群部署的三台节点ip
version: '2.1'
services:
    etcd0:
        container_name: etcd_infra0
        image: gcr.io/google_containers/etcd-amd64:3.0.17  #这里最后改为3.1.11
        command: |
                etcd --name infra0
                --initial-advertisie-peer-urls http://10.230.204.160:2380
                --listen-peer-urls http://10.230.204.160:2380
                --listen-client-urls http://10.230.204.160:2379,http://127.0.0.1:2379
                --advertise-client-urls http://10.230.204.160:2379
                --data-dir /etcd-data.etcd
                --initial-cluster-token etcd-cluster-1
                --initial-cluster infra0=http://10.230.204.160:2380,infra1=http://10.230.204.165:2380,infra2=http://10.230.204.151:2380
                --initial-cluster-state new
        restart: always
        volumes:
           - /data/etcd-data.etcd:/etcd-data.etcd
        network_mode: "host"
```

三个节点分别启动：

```bash
$ docker-compose -f out/etcd/etcd-docker-compose-x.yml up -d  
#正常输出Creating etcd_infrax ... done  x为每个etcd编号
```

检查是不是安装成功:

```bash
$ docker exec etcd_infra0 etcdctl member list  #master1上可能运行报错容易提示容器正在重启。。原因暂时未知，其他master上可以
#成功应该是类似显示
5ded6dd284b89d31: name=infra1 peerURLs=http://10.230.204.153:2380 clientURLs=http://10.230.204.153:2379 isLeader=true
6d4b5eee32c1497a: name=infra0 peerURLs=http://10.230.204.150:2380 clientURLs=http://10.230.204.150:2379 isLeader=false
729d9cd56debb1a1: name=infra2 peerURLs=http://10.230.204.154:2380 clientURLs=http://10.230.204.154:2379 isLeader=false

#如果出现有peerURL不显示说明没有成功，尝试remove重新创建
$ docker-compose -f  out/etcd/etcd-docker-compose-x.yml down -v
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
$ kubeadm init --config out/kube/config
```
把成功后的kubeadm join命令存在文件里，那东西不能丢了

## 启动calico等

`mkdir ~/.kube && cp /etc/kubernetes/admin.conf ~/.kube/config ` （如果已经存在请校验一下是否相同,不确定建议删掉重新cp过去）

修改calico配置，把etcd地址换成你安装好的集群地址：
out/net/calico.yaml:
```
kind: ConfigMap
apiVersion: v1
metadata:
  name: calico-config
  namespace: kube-system
data:
  # The location of your etcd cluster.  This uses the Service clusterIP
  # defined below.
  etcd_endpoints: "http://10.96.232.136:6666" # 这里改成etcd集群地址如 "http://172.31.244.232:2379,http://172.31.244.233:2379,http://172.31.244.234:2379"
```

```bash
$ kubectl apply -f out/net/calico.yaml
$ kubectl apply -f out/heapster/influxdb
$ kubectl apply -f out/heapster/rbac
$ kubectl apply -f out/dashboard
#上面命令可整合为
$ kubectl apply -f out/net/calico.yaml -f out/heapster/influxdb -f out/heapster/rbac -f out/dashboard
```

1. 然后访问https://master1IP:32000端口即可，在chrome下无法进入提示证书有误，更换firefox可以，提示说证书日期不对（待修复）

## 启动多个master

第一个master我们称之为master0 (假设其他master已经init.sh过)，现在把第一个master的/etc/kubernetes/pki目录拷贝到别的master节点上

```bash
$ mkdir -p /etc/kubernetes
$ scp -r /etc/kubernetes/pki root@10.1.245.93:/etc/kubernetes/pki
```

删除pki目录下的apiserver.crt 和 apiserver.key文件`rm -rf apiserver.crt apiserver.key`，注意如果**不删除会只能看到一个master，是不正常的。**

同样使用master0上的out/kube/config文件，复制内容，拷贝到master1上，`scp out/kube/config  root@10.230.204.151:/root/` 执行`kubeadm init --config ~/config`

master2节点同master1

## 启动loadbalance

我比较推荐使用四层代理 HAproxy配置out/haproxy目录:

`vi out/haproxy/haproxy.cfg` 

```bash
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
  #下面三个ip替换成三个你自己master的地址
  server k8s-1 10.1.245.93:6443 check 
  server k8s-1 10.1.245.94:6443 check
  server k8s-2 10.1.245.95:6443 check
```

先` mkdir /etc/haproxy` 然后把这个文件拷贝在`cp out/haproxy/haproxy.cfg /etc/haproxy/haproxy.cfg`

```bash
$ docker run --net=host -v /etc/haproxy:/usr/local/etc/haproxy --name ha -d haproxy:1.7
```

## 修改kubeproxy配置

```bash
$ kubectl -n kube-system edit configmap kube-proxy
```

找到master地址，修改成LB地址，6444端口  （这里关键在于怎么知道LB的地址到底是哪一个呀？上面配置之后三个masterIP 轮询并不知道哪个是LB地址）

```yaml
#找到文件的这一块，第七行server 有个ip地址
apiVersion: v1
    kind: Config
    clusters:
    - cluster:
        certificate-authority: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        server: https://10.230.204.151:6443 #修改为 LoadBalanceIP:6444
      name: default
    contexts:
    - context:
        cluster: default
        namespace: default
        user: default
      name: default
    current-context: default
    users:
    - name: default
      user:
        tokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
```

## join node节点
还是在node节点执行第一个master输出的命令

```bash
$ kubeadm join --token <token> 10.1.245.94:6443 --discovery-token-ca-cert-hash sha256:<hash>
```

## 修改node节点kubelet配置

`vi /etc/kubernetes/kubelet.conf ` 同样把地址修改成LB地址,如：`10.1.245.94:6444` ，修改如下第五行（展示的例子已经修改过）

```yaml
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: xxxxxx #此处省略几百字符
    server: https://10.230.204.160:6444 #修改这里为LB:6444，原本是另外的ip:6443
  name: default-cluster
contexts:
- context:
    cluster: default-cluster
    namespace: default
    user: default-auth
  name: default-context
current-context: default-context
```

## kubectl配置

修改`~/.kube/config`文件,server的ip改成LB的ip `10.1.245.94:6444` 

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

