## 借助kubeadm构建企业级高可用k8s集群

### 构建etcd集群

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
