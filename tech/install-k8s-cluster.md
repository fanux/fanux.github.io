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

## 起动etcd集群
cat etcd.yaml
```
version: '2'
services:
    etcd:
        container_name: etcd_infra0
        image: quay.io/coreos/etcd:v3.1.10
        command: |
                etcd --name infra0
                --initial-advertise-peer-urls http://10.1.245.94:2380
                --listen-peer-urls http://10.1.245.94:2380
                --listen-client-urls http://10.1.245.94:2379,http://127.0.0.1:2379
                --advertise-client-urls http://10.1.245.94:2379
                --data-dir /etcd-data.etcd
                --initial-cluster-token etcd-cluster-1
                -initial-cluster infra0=http://10.1.245.93:2380,infra1=http://10.1.245.94:2379,infra2=http://10.1.245.95:2379
                --initial-cluster-state new
        volumes:
           - /data/etcd-data.etcd:/etcd-data.etcd
        network_mode: "host"
```
其它两个节点照抄，修改ip即可

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
config.yaml
```
apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
apiServerCertSANs:
- 10.1.245.93
- 10.1.245.94
- 10.1.245.95
- 47.52.227.242
etcd:
  endpoints:
  - http://10.1.245.94:2379
networking:
  podSubnet: 192.168.0.0/16
kubernetesVersion: v1.8.2
```
注意版本号
apiServerCertSANs与证书配置有关，把你所有master的ip和lb的ip都写进去，或者你允许的域名等
```
$ kubeadm init --config config.yaml
```

## 启动多个master
> 别的master节点初始化好之后，把第一个master的/etc/kubernetes目录拷贝到别的master节点上

```
$ scp -r root@10.1.245.93:/etc/kubernetes /etc
```

> 修改该目录下各conf的ip，改成本机ip, 如下命令搜出来的都要改

```
grep 245.93 . -rn
```

> 启动kubelet

```
systemctl start kubelet
```

## 启动loadbalance

我比较推荐使用四层代理
HAproxy配置:
cat /root/haproxy/haproxy.cfg
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
  server k8s-1 10.1.245.93:6443 check
  server k8s-1 10.1.245.94:6443 check
  server k8s-2 10.1.245.95:6443 check
```
```
docker run --net=host -v /root/haproxy:/usr/local/etc/haproxy --name ha -d haproxy:1.7
```

## join node节点
还是在node节点执行第一个master输出的命令，不过IP换成LB的ip地址，就是上面haproxy的地址  如 
```
$ kubeadm join --token <token> 10.1.245.94:6444 --discovery-token-ca-cert-hash sha256:<hash>
```

## kubectl配置
修改~/.kube/config文件,ip改成LB的ip 10.1.245.94:6444

或者通过命令修改：
```
$ kubectl config set-cluster kubernetes --server=https://47.52.227.242:6443 --kubeconfig=$HOME/.kube/config
```

# 问题
~~~上述方式这样安装完是有问题的，用kubectl get node 只能看到一个master，虽然任意一个master挂了kubectl可以正常访问集群，但是dns什么的是无法切换到别的节点上的。
要想看到三个master，必须到三个master上都执行kubeinit,把ca.crt ca.key拷贝到对应机器，要注意一定要使用相同根证书，不然会出证书错误。~~~

应该把证书都拷贝过去，只删除apiserver.crt 和apiserver.key


==================================华丽分割线===================================

### 启动第一个master
```
apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
apiServerCertSANs:
- 172.31.244.235
- 172.31.244.236
- 172.31.244.237
- 172.31.244.238
- master1
- master2
- master3
- node1
- 47.75.6.242

etcd:
  endpoints:
  - http://172.31.244.235:2379

apiServerExtraArgs:
  endpoint-reconciler-type: lease

networking:
  podSubnet: 192.168.0.0/16
kubernetesVersion: v1.9.1
featureGates:
   CoreDNS: true
```
### 创建网络
kubectl apply -f calico.yaml

### join node节点
略

### 启动别的master
cp /etc/kubernetes/pki 到其它master节点相同目录, 其它两节点删除 apiserver.crt apiserver.key
不删的话启动完了你只能看到一个master。 然后和master1一样去启动.

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

### 配置kubelet 与kubeproxy

### DNS破坏性测试
### 网络破坏性测试
### master节点破坏性测试

--------------------------再分割-------------------------
# 安装etcd
# 安装master0
# 安装calico,替换etcd
# 拷贝pki，貌似不需要删啥
# 启动别的apiserver
# 启动负载均衡器
# 修改kubelet配置
# 修改kubeproxy配置
# 启动coreDNS副本


# 启动三个busybox验证, 验证dns,创建pod和telnet 10.96.0.1 443
# 删掉一个master
# 再删掉一个master
# 恢复一个master
# 再删掉最后一个master 
