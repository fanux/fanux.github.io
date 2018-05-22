## 机器

IP           | 用途                        | 备注
------------ | ----------------------------|--------------------
10.100.81.11 | master、etcd                | 主节点
10.100.81.12 | master、etcd、keepalived、haproxy | 主节点，同时部署keepalived、haproxy，保证master高可用
10.100.81.13 | master、etcd、keepalived、haproxy | 主节点，同时部署keepalived、haproxy，保证master高可用
10.100.81.14 | node、etcd                       | 非业务节点
10.100.81.15 | node、etcd                  | 非业务节点
10.100.81.16 | node                        | 业务节点
10.100.81.17 | node                        | 业务节点
10.100.81.18 | node                        | 业务节点
10.100.81.19 | node                        | 业务节点
10.100.81.20 | node                        | 业务节点
10.100.81.21 | node                        | 业务节点 
10.100.81.22 | node                        | 业务节点
10.100.81.23 | node                        | 业务节点
10.100.81.24 | node、harbor                | 业务节点
10.100.81.25 | node                        | 业务节点

## 组件版本

组件名  | 版本
--------|--------
docker  |Docker version 1.12.6, build 78d1802
kubernetes|v1.10.0
harbor|v1.2.0
keepalived|v1.3.5
haproxy|1.7

## 配置

组件配置

### docker

配置文件：/usr/lib/systemd/system/docker.service

```
[Unit]
Description=Docker Application Container Engine
Documentation=https://docs.docker.com
After=network.target

[Service]
Type=notify
# the default is not to use systemd for cgroups because the delegate issues still
# exists and systemd currently does not support the cgroup feature set required
# for containers run by docker
ExecStart=/usr/bin/dockerd -H 0.0.0.0:2375 -H unix:///var/run/docker.sock --registry-mirror https://registry.docker-cn.com --insecure-registry 172.16.59.153 --insecure-registry hub.xfyun.cn --insecure-registry k8s.gcr.io --insecure-registry quay.io --default-ulimit core=0:0 --live-restore
ExecReload=/bin/kill -s HUP $MAINPID
# Having non-zero Limit*s causes performance problems due to accounting overhead
# in the kernel. We recommend using cgroups to do container-local accounting.
LimitNOFILE=infinity
LimitNPROC=infinity
LimitCORE=infinity
# Uncomment TasksMax if your systemd version supports it.
# Only systemd 226 and above support this version.
#TasksMax=infinity
TimeoutStartSec=0
# set delegate yes so that systemd does not reset the cgroups of docker containers
Delegate=yes
# kill only the docker process, not all processes in the cgroup
KillMode=process

MountFlags=slave

[Install]
WantedBy=multi-user.target
```

```
--registry-mirror：指定 docker pull 时使用的注册服务器镜像地址,指定为https://registry.docker-cn.com可以加快docker hub中的镜像拉取速度
--insecure-registry：配置非安全的docker镜像注册服务器
--default-ulimit：配置容器默认的ulimit选项
--live-restore：开启此选项，当dockerd服务出现问题时，容器照样运行，服务恢复后，容器也可以再被服务抓到并可管理
MountFlags=slave：解决移除容器时出现的"Unable to remove filesystem for $id: remove /var/lib/docker/containers/$id/shm: device or resource busy"问题
```

### kubernetes

#### etcd

以10.100.81.11节点为例，其它节点类似：

```
apiVersion: v1
kind: Pod
metadata:
  labels:
    component: etcd
    tier: control-plane
  name: etcd-10.100.81.11
  namespace: kube-system
spec:
  containers:
  - command:
    - etcd
    - --name=infra0
    - --initial-advertise-peer-urls=http://10.100.81.11:2380
    - --listen-peer-urls=http://10.100.81.11:2380
    - --listen-client-urls=http://10.100.81.11:2379,http://127.0.0.1:2379
    - --advertise-client-urls=http://10.100.81.11:2379
    - --data-dir=/var/lib/etcd
    - --initial-cluster-token=etcd-cluster-1
    - --initial-cluster=infra0=http://10.100.81.11:2380,infra1=http://10.100.81.12:2380,infra2=http://10.100.81.13:2380,infra3=http://10.100.81.14:2380,infra4=http://10.100.81.15:2380
    - --initial-cluster-state=new
    image: k8s.gcr.io/etcd-amd64:3.1.12
    livenessProbe:
      httpGet:
        host: 127.0.0.1
        path: /health
        port: 2379
        scheme: HTTP
      failureThreshold: 8
      initialDelaySeconds: 15
      timeoutSeconds: 15
    name: etcd
    volumeMounts:
    - name: etcd-data
      mountPath: /var/lib/etcd
  hostNetwork: true
  volumes:
  - hostPath:
      path: /var/lib/etcd
      type: DirectoryOrCreate
    name: etcd-data
```

