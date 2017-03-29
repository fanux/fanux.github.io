## 配置docker engine使engine挂掉时容器继续运行
两种方式：
* 修改/etc/docker/daemon.json 如果不想engine重启，给engine发送SIGHUP信号使engine重新加载配置文件
* 直接加启动参数：--live-restore,  如用systemd管理修改这个配置文件：/usr/lib/systemd/system/docker.service,
  然后执行systemctl daemon-reload && service docker restart

加了这个参数之后，重启engine就不会使容器退出了

## 实践
此功能在做engine升级时非常有用，现在就以docker1.12升级到1.13为例详细介绍。

我们有一个容器正在运行，已经运行了两个星期了(ps:我们已经配置了 --live-restore启动参数)
```
[root@dev-86-201 ~]# docker ps
CONTAINER ID        IMAGE                                      COMMAND             CREATED             STATUS              PORTS               NAMES
4c73d9658275        dev.reg.iflytek.com/devops/whoami:latest   "/whoamI"           2 weeks ago         Up 10 minutes       80/tcp              0Fack
```
