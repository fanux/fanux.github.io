# kubeadm源码分析

说句实在话，kubeadm的代码写的真心一般，质量不是很高。

几个关键点来先说一下kubeadm干的几个核心的事：

* kubeadm 生成证书在/etc/kubernetes/pki目录下
* kubeadm 生成static pod yaml配置，全部在/etc/kubernetes/manifasts下
* kubeadm 生成kubelet配置，kubectl配置等 在/etc/kubernetes下
* kubeadm 通过client go去启动dns

## kubeadm init
代码入口 `cmd/kubeadm/app/cmd/init.go` 建议大家去看看cobra

找到Run函数来分析下主要流程：

1. 如果证书不存在，就创建证书，所以如果我们有自己的证书可以把它放在/etc/kubernetes/pki下即可, 下文细看如果生成证书
```
	if res, _ := certsphase.UsingExternalCA(i.cfg); !res {
		if err := certsphase.CreatePKIAssets(i.cfg); err != nil {
			return err
		}
```

2. 创建kubeconfig文件
```
		if err := kubeconfigphase.CreateInitKubeConfigFiles(kubeConfigDir, i.cfg); err != nil {
			return err
		}
```

3. 创建manifest文件，etcd apiserver manager scheduler都在这里创建, 可以看到如果你的配置文件里已经写了etcd的地址了，就不创建了，这我们就可以自己装etcd集群，而不用默认单点的etcd，很有用
```
controlplanephase.CreateInitStaticPodManifestFiles(manifestDir, i.cfg); 
if len(i.cfg.Etcd.Endpoints) == 0 {
	if err := etcdphase.CreateLocalEtcdStaticPodManifestFile(manifestDir, i.cfg); err != nil {
		return fmt.Errorf("error creating local etcd static pod manifest file: %v", err)
	}
}
```

4. 等待APIserver和kubelet启动成功，这里就会遇到我们经常遇到的镜像拉不下来的错误，其实有时kubelet因为别的原因也会报这个错，让人误以为是镜像弄不下来
```
if err := waitForAPIAndKubelet(waiter); err != nil {
	ctx := map[string]string{
		"Error":                  fmt.Sprintf("%v", err),
		"APIServerImage":         images.GetCoreImage(kubeadmconstants.KubeAPIServer, i.cfg.GetControlPlaneImageRepository(), i.cfg.KubernetesVersion, i.cfg.UnifiedControlPlaneImage),
		"ControllerManagerImage": images.GetCoreImage(kubeadmconstants.KubeControllerManager, i.cfg.GetControlPlaneImageRepository(), i.cfg.KubernetesVersion, i.cfg.UnifiedControlPlaneImage),
		"SchedulerImage":         images.GetCoreImage(kubeadmconstants.KubeScheduler, i.cfg.GetControlPlaneImageRepository(), i.cfg.KubernetesVersion, i.cfg.UnifiedControlPlaneImage),
	}

	kubeletFailTempl.Execute(out, ctx)

	return fmt.Errorf("couldn't initialize a Kubernetes cluster")
}
```

5. 给master加标签，加污点, 所以想要pod调度到master上可以把污点清除了
```
if err := markmasterphase.MarkMaster(client, i.cfg.NodeName); err != nil {
	return fmt.Errorf("error marking master: %v", err)
}
```

6. 生成tocken
```
if err := nodebootstraptokenphase.UpdateOrCreateToken(client, i.cfg.Token, false, i.cfg.TokenTTL.Duration, kubeadmconstants.DefaultTokenUsages, []string{kubeadmconstants.NodeBootstrapTokenAuthGroup}, tokenDescription); err != nil {
	return fmt.Errorf("error updating or creating token: %v", err)
}
```

7. 调用clientgo创建dns和kube-proxy

```
if err := dnsaddonphase.EnsureDNSAddon(i.cfg, client); err != nil {
	return fmt.Errorf("error ensuring dns addon: %v", err)
}

if err := proxyaddonphase.EnsureProxyAddon(i.cfg, client); err != nil {
	return fmt.Errorf("error ensuring proxy addon: %v", err)
}
```

笔者批判代码无脑式的一个流程到底，要是笔者操刀定抽象成接口 RenderConf Save Run Clean等，DNS kube-porxy以及其它组件去实现，然后问题就是没把dns和kubeproxy的配置渲染出来，可能是它们不是static pod的原因, 然后就是join时的bug下文提到

### 证书生成 

## kubeadm join
