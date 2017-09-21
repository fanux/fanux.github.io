# CI 概述
### 用一个可描述的配置定义整个工作流

程序员是很懒的动物，所以想各种办法解决重复劳动的问题，如果你的工作流中还在重复一些事，那么可能就得想想如何优化了

持续集成就是可以帮助我们解决重复的代码构建，自动化测试，发布等重复劳动，通过简单一个提交代码的动作，解决接下来要做的很多事。

容器技术使这一切变得更完美。

典型的一个场景：

我们写一个前端的工程，假设是基于vue.js的框架开发的，提交代码之后希望跑一跑测试用例，然后build压缩一个到dist目录里，再把这个目录的静态文件用nginx代理一下。 
最后打成docker镜像放到镜像仓库。 甚至还可以增加一个在线上运行起来的流程。

现在告诉你，只需要一个git push动作，接下来所有的事CI工具会帮你解决！这样的系统如果你还没用上的话，那请问还在等什么。接下来会系统的向大家介绍这一切。

# 代码仓库管理
首先SVN这种渣渣软件就该尽早淘汰，没啥好说的，有git真的没有SVN存在的必要了我觉得。

所以我们选一个git仓库，强烈推荐gogs，一个很优秀的开源软件，谁用谁知道。（广告：sealyun提供一整套打包部署工具，Email:fhtjob@hotmail.com）

啥？如何安装？
```
docker run -d --name gogs-time -v /etc/localtime:/etc/localtime -e TZ=Asia/Shanghai --publish 8022:22 \
           --publish 3000:3000 --volume /data/gogs:/data gogs:latest
```
访问3000端口，然后就没有然后了

# CI 工具
至于jenkins这种老掉牙基于传统的方式去做CI的东西，即便功能再强大本尊也是不推崇的。  做一个功能强大的东西不难，难的是大道至简。

当你用过drone之后。。。

装：
```
version: '2'

services:
  drone-server:
    image: drone/drone:0.7
    ports:
      - 80:8000
    volumes:
      - /var/lib/drone:/var/lib/drone/
    restart: always
    environment:
      - DRONE_OPEN=true
      - DOCKER_API_VERSION=1.24
      - DRONE_HOST=10.1.86.206
      - DRONE_GOGS=true
      - DRONE_GOGS_URL=http://10.1.86.207:3000/
      - DRONE_SECRET=ok

  drone-agent:
    image: drone/drone:0.7
    command: agent
    restart: always
    depends_on:
      - drone-server
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - DOCKER_API_VERSION=1.24
      - DRONE_SERVER=ws://drone-server:8000/ws/broker
      - DRONE_SECRET=ok
```
`docker-compose up -d` 然后你懂的，也没有然后了

用gogs账户登录drone即可

每个步骤就是个容器，每个插件也是个容器，各种组合，简直就是活字印刷术

怎么使用这种初级肤浅的内容我就不赘述了，但是有很多坑的地方：

* 装drone的机器能用aufs尽量用，device mapper有些插件是跑不了的，如一些docker in docker的插件，这不算是drone的毛病，只能说docker对 docker in docker支持不够好
* centos对aufs支持不够好，如果想用centos支持aufs，那你可得折腾折腾了，社区方案在此：https://github.com/sealyun/kernel-ml-aufs

# 镜像仓库
用harbor吧，反正也没遇到更好的了,官方离线包也是一键安装的，没啥好说的了。

# 关于CD
CI是以git触发的，可能我们还想用别的方式触发，典型的场景就是运维不懂git，只想在界面上点击一下执行一个工作流。 现有的drone还不支持这个。

我们是自己开发的。

其实我个人推崇git方式触发部署。传说中的gitops。
