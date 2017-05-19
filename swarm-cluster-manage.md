## 原理解析
考虑一下在多个节点上使用docker的场景：

没有swarm时(需要到每个节点上去运行docker)：
```
  docker   docker    docker
  run      run       run
   |        |         |
   V        V         V
  +------+ +------+ +------+ 
  | node | | node | | node |
  +------+ +------+ +------+
```
有了swarm(讲请求转发给对应主机上的docker engine):
```
  docker  +-------+
  run---->| swarm |
          +-------+
              |
     +--------+--------+
     |        |        |
     V        V        V
  +------+ +------+ +------+ 
  | node | | node | | node |
  +------+ +------+ +------+
```

swarm分两个组件，manager和join，swarm join逻辑非常简单，干的事也非常单一，就是把自己节点的engine地址注册到服务发现中，并维持一个心跳。
所有容器的操作都是由swarm manager直接发给对应节点上的docker engine的。
```
         +------+   +--------+  +-------+          +-----------------+ 
         | node |   | node   |  | node  |  <====>  | swarm  manager  |
         +------+   +--------+  +-------+          +-----------------+ 
            |          |          |                        |          
            |      +------------------+                    |         
            |______|  etcd cluster    |____________________|        
                   +------------------+                 
```
## swarm 安装
### 安装etcd

找三个节点，到对应节点上修改好docker-compose文件
compose 文件(不知道什么是compose文件的请看compose.md):
```
[root@yjybj-3-031 docker-compose]# cat docker-compose-etcd.yml 
version: '2'
services:
    etcd:
        container_name: etcd_infra2                             #容器名，三个节点分别为etcd_infra[0|1|2]
        image: reg.iflytek.com/release/etcd:2.3.1               #镜像名 
        command: |
                etcd --name infra2                              # etcd节点名称 分别改成 infra[0|1|2]
                --initial-advertise-peer-urls http://172.27.3.31:2380   # 改成节点的ip
                --listen-peer-urls http://172.27.3.31:2380              # 改成节点ip
                --listen-client-urls http://172.27.3.31:2379,http://127.0.0.1:2379 # 改成节点ip
                --advertise-client-urls http://172.27.3.31:2379 # 改成节点ip
                --data-dir /etcd-data.etcd
                --initial-cluster-token etcd-cluster-1      # 三个节点一样，不需要修改
                --initial-cluster infra0=http://172.27.0.13:2380,infra1=http://172.27.3.30:2380,infra2=http://172.27.3.31:2380 # 三个借点ip, 三个节点都一样，不需修改
                --initial-cluster-state new
        volumes:
           - /data/etcd-data.etcd:/etcd-data.etcd 
        network_mode: "host"
```
执行 `docker-compose -f docker-compose-etcd.yml up -d`
三个节点都启动成功后，检查etcd集群是否正常：
三个节点中找一个节点运行：
```
$ docker run --rm --net=host reg.iflytek.com/release/etcd:2.3.1 etcdctl cluster-health
```
### 安装swarm manager
寻找一个节点作为swarm的管理节点，compose文件如下：
```
[root@yjybj-3-031 docker-compose]# cat docker-compose-swarm-manage.yml 
version: '2'
services: 
        swarm_manage:
                container_name: swarm_manage_3_31
                image: reg.iflytek.com/release/swarm:latest
                network_mode: "host"
                command: manage -H tcp://172.27.3.31:4000 etcd://172.27.0.13:2379,172.27.0.15:2379,172.27.3.31:2379
```
* -H 指定自己监听的ip和端口,后面是etcd的地址
同样执行`docker-compose -f docker-compose-swarm-manage.yml up -d`

所有节点配置主机名解析manager节点主机名为：swarm.iflytek.com