### kubernetes系统组件

#### kubeadm init 启动k8s集群config.yaml配置

```
apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
networking:
  podSubnet: 192.168.0.0/16
api:
  advertiseAddress: 10.100.81.11
etcd:
  endpoints:
  - http://10.100.81.11:2379 
  - http://10.100.81.12:2379
  - http://10.100.81.13:2379
  - http://10.100.81.14:2379
  - http://10.100.81.15:2379

apiServerCertSANs:
  - 10.100.81.11
  - master01.bja.paas
  - 10.100.81.12
  - master02.bja.paas
  - 10.100.81.13
  - master03.bja.paas
  - 10.100.81.10
  
  - 127.0.0.1
token:
kubernetesVersion: v1.10.0
apiServerExtraArgs:
  endpoint-reconciler-type: lease
  bind-address: 10.100.81.11
  runtime-config: storage.k8s.io/v1alpha1=true
  admission-control: NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota
featureGates:
  CoreDNS: true
```

#### kubelet配置

/etc/systemd/system/kubelet.service.d/10-kubeadm.conf

```
[Service]
Environment="KUBELET_KUBECONFIG_ARGS=--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf --kubeconfig=/etc/kubernetes/kubelet.conf"
Environment="KUBELET_SYSTEM_PODS_ARGS=--pod-manifest-path=/etc/kubernetes/manifests --allow-privileged=true"
Environment="KUBELET_NETWORK_ARGS=--network-plugin=cni --cni-conf-dir=/etc/cni/net.d --cni-bin-dir=/opt/cni/bin"
Environment="KUBELET_DNS_ARGS=--cluster-dns=10.96.0.10 --cluster-domain=cluster.local"
Environment="KUBELET_AUTHZ_ARGS=--authorization-mode=Webhook --client-ca-file=/etc/kubernetes/pki/ca.crt"
Environment="KUBELET_CADVISOR_ARGS=--cadvisor-port=0"
Environment="KUBELET_CGROUP_ARGS=--cgroup-driver=cgroupfs"
Environment="KUBELET_CERTIFICATE_ARGS=--rotate-certificates=true --cert-dir=/var/lib/kubelet/pki --eviction-hard=memory.available<5%,nodefs.available<5%,imagefs.available<5%"
ExecStart=
ExecStart=/usr/bin/kubelet $KUBELET_KUBECONFIG_ARGS $KUBELET_SYSTEM_PODS_ARGS $KUBELET_NETWORK_ARGS $KUBELET_DNS_ARGS $KUBELET_AUTHZ_ARGS $KUBELET_CADVISOR_ARGS $KUBELET_CGROUP_ARGS $KUBELET_CERTIFICATE_ARGS $KUBELET_EXTRA_ARGS

```

### keepalived

keepalived采取直接在物理机部署，使用``` yum install keepalived ```安装。

启动配置文件：/etc/keepalived/keepalived.conf。keepalived的MASTER和BACKUP配置有部分差异


MASTER

```
! Configuration File for keepalived

global_defs {
   notification_email {
     root@localhost
   }
   router_id master02
}

vrrp_script chk_haproxy {
       script "/etc/keepalived/haproxy_check.sh"
       interval 3
       weight -20
}

vrrp_instance VI_1 {
    state MASTER    # BACKUP节点改成BACKUP
    interface bond1
    virtual_router_id 151
    priority 110    # BACKUP节点改成100
    advert_int 1
    authentication {
        auth_type PASS
        auth_pass 1111
    }
    virtual_ipaddress {
       10.100.81.10 # k8s使用的VIP
       10.100.81.9  # 数据库组件使用的VIP
    }
    track_script {
       chk_haproxy
    }
}

```

haproxy检查脚本：/etc/keepalived/haproxy_check.sh

```
#!/bin/bash

if [ `ps -C haproxy --no-header |wc -l` -eq 0 ] ; then
    docker restart k8s-haproxy
    sleep 2
    if [ `ps -C haproxy --no-header |wc -l` -eq 0 ] ; then
        service keepalived stop
    fi
fi
```

