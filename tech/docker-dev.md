## docker开发流程
注意容器构建时的信息：
```
Install runc version 992a5be178a62e026f4069f443c6164912adbf09
+ git clone https://github.com/opencontainers/runc.git /tmp/tmp.NdftaLJucp/src/github.com/opencontainers/runc
Cloning into '/tmp/tmp.NdftaLJucp/src/github.com/opencontainers/runc'...
+ cd /tmp/tmp.NdftaLJucp/src/github.com/opencontainers/runc
+ git checkout -q 992a5be178a62e026f4069f443c6164912adbf09
+ make BUILDTAGS=seccomp apparmor selinux static
CGO_ENABLED=1 go build -i -tags "seccomp apparmor selinux cgo static_build" -ldflags "-w -extldflags -static -X main.gitCommit="992a5be178a62e026f4069f443c6164912adbf09" -X main.version=1.0.0-rc3" -o runc .
CGO_ENABLED=1 go build -i -tags "seccomp apparmor selinux cgo static_build" -ldflags "-w -extldflags -static -X main.gitCommit="992a5be178a62e026f4069f443c6164912adbf09" -X main.version=1.0.0-rc3" -o contrib/cmd/recvtty/recvtty ./contrib/cmd/recvtty
+ cp runc /usr/local/bin/docker-runc
+ install_containerd static
+ echo Install containerd version 8ef7df579710405c4bb6e0812495671002ce08e0
Install containerd version 8ef7df579710405c4bb6e0812495671002ce08e0
+ git clone https://github.com/containerd/containerd.git /tmp/tmp.NdftaLJucp/src/github.com/containerd/containerd
Cloning into '/tmp/tmp.NdftaLJucp/src/github.com/containerd/containerd'...
+ cd /tmp/tmp.NdftaLJucp/src/github.com/containerd/containerd
+ git checkout -q 8ef7df579710405c4bb6e0812495671002ce08e0
+ make static
cd ctr && go build -ldflags "-w -extldflags -static -X github.com/containerd/containerd.GitCommit=8ef7df579710405c4bb6e0812495671002ce08e0 " -tags "" -o ../bin/ctr
cd containerd && go build -ldflags "-w -extldflags -static -X github.com/containerd/containerd.GitCommit=8ef7df579710405c4bb6e0812495671002ce08e0 " -tags "" -o ../bin/containerd
cd containerd-shim && go build -ldflags "-w -extldflags -static -X github.com/containerd/containerd.GitCommit=8ef7df579710405c4bb6e0812495671002ce08e0 " -tags "" -o ../bin/containerd-shim
+ cp bin/containerd /usr/local/bin/docker-containerd
+ cp bin/containerd-shim /usr/local/bin/docker-containerd-shim
+ cp bin/ctr /usr/local/bin/docker-containerd-ctr
+ echo Install tini version 949e6facb77383876aeff8a6944dde66b3089574
+ git clone https://github.com/krallin/tini.git /tmp/tmp.NdftaLJucp/tini
Install tini version 949e6facb77383876aeff8a6944dde66b3089574
Cloning into '/tmp/tmp.NdftaLJucp/tini'...
+ cd /tmp/tmp.NdftaLJucp/tini
+ git checkout -q 949e6facb77383876aeff8a6944dde66b3089574
+ cmake .
-- The C compiler identification is GNU 4.9.2
-- Check for working C compiler: /usr/bin/cc
-- Check for working C compiler: /usr/bin/cc -- works
-- Detecting C compiler ABI info
-- Detecting C compiler ABI info - done
-- Performing Test HAS_BUILTIN_FORTIFY
-- Performing Test HAS_BUILTIN_FORTIFY - Failed
-- Configuring done
-- Generating done
-- Build files have been written to: /tmp/tmp.NdftaLJucp/tini
+ make tini-static
Scanning dependencies of target tini-static
[100%] Building C object CMakeFiles/tini-static.dir/src/tini.c.o
Linking C executable tini-static
[100%] Built target tini-static
+ cp tini-static /usr/local/bin/docker-init
+ export CGO_ENABLED=0
+ install_proxy
+ echo Install docker-proxy version 7b2b1feb1de4817d522cc372af149ff48d25028e
Install docker-proxy version 7b2b1feb1de4817d522cc372af149ff48d25028e
+ git clone https://github.com/docker/libnetwork.git /tmp/tmp.NdftaLJucp/src/github.com/docker/libnetwork
Cloning into '/tmp/tmp.NdftaLJucp/src/github.com/docker/libnetwork'...
+ cd /tmp/tmp.NdftaLJucp/src/github.com/docker/libnetwork
+ git checkout -q 7b2b1feb1de4817d522cc372af149ff48d25028e
+ go build -ldflags= -o /usr/local/bin/docker-proxy github.com/docker/libnetwork/cmd/proxy
+ install_bindata
+ echo Install go-bindata version a0ff2567cfb70903282db057e799fd826784d41d
+ git clone https://github.com/jteeuwen/go-bindata /tmp/tmp.NdftaLJucp/src/github.com/jteeuwen/go-bindata
Install go-bindata version a0ff2567cfb70903282db057e799fd826784d41d
Cloning into '/tmp/tmp.NdftaLJucp/src/github.com/jteeuwen/go-bindata'...
+ cd /tmp/tmp.NdftaLJucp/src/github.com/jteeuwen/go-bindata
+ git checkout -q a0ff2567cfb70903282db057e799fd826784d41d
+ go build -o /usr/local/bin/go-bindata github.com/jteeuwen/go-bindata/go-bindata
+ install_dockercli
+ echo Install docker/cli version 7230906e0e297999eb33da74e0279c5cf41a119e
+ git clone https://github.com/dperny/cli /tmp/tmp.NdftaLJucp/src/github.com/docker/cli
Install docker/cli version 7230906e0e297999eb33da74e0279c5cf41a119e
Cloning into '/tmp/tmp.NdftaLJucp/src/github.com/docker/cli'...
+ cd /tmp/tmp.NdftaLJucp/src/github.com/docker/cli
+ git checkout -q 7230906e0e297999eb33da74e0279c5cf41a119e
+ go build -o /usr/local/bin/docker github.com/docker/cli/cmd/docker
+ [ 1 -eq 1 ]
+ rm -rf /tmp/tmp.NdftaLJucp
```

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
我们启动容器时小作修改使可以很方便的在本机上改代码，在容器里构建：
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
    -v /Users/fanux/work/src/github.com:/go/src/github.com \
    -t "docker-dev:dry-run-test" bash
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
这时你可以修改一些docker的代码了，然后重新编译即可，官方教程是修改了docker的代码，而我更感兴趣的是runc，下面就来改改runc试试。

