# 使用kubeadm安装安全高可用kubernetes集群

## 系统架构图
```
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
│   ├── kubeadm
│   ├── kubectl
│   └── kubelet
├── image        依赖的所有镜像包
│   └── images.tar
├── out          所有的配置文件
│   ├── dashboard  dashboard相关配置
│   │   ├── dashboard-admin.yaml
│   │   └── kubernetes-dashboard.yaml
│   ├── etcd  etcd相关配置
│   │   ├── etcd-docker-compose-0.yml
│   │   ├── etcd-docker-compose-1.yml
│   │   └── etcd-docker-compose-2.yml
│   ├── haproxy  haproxy相关配置
│   │   └── haproxy.cfg
│   ├── heapster   heapster相关yaml配置
│   │   ├── influxdb
│   │   │   ├── grafana.yaml
│   │   │   ├── heapster.yaml
│   │   │   └── influxdb.yaml
│   │   └── rbac
│   │       └── heapster-rbac.yaml
│   ├── kube    k8s自身配置
│   │   ├── 10-kubeadm.conf
│   │   ├── config    kubeadm配置
│   │   └── kubelet.service
│   ├── kubeinit.json  忽略
│   └── net  网络想着配置
│       ├── calico.yaml
│       └── calicoctl.yaml
└── shell    初始化脚本
    ├── init.sh   初始化节点,安装bin文件，systemd配置等
    └── master.sh  执行kubeadm init和其它组件
```

## 初始化节点
集群中所有节点都需要执行
cd shell && sh init.sh

## 起动etcd集群
在out/etcd目录下有相关模板，启动多个节点时修改成自己的ip地址
其它两个节点照抄，修改ip即可, 镜像替换成上面导入的，可用docker images查看一下.

使用docker-compose启动，如果没装：
```
$ pip install docker-compose
```
三个节点分别启动：
```
$ docker-compose -f etcd.yaml up -d
```

检查是不是安装成功:
```
$ docker exec etcd_infra0 etcdctl menber list
```

## kubeadm配置
out/kube/config 文件
```
apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
apiServerCertSANs:    此处填所有的masterip和lbip和其它你可能需要通过它访问apiserver的地址和域名或者主机名等，如阿里fip，证书中会允许这些ip
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
  endpoints:   这里填你上面安装的etcd集群地址列表
  - http://172.31.244.232:2379
  - http://172.31.244.233:2379
  - http://172.31.244.234:2379

apiServerExtraArgs:
  endpoint-reconciler-type: lease

networking:
  podSubnet: 192.168.0.0/16  不用改
kubernetesVersion: v1.9.2 不用改
featureGates: 不用改
   CoreDNS: true
```
执行：
```
$ kubeadm init --config config
```

把成功后的kubeadm join命令存在文件里，那东西不能丢了

## 启动calico等
mkdir ~/.kube
cp /etc/kubernetes/admin.conf ~/.kube/config
```
kubectl apply -f out/net/calico.yaml
kubectl apply -f out/heapster/influxdb
kubectl apply -f out/heapster/rbac
kubectl apply -f out/dashboard
```
然后访问https://master0ip:32000端口即可

## 启动多个master
第一个master我们称之为master0, 现在在master1上同样拷贝压缩包执行 `cd shell && sh init.sh`

别的master节点初始化好之后，把第一个master的/etc/kubernetes/pki目录拷贝到别的master节点上

```
$ scp -r root@10.1.245.93:/etc/kubernetes/pki /etc/kubernetes
```

删除pki目录下的apiserver.crt apiserver.key文件，注意如果不删除会只能看到一个master，是不正常的。

同样使用master0上的out/kube/config文件，复制内容，拷贝到master1上，执行`kubeadm init --config config`

master2节点同master1

## 启动loadbalance

我比较推荐使用四层代理
HAproxy配置out/haproxy目录:

cat out/haproxy/haproxy.cfg
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
把这个文件拷贝在cp out/haproxy/haproxy.cfg /etc/haproxy/haproxy.cfg
```
docker run --net=host -v /etc/haproxy:/usr/local/etc/haproxy --name ha -d haproxy:1.7
```

## 修改kubeproxy配置
```
kubectl -n kube-system edit configmap kube-proxy
```
找到master地址，修改成LB地址。6444端口

## join node节点
还是在node节点执行第一个master输出的命令，就是上面haproxy的地址   
```
$ kubeadm join --token <token> 10.1.245.94:6443 --discovery-token-ca-cert-hash sha256:<hash>
```

## 修改node节点kubelet配置
vim /etc/kubernetes/kubelet.conf 同样把地址修改成LB地址,如：10.1.245.94:6444

## kubectl配置
修改~/.kube/config文件,ip改成LB的ip 10.1.245.94:6444

或者通过命令修改：
```
$ kubectl config set-cluster kubernetes --server=https://47.52.227.242:6444 --kubeconfig=$HOME/.kube/config
```
### 启动多DNS副本
```
kubectl edit deploy coredns -n kube-system
```
replicas: 3

```
[root@master1 ~]# kubectl get pod -n kube-system -o wide|grep core
coredns-65dcdb4cf-4j5s8                  1/1       Running   0          39m       192.168.137.65    master1
coredns-65dcdb4cf-ngx4h                  1/1       Running   0          38s       192.168.180.1     master2
coredns-65dcdb4cf-qbsr6                  1/1       Running   0          38s       192.168.166.132   node1
```
这样，启动了三个dns

### 验证
```
kubectl run test --image busybox sleep 10000
kubectl exec your-busybox-pod-name nslookup kubernetes
```
杀非LB的master，多次测试看创建pod与dns是否正常，还可以telnet 10.96.0.1 443 去验证clusterip是否正常
