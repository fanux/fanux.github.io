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
循环的调用了这一坨函数，我们只需要看其中一两个即可，其它的都差不多

```
certActions := []func(cfg *kubeadmapi.MasterConfiguration) error{
	CreateCACertAndKeyfiles,
	CreateAPIServerCertAndKeyFiles,
	CreateAPIServerKubeletClientCertAndKeyFiles,
	CreateServiceAccountKeyAndPublicKeyFiles,
	CreateFrontProxyCACertAndKeyFiles,
	CreateFrontProxyClientCertAndKeyFiles,
}
```
根证书生成：

```

//返回了根证书的公钥和私钥
func NewCACertAndKey() (*x509.Certificate, *rsa.PrivateKey, error) {

	caCert, caKey, err := pkiutil.NewCertificateAuthority()
	if err != nil {
		return nil, nil, fmt.Errorf("failure while generating CA certificate and key: %v", err)
	}

	return caCert, caKey, nil
}

```
k8s.io/client-go/util/cert 这个库里面有两个函数，一个生成key的一个生成cert的：

```
key, err := certutil.NewPrivateKey()
config := certutil.Config{
	CommonName: "kubernetes",
}
cert, err := certutil.NewSelfSignedCACert(config, key)
```
config里面我们也可以填充一些别的证书信息：
```
type Config struct {
	CommonName   string
	Organization []string
	AltNames     AltNames
	Usages       []x509.ExtKeyUsage
}
```
私钥就是封装了rsa库里面的函数：
```
	"crypto/rsa"
	"crypto/x509"
func NewPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(cryptorand.Reader, rsaKeySize)
}
```
自签证书,所以根证书里只有CommonName信息，Organization相当于没设置：
```
func NewSelfSignedCACert(cfg Config, key *rsa.PrivateKey) (*x509.Certificate, error) {
	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(duration365d * 10).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA: true,
	}

	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}
```
生成好之后把之写入文件：
```
 pkiutil.WriteCertAndKey(pkiDir, baseName, cert, key);
certutil.WriteCert(certificatePath, certutil.EncodeCertPEM(cert))
```
这里调用了pem库进行了编码
```
encoding/pem

func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  CertificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}
```

然后我们看apiserver的证书生成：
```
caCert, caKey, err := loadCertificateAuthorithy(cfg.CertificatesDir, kubeadmconstants.CACertAndKeyBaseName)
//从根证书生成apiserver证书
apiCert, apiKey, err := NewAPIServerCertAndKey(cfg, caCert, caKey)
```

这时需要关注AltNames了比较重要，所有需要访问master的地址域名都得加进去，对应配置文件中apiServerCertSANs字段,其它东西与根证书无差别
```
config := certutil.Config{
	CommonName: kubeadmconstants.APIServerCertCommonName,
	AltNames:   *altNames,
	Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
}
```

### 创建k8s配置文件
可以看到创建了这些文件
```
return createKubeConfigFiles(
	outDir,
	cfg,
	kubeadmconstants.AdminKubeConfigFileName,
	kubeadmconstants.KubeletKubeConfigFileName,
	kubeadmconstants.ControllerManagerKubeConfigFileName,
	kubeadmconstants.SchedulerKubeConfigFileName,
)
```
k8s封装了两个渲染配置的函数：
区别是你的kubeconfig文件里会不会产生token，比如你进入dashboard需要一个token，或者你调用api需要一个token那么请生成带token的配置
生成的conf文件基本一直只是比如ClientName这些东西不同，所以加密后的证书也不同，ClientName会被加密到证书里，然后k8s取出来当用户使用

所以重点来了，我们做多租户时也要这样去生成。然后给该租户绑定角色。
```
return kubeconfigutil.CreateWithToken(
	spec.APIServer,
	"kubernetes",
	spec.ClientName,
	certutil.EncodeCertPEM(spec.CACert),
	spec.TokenAuth.Token,
), nil

return kubeconfigutil.CreateWithCerts(
	spec.APIServer,
	"kubernetes",
	spec.ClientName,
	certutil.EncodeCertPEM(spec.CACert),
	certutil.EncodePrivateKeyPEM(clientKey),
	certutil.EncodeCertPEM(clientCert),
), nil
```
然后就是填充Config结构体喽, 最后写到文件里，略
```
"k8s.io/client-go/tools/clientcmd/api
return &clientcmdapi.Config{
	Clusters: map[string]*clientcmdapi.Cluster{
		clusterName: {
			Server: serverURL,
			CertificateAuthorityData: caCert,
		},
	},
	Contexts: map[string]*clientcmdapi.Context{
		contextName: {
			Cluster:  clusterName,
			AuthInfo: userName,
		},
	},
	AuthInfos:      map[string]*clientcmdapi.AuthInfo{},
	CurrentContext: contextName,
}
```

