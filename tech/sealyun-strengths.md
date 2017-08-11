# 关于Docker engine
* sealyun容器引擎基于moby构建，青出于蓝,兼容Docker所有操作，新增一些非常有用的功能
* 原生支持更牛逼的网络模式
    * docker的bridge网络模式是不能跨主机通信的，而我们通过定制实现了linux bridge的跨主机通信
    * 增加了网络的ovs实现，容器可以划分vlan, 或者使用vxlan GRE等实现跨主机通信，且能做到网络租户隔离，避免产生ARP风暴
* 更好的支持runc，如hook的功能，在容器异常结束时执行回调脚本，这在有些场景下非常有用，如针对业务异常退出作些清理，而不用监听docker event
* 支持挂在磁盘时的反向覆盖，即容器里面的文件覆盖宿主机的，这也非常有用，比如mysql里已经有配置文件了，但是宿主机上没有，可能想把配置挂载出来，但是一般启动加 -v参数时外面空目录会覆盖里面的，这不是我们希望的，所以sealyun容器引擎支持这样的挂载： -v /etc/mysql.cnf<-/etc/mysql.cnf 里面覆盖外面 -v /etc/mysql.cnf->/etc/mysql.cnf 外面覆盖里面，当然也支持正常的docker 挂载
* 稳定性更高，docker devicemapper在磁盘占满时 engine会defunt，甚至会导致重装docker engine也无法解决的问题，这是loop-lvm造成的，但是derict-lvm配置又麻烦，所以大家想用overlay存储驱动，不过在内核3.10或者更低版本是的问题的，比如在centos7.2上跑FROM ubuntu的镜像就会出错，所以我们对存储驱动进行了优化, sealyun container会跑的更稳定，如果大规模部署时会发现经常产生僵尸容器，我们的容器僵尸率比开源容器低的多
    
# 关于Swarm
# 管理UI对比
* 开源管理容器的UI都有bug，以shipyard为例，首先在scale up时没加锁，会导致节点重复分配，与swarm的兼容性就会出问题，然后容器数量一多大概300个时会很慢，然后界面上会出无故的错误。
* 开源工具分页支持差，不支持compose文件, 我们完全支持
* 我们的UI使用vue.js完全重写,单页面渲染上千个容器毫无压力
* 我们UI更简洁，功能更细致，如删除容器时其实是有三个参数是可以指定的，是否删除link,volume是否强制删除, 停止容器时支持stop timeout参数，这些在真正生产环境中都是非常必要的。
* 此外我们还在UI上支持了多集群管理与快速创建集群等功能

# CI CD系统对比
