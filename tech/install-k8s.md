## 使用kubeadm部署k8s集群

### 基础环境

> 关闭swap

swapoff -a
再把/etc/fstab文件中带有swap的行删了,没有就无视

> 装这两工具如果没装的话

yum install -y ebtables socat

### 墙外安装 

### 离线安装

福利，我已经把所有依赖的镜像，二进制文件，配置文件都打成了包，解决您所有依赖,花了很多时间整理这个，希望大家给点小支持

#### 安装kubelet服务，和kubeadm
> 下载bin文件 [地址](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.8.md#v181)

把下载好的kubelet kubectl kubeadm 直接拷贝到/usr/bin下面

#### 启动master节点

#### 安装calico网络

#### join node节点

#### 安装dashboard