### 创建static pod yaml文件
这里返回了apiserver manager scheduler的pod结构体,
```
specs := GetStaticPodSpecs(cfg, k8sVersion)
staticPodSpecs := map[string]v1.Pod{
	kubeadmconstants.KubeAPIServer: staticpodutil.ComponentPod(v1.Container{
		Name:          kubeadmconstants.KubeAPIServer,
		Image:         images.GetCoreImage(kubeadmconstants.KubeAPIServer, cfg.GetControlPlaneImageRepository(), cfg.KubernetesVersion, cfg.UnifiedControlPlaneImage),
		Command:       getAPIServerCommand(cfg, k8sVersion),
		VolumeMounts:  staticpodutil.VolumeMountMapToSlice(mounts.GetVolumeMounts(kubeadmconstants.KubeAPIServer)),
		LivenessProbe: staticpodutil.ComponentProbe(cfg, kubeadmconstants.KubeAPIServer, int(cfg.API.BindPort), "/healthz", v1.URISchemeHTTPS),
		Resources:     staticpodutil.ComponentResources("250m"),
		Env:           getProxyEnvVars(),
	}, mounts.GetVolumes(kubeadmconstants.KubeAPIServer)),
	kubeadmconstants.KubeControllerManager: staticpodutil.ComponentPod(v1.Container{
		Name:          kubeadmconstants.KubeControllerManager,
		Image:         images.GetCoreImage(kubeadmconstants.KubeControllerManager, cfg.GetControlPlaneImageRepository(), cfg.KubernetesVersion, cfg.UnifiedControlPlaneImage),
		Command:       getControllerManagerCommand(cfg, k8sVersion),
		VolumeMounts:  staticpodutil.VolumeMountMapToSlice(mounts.GetVolumeMounts(kubeadmconstants.KubeControllerManager)),
		LivenessProbe: staticpodutil.ComponentProbe(cfg, kubeadmconstants.KubeControllerManager, 10252, "/healthz", v1.URISchemeHTTP),
		Resources:     staticpodutil.ComponentResources("200m"),
		Env:           getProxyEnvVars(),
	}, mounts.GetVolumes(kubeadmconstants.KubeControllerManager)),
	kubeadmconstants.KubeScheduler: staticpodutil.ComponentPod(v1.Container{
		Name:          kubeadmconstants.KubeScheduler,
		Image:         images.GetCoreImage(kubeadmconstants.KubeScheduler, cfg.GetControlPlaneImageRepository(), cfg.KubernetesVersion, cfg.UnifiedControlPlaneImage),
		Command:       getSchedulerCommand(cfg),
		VolumeMounts:  staticpodutil.VolumeMountMapToSlice(mounts.GetVolumeMounts(kubeadmconstants.KubeScheduler)),
		LivenessProbe: staticpodutil.ComponentProbe(cfg, kubeadmconstants.KubeScheduler, 10251, "/healthz", v1.URISchemeHTTP),
		Resources:     staticpodutil.ComponentResources("100m"),
		Env:           getProxyEnvVars(),
	}, mounts.GetVolumes(kubeadmconstants.KubeScheduler)),
}

//获取特定版本的镜像
func GetCoreImage(image, repoPrefix, k8sVersion, overrideImage string) string {
	if overrideImage != "" {
		return overrideImage
	}
	kubernetesImageTag := kubeadmutil.KubernetesVersionToImageTag(k8sVersion)
	etcdImageTag := constants.DefaultEtcdVersion
	etcdImageVersion, err := constants.EtcdSupportedVersion(k8sVersion)
	if err == nil {
		etcdImageTag = etcdImageVersion.String()
	}
	return map[string]string{
		constants.Etcd:                  fmt.Sprintf("%s/%s-%s:%s", repoPrefix, "etcd", runtime.GOARCH, etcdImageTag),
		constants.KubeAPIServer:         fmt.Sprintf("%s/%s-%s:%s", repoPrefix, "kube-apiserver", runtime.GOARCH, kubernetesImageTag),
		constants.KubeControllerManager: fmt.Sprintf("%s/%s-%s:%s", repoPrefix, "kube-controller-manager", runtime.GOARCH, kubernetesImageTag),
		constants.KubeScheduler:         fmt.Sprintf("%s/%s-%s:%s", repoPrefix, "kube-scheduler", runtime.GOARCH, kubernetesImageTag),
	}[image]
}
//然后就把这个pod写到文件里了，比较简单
 staticpodutil.WriteStaticPodToDisk(componentName, manifestDir, spec); 
```
创建etcd的一样，不多废话