### haproxy

haproxy以容器的形式启动，启动命令如下：

```
docker run -d --net host --name k8s-haproxy -v /etc/haproxy:/usr/local/etc/haproxy:ro haproxy:1.7
```

haproxy配置文件：/etc/haproxy/haproxy.conf

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
  server k8s-1 10.100.81.11:6443 check
  server k8s-2 10.100.81.12:6443 check
  server k8s-3 10.100.81.13:6443 check
```

## 部署完成后操作

### 修改kube-proxy configmap

```
kubectl edit configmap kube-proxy -n kube-system
```

```
.....
kubeconfig.conf: |-
  apiVersion: v1
  kind: Config
  clusters:
  - cluster:
      certificate-authority: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
      server: https://10.100.81.10:6444  # 更改此行ip为vip,改成10.100.81.10
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
......
```

执行如下命令让kube-proxy组件重新启动

```
kubectl get pod -n kube-system | grep kube-proxy | awk '{print $1}' | xargs kubectl delete pod -n kube-system
```

### 修改所有node节点kubelet.conf

```
/etc/kubernetes/kubelet.conf
```

```
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN5RENDQWJDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRFNE1EVXhPREF4TXpNME1Gb1hEVEk0TURVeE5UQXhNek0wTUZvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTGJoCmw1TDRaNHFiWTJ3MmY5TFlEb0ZqVlhhcHRhYklkQmZmTS9zMTJaWFd1NU5LYWlPR09ub3RxK1gwM0VJb3Z4VEkKUGh5NzBqY294VGlLUTk5ZkFsUS82a2Vhc0x5MDNGZXJvYkhmaldUenBkZE5mWVNEZStMazlMV0hIZ0phOXVUQQpDU3kyay9sZGo3VWQ0Sk9pMi9lcGhVTUNNMUNlbmdPeWZDNUl0SUpFZzJmMk95cTE5U0JBeW1zYzFTalg5Q0F6CnNyMlhiTm9hK1lVS2Flek1QSldvYlNxdEg0czQ1TkluYytMREJFTkk4VGVITktybENsamdIeUorUjU1V2pCTW8KeSs3Y1BxL2cwTkxmSU4xRjJVbkFFa3RTSmVYUFBSaGlQUUhJcGRBU0xySXhVcE9HNlN3Yk51bmRGdGsxaUJiUgpUSW9md2UyT0VhZkhySmV5OHJrQ0F3RUFBYU1qTUNFd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFLME1mOFM5VjUyaG9VZ3JYcGQyK09rOHF2Ny8KR3hpWnRFelFISW9RdWRLLzJ2ZGJHbXdnem10V3hFNHRYRWQyUnlXTTZZR1VQSmNpMmszY1Z6QkpSaGcvWFB2UQppRVBpUDk5ZkdiM0kxd0QyanlURWVaZVd1ekdSRDk5ait3bStmcE9wQzB2ZU1LN3hzM1VURjRFOFlhWGcwNmdDCjBXTkFNdTRxQmZaSUlKSEVDVDhLUlB5TEN5Zlgvbm84Q25WTndaM3pCbGZaQmFONGZaOWw0UUdGMVd4dlc0OHkKYmpvRDhqUVJnL1kwYUVUMWMrSEhpWTNmNDF0dG9kMWJoSWR3c1NDNUhhRjJQSVAvZ2dCSnZ2Uzh2V1cwcVRDegpDV2EzcVJ0bVB0MHdtcEZic2RPWmdsWkl6aWduYTdaaDFWMDJVM0VFZ2kwYjNGZWR5OW5MRUZaMGJZbz0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    server: https://10.100.81.10:6444   # 此处改为VIP加haproxy监听端口6444
  name: default-cluster
contexts:
- context:
    cluster: default-cluster
    namespace: default
    user: default-auth
  name: default-context
current-context: default-context
kind: Config
preferences: {}
users:
- name: default-auth
  user:
    client-certificate: /var/lib/kubelet/pki/kubelet-client.crt
    client-key: /var/lib/kubelet/pki/kubelet-client.key
```

## 部署前注意事项

### 1. 确保所有节点时间同步
### 2. 确保所有节点ip转发功能打开

```
net.ipv4.ip_forward = 1
```
