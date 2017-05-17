## docker开发流程

### 编译docker源码
clone moby
```
$ git clone https://github.com/moby/moby
```
创建一个分支：
```
$ git checkout dry-run-test
```
构建容器编译：

```
$ make BIND_DIR=. shell
```
运行容器：
```
docker run --rm -i --privileged \
-e BUILDFLAGS -e KEEPBUNDLE \
-e DOCKER_BUILD_GOGC \
-e DOCKER_BUILD_PKGS \
-e DOCKER_CLIENTONLY \
-e DOCKER_DEBUG \
-e DOCKER_EXPERIMENTAL \
-e DOCKER_GITCOMMIT \
-e DOCKER_GRAPHDRIVER=devicemapper \
-e DOCKER_INCREMENTAL_BINARY \
-e DOCKER_REMAP_ROOT -e DOCKER_STORAGE_OPTS \
-e DOCKER_USERLANDPROXY -e TESTDIRS \
-e TESTFLAGS -e TIMEOUT \
-v "home/ubuntu/repos/docker/bundles:/go/src/github.com/moby/moby/bundles" -t "docker-dev:dry-run-test" bash
root@f31fa223770f:/go/src/github.com/moby/moby#
```
容器内编译源码：
```
root@a8b2885ab900:/go/src/github.com/moby/moby# hack/make.sh binary
...output snipped...
bundles/1.12.0-dev already exists. Removing.

---> Making bundle: binary (in bundles/1.12.0-dev/binary)
Building: bundles/1.12.0-dev/binary/docker-1.12.0-dev
Created binary: bundles/1.12.0-dev/binary/docker-1.12.0-dev
Copying nested executables into bundles/1.12.0-dev/binary
```
拷贝bin文件：
```
root@a8b2885ab900:/go/src/github.com/moby/moby# cp bundles/1.12.0-dev/binary-client/docker* /usr/bin/
root@a8b2885ab900:/go/src/github.com/moby/moby# cp bundles/1.12.0-dev/binary-daemon/docker* /usr/bin/
```
启动containerd:
```
root@a8b2885ab900:/go/src/github.com/docker/docker# dockerd -D &
...output snipped...
DEBU[0001] Registering POST, /networks/{id:.*}/connect
DEBU[0001] Registering POST, /networks/{id:.*}/disconnect
DEBU[0001] Registering DELETE, /networks/{id:.*}
INFO[0001] API listen on /var/run/docker.sock
DEBU[0003] containerd connection state change: READY
```