### 等待kubelet启动成功
这个错误非常容易遇到，看到这个基本就是kubelet没起来，我们需要检查：selinux swap 和Cgroup driver是不是一致
setenforce 0 && swapoff -a && systemctl restart kubelet如果不行请保证 kubelet的Cgroup driver与docker一致，docker info|grep Cg
```
go func(errC chan error, waiter apiclient.Waiter) {
	// This goroutine can only make kubeadm init fail. If this check succeeds, it won't do anything special
	if err := waiter.WaitForHealthyKubelet(40*time.Second, "http://localhost:10255/healthz"); err != nil {
		errC <- err
	}
}(errorChan, waiter)

go func(errC chan error, waiter apiclient.Waiter) {
	// This goroutine can only make kubeadm init fail. If this check succeeds, it won't do anything special
	if err := waiter.WaitForHealthyKubelet(60*time.Second, "http://localhost:10255/healthz/syncloop"); err != nil {
		errC <- err
	}
}(errorChan, waiter)
```

### 创建DNS和kubeproxy
我就是在此发现coreDNS的
```
if features.Enabled(cfg.FeatureGates, features.CoreDNS) {
	return coreDNSAddon(cfg, client, k8sVersion)
}
return kubeDNSAddon(cfg, client, k8sVersion)
```
然后coreDNS的yaml配置模板直接是写在代码里的：
/app/phases/addons/dns/manifests.go
```
	CoreDNSDeployment = `
apiVersion: apps/v1beta2
kind: Deployment
metadata:
  name: coredns
  namespace: kube-system
  labels:
    k8s-app: kube-dns
spec:
  replicas: 1
  selector:
    matchLabels:
      k8s-app: kube-dns
  template:
    metadata:
      labels:
        k8s-app: kube-dns
    spec:
      serviceAccountName: coredns
      tolerations:
      - key: CriticalAddonsOnly
        operator: Exists
      - key: {{ .MasterTaintKey }}
...
```
然后渲染模板，最后调用k8sapi创建,这种创建方式可以学习一下，虽然有点拙劣，这地方写的远不如kubectl好
```
coreDNSConfigMap := &v1.ConfigMap{}
if err := kuberuntime.DecodeInto(legacyscheme.Codecs.UniversalDecoder(), configBytes, coreDNSConfigMap); err != nil {
	return fmt.Errorf("unable to decode CoreDNS configmap %v", err)
}

// Create the ConfigMap for CoreDNS or update it in case it already exists
if err := apiclient.CreateOrUpdateConfigMap(client, coreDNSConfigMap); err != nil {
	return err
}

coreDNSClusterRoles := &rbac.ClusterRole{}
if err := kuberuntime.DecodeInto(legacyscheme.Codecs.UniversalDecoder(), []byte(CoreDNSClusterRole), coreDNSClusterRoles); err != nil {
	return fmt.Errorf("unable to decode CoreDNS clusterroles %v", err)
}
...
```

这里值得一提的是kubeproxy的configmap真应该把apiserver地址传入进来，允许自定义，因为做高可用时需要指定虚拟ip，得修改，很麻烦
kubeproxy大差不差，不说了,想改的话改： app/phases/addons/proxy/manifests.go

## kubeadm join
kubeadm join比较简单，一句话就可以说清楚，获取cluster info, 创建kubeconfig，怎么创建的kubeinit里面已经说了。带上token让kubeadm有权限
可以拉取
```
return https.RetrieveValidatedClusterInfo(cfg.DiscoveryFile)

cluster info内容
type Cluster struct {
	// LocationOfOrigin indicates where this object came from.  It is used for round tripping config post-merge, but never serialized.
	LocationOfOrigin string
	// Server is the address of the kubernetes cluster (https://hostname:port).
	Server string `json:"server"`
	// InsecureSkipTLSVerify skips the validity check for the server's certificate. This will make your HTTPS connections insecure.
	// +optional
	InsecureSkipTLSVerify bool `json:"insecure-skip-tls-verify,omitempty"`
	// CertificateAuthority is the path to a cert file for the certificate authority.
	// +optional
	CertificateAuthority string `json:"certificate-authority,omitempty"`
	// CertificateAuthorityData contains PEM-encoded certificate authority certificates. Overrides CertificateAuthority
	// +optional
	CertificateAuthorityData []byte `json:"certificate-authority-data,omitempty"`
	// Extensions holds additional information. This is useful for extenders so that reads and writes don't clobber unknown fields
	// +optional
	Extensions map[string]runtime.Object `json:"extensions,omitempty"`
}

return kubeconfigutil.CreateWithToken(
	clusterinfo.Server,
	"kubernetes",
	TokenUser,
	clusterinfo.CertificateAuthorityData,
	cfg.TLSBootstrapToken,
), nil
```

CreateWithToken上文提到了不再赘述，这样就能去生成kubelet配置文件了，然后把kubelet启动起来即可
