## runc 架构破析
这里的spec.Process就是我们真正要运行的容器中的进程。
```
return r.run(&spec.Process)
```
把这个塞到libcontainer.Process里去了：
```
 lp := &libcontainer.Process{
     Args: p.Args,
     Env:  p.Env,
     // TODO: fix libcontainer's API to better support uid/gid in a typesafe way.
     User:            fmt.Sprintf("%d:%d", p.User.UID, p.User.GID),
     Cwd:             p.Cwd,
     Label:           p.SelinuxLabel,
     NoNewPrivileges: &p.NoNewPrivileges,
     AppArmorProfile: p.ApparmorProfile,
 }
```
我试了个busybox的容器，把p.Args和p.Env打印出来看一下,正是config.json中的内容：
```
fmt.Println("Args: ", p.Args, "env", p.Env)
//Args:  [sh] env [PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin TERM=xterm]
```

这里真正调用的是container的Start 或者Run方法
```
    var (
        startFn = r.container.Start
    )
    if !r.create {
        startFn = r.container.Run
    }
    if err = startFn(process); err != nil {
        return -1, err
    }

```
我们看linux_container的Run实现,在libcontainer/container_linux.go下面：
相信你能找到这,这个parent到底是何方神圣？：
```
parent, err := c.newParentProcess(process, isInit)
                    |
                    +---> cmd, err := c.commandTemplate(p, childPipe)
```
这个cmd启动是的啥？我们容器中entrypoint 或者CMD?? 事实证明都不是：
```
cmd := exec.Command(c.initArgs[0], c.initArgs[1:]...)
fmt.Println("cmd is: ", c.initArgs[0], c.initArgs[1:])
//cmd is:  /proc/self/exe [init]
```
linux菜鸟表示看不懂了，`/proc/self/exe`是什么鬼？赶紧百度一下。强(sha)大(bi)百度告诉我们这代表当前进程，所以实际上是想调用`runc init`

## runc run进程与runc init进程之间的通信
#### runc run进程
大家记住一点，容器外面的一些设置runc run去做，在init启动时就设置了init的namespace所以容器内需要做什么都由init去实现。
比较典型的如在实现bridge网桥时，在容器里面创建eth0网卡就由init进程实现

run先把bootstrapData发给init，具体是什么回头讨论
```
                run            init
                 |               |
                 |-------------->| bootstrapData
setNs            |               |
createNetwork    |               |
                 |-------------->| sendConfig
                 |<--------------| procReady 我准备好啦
启动程序吧procRun|-------------->| 
                 |<--------------| procHooks 执行钩子
继续procResume   |-------------->| 
                 |               |
```
看看sendConfig发了些什么：
```
func (p *initProcess) sendConfig() error {
    fmt.Println("sendconfig to init process, config is: ", p.config)
    //sendconfig to init process, config is:  
    //&{[sh] [PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin TERM=xterm] / 0xc420018780   true 0:0 [] 0xc4200ec1e0 [0xc420075380] 0 test [{7 1024 1024}] true false}
    return utils.WriteJSON(p.parentPipe, p.config)
}
```
就是我们要启动进程的信息, 除了initProcess还有个setnsProcess 两者区别是：
TODO

#### runc init进程
我们想在init里面如上面一样打印一些调试信息就会发现不太管用了,因为已经在子进程中了,所以我们把调试信息输出到文件中
```
factory, _ := libcontainer.New("")
factory.StartInitialization(); 
```
libcontainer/factory_linux.go里面有实现
在pipe中获取到config信息
```
newContainerInit(t initType, pipe *os.File, consoleSocket *os.File, stateDirFD int) 
    if err := json.NewDecoder(pipe).Decode(&config); err != nil {
} 

就是我们需要启动进程的信息
//{[sh] [PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin TERM=xterm] / 0xc42005e480   true 0:0 [] 0xc4200dc1e0 [0xc420070240] 0 test [{7 1024 1024}] true false}
```
所以我们去看linuxStandardInit的Init方法即可：
做好一些准备后给run进程发送准备好了的信息，表示可以进行Execv启动进程了
```
    // Tell our parent that we're ready to Execv. This must be done before the
    // Seccomp rules have been applied, because we need to be able to read and
    // write to a socket.
    if err := syncParentReady(l.pipe); err != nil {
        return err
    }
```
我们要的东西：
```
    if err := syscall.Exec(name, l.config.Args[0:], os.Environ()); err != nil {
        return newSystemErrorWithCause(err, "exec user process")
    }
```

### 切换rootfs
这里有个有意思的地方，我们ShowLog的作用是往`/log`文件中写日志，请看下面：
```
    utils.ShowLog("2 - linuxSetnsInit Init")
    // prepareRootfs() can be executed only for a new mount namespace.
    if l.config.Config.Namespaces.Contains(configs.NEWNS) {
        if err := prepareRootfs(l.pipe, l.config.Config); err != nil {
            return err
        }
    }
    utils.ShowLog("3 - linuxSetnsInit Init")
```
上面一条ShowLog会写到宿主机的`/ShowLog`文件中，而 下面一个ShowLog因为已经切换了rootfs会写到容器里面的
`/log`文件中

## 总结
至此我们容器创建流程大的架构梳理了一遍，为了看清整个架构忽略了很多细节，当然我会在其它文章中介绍别的一些细节内容. 欢迎大家关注[sealyun](lameleg.com)
