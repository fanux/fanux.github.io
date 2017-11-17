## 借助kubeadm构建企业级高可用k8s集群

### 构建etcd集群
```
version: '2'
services:
    etcd:
        container_name: etcd_infra0
        image: 172.16.59.153/develop/etcd:2.3.1
        command: |
                etcd --name infra0
                --initial-advertise-peer-urls http://172.16.59.151:2380
                --listen-peer-urls http://172.16.59.151:2380
                --listen-client-urls http://172.16.59.151:2379,http://127.0.0.1:2379
                --advertise-client-urls http://172.16.59.151:2379
                --data-dir /etcd-data.etcd
                --initial-cluster-token etcd-cluster-1
                --initial-cluster infra0=http://172.16.59.151:2380,infra1=http://172.16.59.152:2380,infra2=http://172.16.59.153:2380
                --initial-cluster-state new
        volumes:
           - /data/etcd-data.etcd:/etcd-data.etcd
        network_mode: "host"
```

### 配置apiserver使用外部etcd集群
```
apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
#api:
#  advertiseAddress: <address|string>
#  bindPort: <int>
etcd:
  endpoints:
  - http://10.1.245.94:2379
  #caFile: <path|string>
  #certFile: <path|string>
  #keyFile: <path|string>
networking:
  #dnsDomain: <string>
  #serviceSubnet: <cidr>
  podSubnet: 192.168.0.0/16
kubernetesVersion: v1.8.2
# cloudProvider: <string>
# nodeName: <string>
# authorizationModes:
# - <authorizationMode1|string>
# - <authorizationMode2|string>
# token: <string>
# tokenTTL: <time duration>
# selfHosted: <bool>
apiServerExtraArgs:
 etcd-servers: http://10.1.245.94:2379
# controllerManagerExtraArgs:
#   <argument>: <value|string>
#   <argument>: <value|string>
# schedulerExtraArgs:
#   <argument>: <value|string>
#   <argument>: <value|string>
# apiServerCertSANs:
# - <name1|string>
# - <name2|string>
# certificatesDir: <string>
# imageRepository: <string>
# unifiedControlPlaneImage: <string>
# featureGates:
#   <feature>: <bool>
#  <feature>: <bool>
```
kubeadm init --config config.yaml

### 配置calico网络使用外部etcd集群

### 启动多个apiserver manager sheduler

### 负载均衡apiserver

### kubeconfig文件配置负载均衡器地址
```
kubectl config set-cluster kubernetes --server=https://47.52.227.242:6443 --kubeconfig=$HOME/.kube/config
```

### 配置认证信息
```
# 设置集群参数
export KUBE_APISERVER="https://172.20.0.113:6443"
kubectl config set-cluster kubernetes \
--certificate-authority=/etc/kubernetes/ssl/ca.pem \
--embed-certs=true \
--server=${KUBE_APISERVER} \
--kubeconfig=devuser.kubeconfig

# 设置客户端认证参数
kubectl config set-credentials devuser \
--client-certificate=/etc/kubernetes/ssl/devuser.pem \
--client-key=/etc/kubernetes/ssl/devuser-key.pem \
--embed-certs=true \
--kubeconfig=devuser.kubeconfig

# 设置上下文参数
kubectl config set-context kubernetes \
--cluster=kubernetes \
--user=devuser \
--namespace=dev \
--kubeconfig=devuser.kubeconfig

# 设置默认上下文
kubectl config use-context kubernetes --kubeconfig=devuser.kubeconfig
```
```
kubectl config get-contexts
CURRENT   NAME              CLUSTER           AUTHINFO        NAMESPACE
*         kubernetes        kubernetes        admin
          default-context   default-cluster   default-admin
```
