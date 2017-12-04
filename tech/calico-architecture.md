## calico架构分析

### 组件
* Felix calico每个节点上跑的代理
* Orchestrator plugin网络编排插件
* etcd 存储配置数据
* BIRD BGP客户端，分发路由信息
* BGP Route Reflector(BIRD) 另一个可选方案，适合更大规模

### Felix
每个节点上的一个守护进程，负责编写路由和ACLs（访问控制列表）. 还有一些其它节点上需要设置的东西。
主要包含：

 > 网络接口管理

把接口的一些信息告诉内核，让内核正确的处理这个接口的链路，特殊情况下，会去响应ARP请求，允许ip forwarding有等。
接口发现，注销的功能等

> 路由管理

在节点上把endpoints的路由配置到Linux kernel FIB(forwarding information base), 保障包正确的到达节点的endpoint上

我的理解endpoints是节点上的虚拟网卡

> ACL管理 准入控制列表

设置内核的ACL,保证只有合法的包才可以在链路上发送,保障安全。  

> 状态报告

把节点的网络状态信息写入etcd。

### 编排插件 orchestrator Plugin
需要和别的编排调度平台结合时的插件，如Calico Neutron ML2 mechanism driver. 这样就可以把calico当成neutron的网络实现。

> API 转化

编排系统 kubernetes openstack等有自己的API，编排插件翻译成calico的数据模型存到calico的数据库中。

> 反馈

把网络状态的一些信息反馈给上层的编排调度系统

### etcd
两个主要功能，存储数据与各组建之间的通信。

根据编排系统的不同，etcd可能是个主存储或者是个镜像存储，在openstack中就是一个镜像存储

### BGP Client(BIRD)
读取Felix设置的内核路由状态，在数据中心分发状态。

### BGP Route Reflector (BIRD)
大型部署，简单的BGP会有限制，每个BGP客户端之间都会相互连接，会以 N^2次方
增长。拓扑也会变的复杂

reflector负责client之间的连接，防止它们需要两两相连。

为了冗余，可以部署多个reflectors, 它仅仅包含控制面，endpoint之间的数据不经过它们

路由广播
