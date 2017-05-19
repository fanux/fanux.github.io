## 基本概念
* 镜像(image) - 一堆文件和目录的集合，如centos镜像里面有/usr /lib /bin等目录。可类比成操作系统镜像。
* 容器(container) - 用过虚拟机的都知道如一个centos虚拟机镜像可以创建多个虚拟机，容器就可类比成虚拟机。很多资料上说容器是“运行着的镜像”其实不确切，容易产生误导，因为容器可以停止，停止了的容器就和关机的虚拟机一样也是存在的。

```
[root@docker rootfs]# pwd
/var/lib/docker/devicemapper/mnt/4c179e5d58d5f0fd5181618c3a7f0a87f47473bbfffdd18e515c74a40caf2be1/rootfs
[root@docker rootfs]# ls
bin  boot  dev  etc  home  lib  lib64  media  mnt  opt  proc  root  run  sbin  srv  sys  tmp  usr  var
```

## 常用命令
![状态图](http://192.168.86.170:10080/iflytek/docs/raw/master/images/status.png)
```
$ docker run|create $(image)             创建容器，容器从无到有
$ docker ps                              查看正在运行的容器
$ docker ps -a                           查看所有的容器，包含退出状态的容器
$ docker start|stop|rm $(container)      启动|停止|删除 已有的容器
$ docker images                          查询镜像
$ docker rmi $(image)                    删除镜像，注意若有容器使用到了该镜像就无法删除，需要先删除容器
$ docker tag $(image) $(new image)       给镜像取个别名，tag出来的镜像与源镜像id相同，所以删除tag时不能使用id，要使用镜像名
$ docker pull|push $(image)              从仓库拉取镜像和推送镜像 
$ docker exec $(container) $(command)    如在容器中执行bash命令   docker exec -it $(container) /bin/bash
$ docker logs $(container)               查看容器日志（容器中程序输出到标准输出和标准错误的日志）
$ docker inspect $(container)            查看容器详情
$ docker info                            查看docker engine详情
$ docker save $(images) > $(image).tar   导出镜像
$ docker load -i $(image).tar            导入镜像 
```

## 网络模式(入门教程，不作深入解析)
五种网络模式：
1. 端口映射：
```
$ docker run -p 8080:80 nginx:latest
```
如果没有这个-p，会发现启动了nginx但是无法通过宿主机访问到web服务，而使用了-p参数后就可以通过访问主机的8080断开去访问nginx了。
端口映射的原理是作了net转发

2. 共享主机网络:
```
$ docker run --net=host nginx:latest
```
这种容器没有自己的网络，完全共享主机的网络，所以可以通过主机ip直接访问容器服务。 坏处是容器与其它容器端口冲突

3. link网络
```
$ docker run --name mysql mysql:latest
$ docker run --link=mysql nginx:latest
```
这样nginx可以通过容器名去访问mysql，其原理是在nginx容器中的/etc/hosts中加入了mysql主机名解析。这种共享不可跨主机

```
$ docker run --rm -it --name c1 centos:latest /bin/bash
```
```
$ docker run --rm -it --name c2 --link c1  centos:latest /bin/bash
[root@178d290d873c /]# cat /etc/hosts
127.0.0.1    localhost
::1    localhost ip6-localhost ip6-loopback
fe00::0    ip6-localnet
ff00::0    ip6-mcastprefix
ff02::1    ip6-allnodes
ff02::2    ip6-allrouters
172.17.0.4    c1 3b7b15fa7e20   # 看这里
172.17.0.5    178d290d873c
```

4. none模式
容器不创建网络，需要自行定义

5. overlay网络
进群中常用的网络模式，使用vxlan等技术作了一层覆盖，使每个容器有自己独立的ip并可跨主机通信。

6. 共享容器网络

如kubernetes里pod的实现，pod是多个容器的集合，这些容器都共享了同一个容器的网络，那么这些容器就如同在一个host上一样。

## 磁盘挂载
使用容器有一点需要注意，就是容器的生命周期可能很短，所以不要在容器中持久化数据。  比如，升级时使用新的镜像那么之前老版本的容器肯定会被删除。

还有就是可能有配置文件或者目录需要修改，进到容器中可能不方便，而且我们希望容器没有了配置文件还在的场景。所以需要挂载宿主机的磁盘。

```
$ docker run -v /data/data.sqlite:/data.sqlite sqlite:latest
```
注意：
* run时宿主机目录会覆盖容器里面的目录/文件
* 容器里写文件时会写在宿主机上，同理宿主机写容器内也一样（其实本质是同一个文件）
* 除非不得已，尽量避免使用配置文件和挂载配置文件。  推荐的方式是使用confd管理配置文件，或者使用命令行/环境变量给应用传参

## Dockerfile
### Why Dockerfile
docker镜像有几种构建方式，比如进入到容器中，安装好我们需要的东西，启动工程，然后基于这个容器导出镜像。

这样做不方便的地方是如果我们修改了代码或者配置什么的，就需要把重复的工作再来一遍，麻烦。

dockerfile指定了镜像构建的规则，这样就避免了手动构建的麻烦。

此外dockerfile的好处就是我们可以看到镜像构建的过程，镜像有什么问题可以找出来。

### 简单事例
pbrain是go语言的集群调度插件系统，我是在本地（开发环境）开发，在容器中运行的。

> 目录结构

```
▾ pbrain/
  ▸ cmd/
  ▸ common/
  ▸ doc/
  ▸ Godeps/
  ▸ manager/
  ▸ plugins/
  ▸ script/
  ▸ vendor/
    Dockerfile
    LICENSE
    main.go
    README.md
```
在工程主目录下创建一个Dockerfile文件

> Dockerfile

```bash
FROM golang:latest                            # 选择一个go的基础镜像，这样编译运行的环境就有了

COPY . /go/src/github.com/fanux/pbrain/       # 把代码拷贝到容器中

RUN go get github.com/tools/godep && \        # 在容器中编译安装
    cd /go/src/github.com/fanux/pbrain/ && \
    godep go install

CMD pbrain --help                             # 容器启动时执行的命令
```
Dockerfile非常的简单, 需要注意的地方有：

1. java 这种build once run any where的就可以不用拷源码到容器中了，直接拷贝可执行jar包，配置，执行脚本什么的，也就是运行需要的依赖。

2. 推荐把配置文件或者配置目录也打入镜像中，除非容器在不同的机器上运行时配置不同，如果是那样的话，可在宿主机上配置然后运行时作磁盘映射，尽量不要这样做。

3. RUN 命令可以在容器中执行一条linux命令。

4. CMD 命令可以在启动容器时被覆盖，如 我们启动容器
```bash
$ docker run pbrain:latest pbrain manager
```
这样`pbran manager` 这条命令就会覆盖镜像中的 `pbrain --help` 命令

> build镜像

在有Dockerfile的目录下执行：
```bash
$ docker build -t pbrain:latest .
$ docker images
```
（不要忘记后面的一点）
这样使用docker images命令就可以看到新的镜像 `pbrain:latest`

> 其它Dockerfile命令

其它命令如需要使用或者有不明白的地方可联系我

* MAINTAINER 用来指定镜像创建者信息
* ENTRYPOINT 设置container启动时执行的操作, 和CMD很像，但是不会被覆盖
* USER 设置容器启动时用户，默认是root用户。
* EXPOSE 暴露端口，我们共享主机网络，不用这个
* ENV 设置环境变量，这个可能需要用到, 当然也可以在容器运行时指定环境变量
* ADD 和COPY很像，就用COPY就可以了
* VOLUME 指定挂载点 挂载本机的目录，配置文件尽量不要挂载，数据输出可以，可在运行时指定
* WORKDIR 切换目录 切换工作目录，这个比较有用, 为了方便可以把运行时文件拷贝到此目录

### 提交镜像到镜像仓库
假设我们仓库地址是：`192.168.86.106:5000`

我们需要提交的镜像是`pbrain:latest`
```bash
$ docker tag pbrain:latest 192.168.86.106:5000/pbrain:latest
$ docker push 192.168.86.106:5000/pbrain:latest
```

注意事项：
* 构建镜像推荐使用dockerfile，虽然可以用容器通过commit的方式创建一个镜像，但是这种镜像不能显式表示镜像构建过程。而且如果需要多次构建和修改镜像都非常不方便。
* CMD中的命令不能在后台运行，如果在后台运行会导致容器立马退出。  有时不方便修改成在前台运行可使用shell脚本启动，使用wait命令等待进程退出

## docker1.13以上版本内建了compose
```
╰─➤  docker stack deploy --help

Usage:  docker stack deploy [OPTIONS] STACK

Deploy a new stack or update an existing stack

Aliases:
  deploy, up

Options:
      --bundle-file string    Path to a Distributed Application Bundle file
  -c, --compose-file string   Path to a Compose file
      --help                  Print usage
      --with-registry-auth    Send registry authentication details to Swarm agents
```