## 修改runc代码
在容器内执行：
```
mkdir /mycontainer
cd /mycontainer
mkdir rootfs
docker export $(docker create busybox) | tar -C rootfs -xvf -
# 生成容器的配置文件config.json
docker-runc spec
docker-runc run mycontainerid
```
进到github.com/opencontainers/runc/run.go，修改代码：
```
spec, err := setupSpec(context)
fmt.Println("spec is: ", *spec)
```
再容器runc目录构建runc：
```
make && make install
```
这时再到mycontainer目录用我们构建的这个runc运行容器,我们打印的信息出来了。
```
root@7d8c68bba090:/go/src/github.com/opencontainers/mycontainer# runc run test
spec is:  {1.0.0-rc5 {linux amd64} {true {0 0} {0 0 [] } [sh] [PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin TERM=xterm] / 0xc42001a500 [{RLIMIT_NOFILE 1024 1024}] true  } {rootfs true} runc [{/proc proc proc []} {/dev tmpfs tmpfs [nosuid strictatime mode=755 size=65536k]} {/dev/pts devpts devpts [nosuid noexec newinstance ptmxmode=0666 mode=0620 gid=5]} {/dev/shm tmpfs shm [nosuid noexec nodev mode=1777 size=65536k]} {/dev/mqueue mqueue mqueue [nosuid noexec nodev]} {/sys sysfs sysfs [nosuid noexec nodev ro]} {/sys/fs/cgroup cgroup cgroup [nosuid noexec nodev relatime ro]}] <nil> map[] 0xc420084380 <nil> <nil>}
/ #
```