### 安装swarm join
在集群中每个节点都执行swarm join:
```
[root@yjybj-3-031 docker-compose]# cat docker-compose-swarm-join.yml 
version: '2'
services: 
        swarm_join:
                container_name: swarm_join_3_031
                image: reg.iflytek.com/release/swarm:latest
                network_mode: "host"
                command: join --advertise=172.27.3.31:2375 etcd://172.27.0.13:2379,172.27.3.30:2379,172.27.3.31:2379
```
* --advertise 指定节点自己的ip和端口
每个节点上都执行：
```
$ docker-compose -f docker-compose-swarm-join.yml up -d
```
### 安装Dface
依次执行两个compose文件：
```
[root@yjybj-3-031 dface]# cat docker-compose-dependson.yml  #啥都不用改
version: '2'
services:
    db:
     container_name: pbrain-db
     image: reg.iflytek.com/release/postgres:latest
     environment:
       POSTGRES_USER: shipyard 
       POSTGRES_DB: shipyard 
       POSTGRES_PASSWORD: 111111 
     network_mode: "host"
     command: postgres
    mq:
       container_name: pbrain-mq
       image: reg.iflytek.com/release/gnatsd:latest
       command: gnatsd
       network_mode: "host"
    rethinkdb:
      container_name: shipyard-rethinkdb
      command: rethinkdb  
      network_mode: "host"
      image: reg.iflytek.com/release/rethinkdb:latest

[root@yjybj-3-031 dface]# cat docker-compose.yml 
version: '2'
services:
    pbrain:
      container_name: pbrain
      command: pbrain manager -o http://172.26.3.31:8888 --docker-host tcp://swarm.iflytek.com:4000 # -o 支持dface跨域请求，写dface地址
      network_mode: "host"
      image: reg.iflytek.com/release/pbrain:latest

    shipyard:
      container_name: dface     # 不用改
      command: controller server --listen :8888 -d tcp://swarm.iflytek.com:4000 --rethinkdb-addr localhost:28015  --rethinkdb-database "dface" 
      image: reg.iflytek.com/release/dface:latest
      network_mode: "host"
```
先`docker-compose -f docker-compose-dependon.yml up -d` 启动成功后执行：`docker-compose -f docker-compose.yml up -d`
然后可以再浏览器中访问dface了。8888端口，默认用户名密码：admin/shipyard

## 使用教程
swarm兼容了docker api, 意味着怎么使用docker就怎么使用swarm, 没有额外的学习成本。 假设swarm manager地址为 tcp://localhost:4000
```
$ docker run -H tcp://localhost:4000 (一切docker 命令)
```

### 过滤器
```
                           +---> Constraint:
                           |          给docker engine打标签：docker deamon --label storage=ssd
                           |          启动容器并调度到ssd节点上  docker -H tcp://swarm.iflytek.com:4000 -e constraint:storage==ssd nginx
        +---node filter----+
        |                  |                                                                 
        |                  |                                                                 
        |                  +---> containerslots:
        |                              节点上最多运行三个容器：docker daemon --label containerslots=3                              
        |                              部署nginx时每个节点最多部署两个: docker -H tcp://swarm.iflytek.com:4000 run -e containerslots=2 nginx                            
        |                                                                                    
filter--
        |                                                                                    
        |                      +----> affinity:                                                             
        |                      |           docker -H tcp://swarm.iflytek.com:4000 run -l foo=bar --name mysql mysql:latest 
        |                      |        麻烦调度nginx容器到一个运行有名字叫mysql容器的节点上：
        |                      |           docker -H tcp://swarm.iflytek.com:4000 run -e affinity:container==mysql nginx:latest 
        |                      |        请调度nginx到一个贴有foo=bar标签的容器运行的节点上去
        |                      |           docker -H tcp://swarm.iflytek.com:4000 run -e affinity:foo==bar nginx:latest 
        |                      |        请寻找到一个节点有nginx:latest镜像的节点上
        |                      |           docker -H tcp://swarm.iflytek.com:4000 run -e affinity:image==nginx:latest nginx:latest 
        |                      |        找一个有nginx:latest镜像的节点运行，如果找不到就随便找个节点运行（==~）约等于
        |                      |           docker -H tcp://swarm.iflytek.com:4000 run -e affinity:image==~nginx:latest nginx:latest 
        +---container filter---+                                                                       
                               |---> port 淘汰被占用端口号的节点
                               |
                               +---> dependency  依赖某些容器，卷，或者网络
```

### dface界面操作
可以在界面上进行容器的启停/删除等操作。查看日志，进入容器。
查看集群中镜像，删除镜像等，比如镜像升级想清除集群中的缓存镜像可以在界面上操作。
