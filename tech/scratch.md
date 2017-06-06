# 如何让镜像尽可能小
很容器想到from scratch, 就是没任何基础镜像
```
FROM scratch
COPY p /
ENTRYPOINT ["/p"]
```
有几点要注意：

* ENTRYPOINT 或者CMD 必须要用[]这种模式，如果直接/p会用sh去启动，而scratch没有shell导致失败
* 二进制程序必须静态编译，也就是不能依赖libc什么的动态库

动态编译的bin程序：
```
[root@dev-86-205 ci-sftp]# ldd p
    linux-vdso.so.1 =>  (0x00007ffd6ef7b000)
    libpthread.so.0 => /lib64/libpthread.so.0 (0x00007fa28f94e000)
    libc.so.6 => /lib64/libc.so.6 (0x00007fa28f58d000)
    /lib64/ld-linux-x86-64.so.2 (0x00007fa28fb72000)
```
这种情况下出来的bin程序可能会出现问题：

```
standard_init_linux.go:175: exec user process caused "no such file or directory”
```

静态编译的bin程序,这是我们scratch需要的：
```
[root@dev-86-205 ci-sftp]# ldd p
    不是动态可执行文件
```

golang中静态编译命令：
```
go build --ldflags '-linkmode external -extldflags "-static”'
```

如果不静态编译那可能得拷贝一堆动态库到镜像中，很多lowB就是那么做的
