## 使用kubeadm部署k8s集群

### 基础环境

> 关闭swap

swapoff -a
再把/etc/fstab文件中带有swap的行删了,没有就无视

> 装这两工具如果没装的话

yum install -y ebtables socat

### 墙外安装 

### 离线安装

福利，我已经把所有依赖的镜像，二进制文件，配置文件都打成了包，解决您所有依赖,花了很多时间整理这个，放在了阿里云市场上，希望大家给点小支持

#### 安装kubelet服务，和kubeadm
> 下载bin文件 [地址](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.8.md#v181)

把下载好的kubelet kubectl kubeadm 直接拷贝到/usr/bin下面

> 配置kubelet systemd服务

```
cat <<EOF > /etc/systemd/system/kubelet.service
[Unit]
Description=kubelet: The Kubernetes Node Agent
Documentation=http://kubernetes.io/docs/

[Service]
ExecStart=/usr/bin/kubelet
Restart=always
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
```

```
cat <<EOF > /etc/systemd/system/kubelet.service.d/10-kubeadm.conf
[Service]
Environment="KUBELET_KUBECONFIG_ARGS=--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf --kubeconfig=/etc/kubernetes/kubelet.conf"
Environment="KUBELET_SYSTEM_PODS_ARGS=--pod-manifest-path=/etc/kubernetes/manifests --allow-privileged=true"
Environment="KUBELET_NETWORK_ARGS=--network-plugin=cni --cni-conf-dir=/etc/cni/net.d --cni-bin-dir=/opt/cni/bin"
Environment="KUBELET_DNS_ARGS=--cluster-dns=10.96.0.10 --cluster-domain=cluster.local"
Environment="KUBELET_AUTHZ_ARGS=--authorization-mode=Webhook --client-ca-file=/etc/kubernetes/pki/ca.crt"
Environment="KUBELET_CADVISOR_ARGS=--cadvisor-port=0"
Environment="KUBELET_CGROUP_ARGS=--cgroup-driver=cgroupfs"
Environment="KUBELET_CERTIFICATE_ARGS=--rotate-certificates=true --cert-dir=/var/lib/kubelet/pki"
ExecStart=
ExecStart=/usr/bin/kubelet $KUBELET_KUBECONFIG_ARGS $KUBELET_SYSTEM_PODS_ARGS $KUBELET_NETWORK_ARGS $KUBELET_DNS_ARGS $KUBELET_AUTHZ_ARGS $KUBELET_CADVISOR_ARGS $KUBELET_CGROUP_ARGS $KUBELET_CERTIFICATE_ARGS $KUBELET_EXTRA_ARGS
EOF
```
这里需要主意的是要看一下docker的cgroup driver与 --cgroup-driver要一致。 可以用 docker info |grep Cgroup 查看，有可能是systemd 或者 cgroupfs

> 增加主机名解析

为了防止无法解析主机名，修改/etc/hosts把主机名与ip的映射写上

#### 启动master节点
这里得把google的一票镜像想办法弄下来，然而我已经打成了一个[tar包](TODO)

```
kubeadm init --pod-network-cidr=192.168.0.0/16 --kubernetes-version v1.8.0 --skip-preflight-checks
```

* --pod-network-cidr 参数安装calico网络时需要
* --kubernetes-version 不加的话会去请求公网查询版本信息
* --skip-preflight-checks 解决一个kubelet目录不空的小bug

看到这些输出时你便成功了：
```
To start using your cluster, you need to run (as a regular user):

  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config

You should now deploy a pod network to the cluster.
Run "kubectl apply -f [podnetwork].yaml" with one of the options listed at:
  http://kubernetes.io/docs/admin/addons/

You can now join any number of machines by running the following on each node
as root:

  kubeadm join --token <token> <master-ip>:<master-port> --discovery-token-ca-cert-hash sha256:<hash>
```

照着执行：
```
  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

#### join node节点
同样到node节点安装kubelet和kubeadm，和master节点操作一样，不再赘述。
然后执行master节点init输出的那个命令：

```
  kubeadm join --token <token> <master-ip>:<master-port> --discovery-token-ca-cert-hash sha256:<hash>
```

执行完成后在master节点用kubectl验证节点是否健康

```
[root@dev-86-202 ~]# kubectl get nodes
NAME         STATUS     ROLES     AGE       VERSION
dev-86-202   NotReady   master    17h       v1.8.1
```
注意，master节点默认是不作为node的，也不推荐做node节点。 如果需要把master当node:
```
[root@dev-86-202 ~]# kubectl taint nodes --all node-role.kubernetes.io/master-
```

#### 安装calico网络
```
kubectl apply -f https://docs.projectcalico.org/v2.6/getting-started/kubernetes/installation/hosted/kubeadm/1.6/calico.yaml
```

#### 安装dashboard
安装dashboard不难，使用时还真有点绕，主要是RBAC, 先介绍个简单的
```
kubectl apply -f https://raw.githubusercontent.com/kubernetes/dashboard/master/src/deploy/alternative/kubernetes-dashboard.yaml
```
安装完之后, 使用nodeport方式访问

```
kubectl -n kube-system edit service kubernetes-dashboard
```
把type: ClusterIP 改成 type: NodePort 然后保存

```
$ kubectl -n kube-system get service kubernetes-dashboard
NAME                   CLUSTER-IP       EXTERNAL-IP   PORT(S)        AGE
kubernetes-dashboard   10.100.124.90   <nodes>       443:31707/TCP   21h
```
https://masterip:31707 就可以访问dashboard了， 然而 。。 还不能用。

创建一个 dashboard-admin.yaml
```
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: kubernetes-dashboard
  labels:
    k8s-app: kubernetes-dashboard
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: kubernetes-dashboard
  namespace: kube-system
```

kubectl create -f dashboard-admin.yaml

然后在界面上直接点skip就可以了，不过你懂的，这很不安全。  真正安全的做法 请关注我进一步讨论：https://github.com/fanux
