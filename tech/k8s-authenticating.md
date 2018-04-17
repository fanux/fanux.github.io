# kubernetes对接第三方认证
广告： [安装包地址](https://market.aliyun.com/products/57742013/cmxz025618.html?spm=5176.730005.0.0.TFKV5K#sku=yuncode1961800000) 

[原文地址](http://lameleg.com/tech/k8s-authenticating.html)

## 概述
本文介绍如何使用github账户去关联自己kubernetes账户。达到如下效果：
1. 使用github用户email作为kubernetes用户，如fhtjob@hotmail.com
2. 创建对应的clusterrole绑定给fhtjob@hotmail.com这个用户
3. 给fhtjob@hotmail这个用户创建一个kubeconfig文件，让改用户可以使用kubectl命令操作集群，且只有部分权限

## dex介绍
[dex](https://github.com/coreos/dex) 是一个统一认证的服务，支持各种认证协议如Ouath2 ldap等，自己可以作为一个identity provider,也可以连到别的id provider(如github)上,dex作为一个中间代理.

## 流程
```
          http://47.52.197.163:5555    http://47.52.197.163:32000
  人(浏览器）   dex client                 dex server               github                      kubectl             kubernetes server
  |   login(scope) |                         |                       |                           |                        |
  |------1-------->|                         |                       |                           |                        |
  |                |----------2------------->|                       |                           |                        |
  |                |                         |----------3----------->|                           |                        |
  |                |                         |   id_token            |                           |                        |
  |                |                         |<---------4------------| callback                  |                        |
  |  id_token      |<----------5-------------|callback               |                           |                        |
  |<-------6-------|                         |                       |                           |                        |
  |                |                         |                       |               id_token    |                        |
  |------------------------------------------------7-------------------------------------------->|        id_token        |
  |                |                         |                       |                           |----------8------------>|
  |                |                         |                       |                           |                        | valid? 
  |                |                         |                       |                           |                        | expired?
  |                |                         |                       |                           |                        | user Authorized?
  |                |                         |                       |                           |<---------9-------------|
  X<----------------------------------------------10---------------------------------------------|                        |
  |                |                         |                       |                           |                        |
  |                |                         |                       |                           |                        |
  |                |                         |                       |                           |                        |
```
* scope: 你需要哪些信息，如邮箱,openid,用户名等
* id_token: 加密后的你需要的信息
* dex client: dex的客户端，比如可以是我们自己写的管理的服务端，会去调用第三方登录的流程，或者我们写的一个网站后台处理登录的逻辑
* dex server: dex的服务端，一边作为client的服务端，另一边其实是github的客户端

1. 用户在浏览器发起登录请求
2. dexclient把请求重定向给dexserver
3. dexserver重定向给github，这时用户就会跳转到github的页面去授权允许访问哪些信息
4. github把对应信息加密调用dexserver的回调url(http://47.52.197.163:32000/callback)把信息传给dex server, 注意区分dex client的回调
5. dexserver把信息回调给dex client(http://47.52.197.163:5555/callback)
6. 浏览器中拿到token
7. 把token加到kubeconfig文件中，让kubectl可以使用
8. kubectl把token传给kubernetes server, server有 dex server的公钥可以解析token，拿到username, 看是否过期，看授权是否允许执行该动作
9. 把执行结果返回给kubectl

## 环境介绍与注意事项
* 采用云服务器进行该实验，Floatingip是47.52.197.163
* 你需要有一个github账户，我的是github.com/fanux 把email(fhtjob@hotmail.com)作为kubernetes账户
* 服务器上要装golang
* 官方教程有很多坑，建议看我的教程
* 需要有个k8s集群，那么我最推荐的安装方式当然是购买我的[安装包](https://market.aliyun.com/products/57742013/cmxz025618.html?spm=5176.730005.0.0.TFKV5K#sku=yuncode1961800000)哈哈

## 安装
### 修改kube apiserver配置
```
[root@master2 ~]# cat /etc/kubernetes/manifests/kube-apiserver.yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    scheduler.alpha.kubernetes.io/critical-pod: ""
  creationTimestamp: null
  labels:
    component: kube-apiserver
    tier: control-plane
  name: kube-apiserver
  namespace: kube-system
spec:
  containers:
  - command:
    - kube-apiserver
    - --oidc-issuer-url=https://47.52.197.163:32000   # 加上这五个参数
    - --oidc-client-id=example-app
    - --oidc-ca-file=/etc/kubernetes/ssl/ca.pem # dex证书，挂载进来的
    - --oidc-username-claim=email
    - --oidc-groups-claim=groups

...

    - mountPath: /etc/kubernetes/ssl  # 把dex的证书挂进去给apiserver使用
      name: dex
      readOnly: true
  volumes:
  - hostPath:
      path: /etc/kubernetes/ssl
      type: DirectoryOrCreate
    name: dex
```
用kubeadm安装的修改/etc/kubernetes/manifests/kube-apiserver.yaml这个文件即可，建议不要直接修改，拷贝出来修改再复制回去，防止kubelet去拉swap文件导致controller manager异常
### 创建github app
点你github头像，settings->developer settins -> new oauth app

Application name: example-app
Homepage URL：https://47.52.197.163:32000  
Authorization callback URL: https://47.52.197.163:32000/callback

URL千万别填错，注意是dex server的URL而不是dex client的5555

然后你就能看到一个ID一个secrect 后面需要用

### 部署dex
没装go的自己去装。。。
```
go get github.com/coreos/dex 
cd $GOPATH/src/github.com/coreos/dex
```
生成证书
gencert.sh需要改一下，把我们IP加进去
```
[alt_names]
DNS.1 = dex.example.com
IP.1 = 47.52.197.163
IP.2 = 172.31.244.238
```
```
$ cd examples/k8s
$ ./gencert.sh
$ cp examples/k8s/ssl /etc/kubernetes # 可曾记得我们挂载的目录
```

创建secrect,这个会给dex server用
```
$ kubectl create secret tls dex.example.com.tls --cert=ssl/cert.pem --key=ssl/key.pem
```
再创建一个secrect给dex server Github OAuth2 客户端用，dex server是github的一个客户端要理解
```
$ kubectl create secret \
    generic github-client \
    --from-literal=client-id=$GITHUB_CLIENT_ID \   # 这俩东西替换成在github页面上创建的APP clientid和secrect
    --from-literal=client-secret=$GITHUB_CLIENT_SECRET
```

启动dex.yaml，注意代码里直接clone下来的没有配置存储，而且镜像比较老，建议用我的：
```
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: dex
  name: dex
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: dex
    spec:
      containers:
      - image: quay.io/coreos/dex:v2.10.0
        name: dex
        command: ["/usr/local/bin/dex", "serve", "/etc/dex/cfg/config.yaml"]

        ports:
        - name: https
          containerPort: 5556

        volumeMounts:
        - name: config
          mountPath: /etc/dex/cfg
        - name: data
          mountPath: /etc/example
        - name: tls
          mountPath: /etc/dex/tls

        env:
        - name: GITHUB_CLIENT_ID
          valueFrom:
            secretKeyRef:
              name: github-client
              key: client-id
        - name: GITHUB_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: github-client
              key: client-secret
      volumes:
      - name: data
        hostPath:
            path: /data/example
      - name: config
        configMap:
          name: dex
          items:
          - key: config.yaml
            path: config.yaml
      - name: tls
        secret:
          secretName: dex.example.com.tls
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: dex
data:
  config.yaml: |
    issuer: https://47.52.197.163:32000
    storage:
      type: sqlite3
      config:
        file: /etc/example/dex.db

    web:
      https: 0.0.0.0:5556
      tlsCert: /etc/dex/tls/tls.crt
      tlsKey: /etc/dex/tls/tls.key
    connectors:
    - type: github
      id: github
      name: GitHub
      config:
        clientID: $GITHUB_CLIENT_ID
        clientSecret: $GITHUB_CLIENT_SECRET
        redirectURI: https://47.52.197.163:32000/callback
        org: kubernetes
    oauth2:
      skipApprovalScreen: true

    staticClients:
    - id: example-app
      redirectURIs:
      - 'http://47.52.197.163:5555/callback'
      name: 'Example App'
      secret: ZXhhbXBsZS1hcHAtc2VjcmV0

    enablePasswordDB: true
    staticPasswords:
    - email: "admin@example.com"
      # bcrypt hash of the string "password"
      hash: "$2a$10$2b2cU8CPhOTaGrs1HRQuAueS7JTT5ZHsHSzYiFPm1leZck7Mc8T4W"
      username: "admin"
      userID: "08a8684b-db88-4b73-90a9-3cd1661f5466"
---
apiVersion: v1
kind: Service
metadata:
  name: dex
spec:
  type: NodePort
  ports:
  - name: dex
    port: 5556
    protocol: TCP
    targetPort: 5556
    nodePort: 32000
  selector:
    app: dex
```
主要修改了：
* 镜像
* 一些地址，改成自己的IP
* 存储，我改成了sqlite, 需要挂载一个文件进去，在宿主机上创建一个文件
```
$ touch /data/example/dex.db
$ kubectl create -f dex.yaml
```
### 启动dex client
编译客户端dex目录下：
```
make
```
启动客户端：
```
$ ./bin/example-app --issuer https://47.52.197.163:32000 --issuer-root-ca examples/k8s/ssl/ca.pem --redirect-uri http://47.52.197.163:5555/callback
```
### 浏览器访问获取token
浏览器访问 http://47.52.197.163:5555 ,点击login后能看到 Log in to dex 下面可以选 log in with Email 和 log in with github
点击log in with github 授权后得到：

```
Token:

eyJhbGciOiJSUzI1NiIsImtpZCI6ImMyZWIzYzkwMmM0NDliMTYwMGNjNzNhMWYyNWVjMjI0MDY4NmE0OGMifQ.eyJpc3MiOiJodHRwczovLzQ3LjUyLjE5Ny4xNjM6MzIwMDAiLCJzdWIiOiJDZ2M0T1RFeU5UVTNFZ1puYVhSb2RXSSIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxNTI0MDIwNzA3LCJpYXQiOjE1MjM5MzQzMDcsImF0X2hhc2giOiI5czJob0lzUHRlMW9nc3VKemRab1pnIiwiZW1haWwiOiJmaHRqb2JAaG90bWFpbC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6InN0ZXZlbiJ9.uJAL08BZioSWPaEFh8R50JQVRw6QXgC1n3sn5ovzaoauy51YFjdSh08UZT8KQon8R5JdZ4U06BczwmOG_tT0mWVd_mDqLnRm6lGpm9znYiC1OLNGZOdzuQVsuxe4Lk1YOvxTsJQtpYuOcXXKkwmdfWNeh4VyZoALiVZxLfL44lSnU55JutLNnGD5S6Aiu6YF0xwlcX5Eq1j2pYtg4isnPtU4k6gbiEYCMPm0Gs3FPljnLT7a-TB1tjZLc4RDwBZ4OoiYRu5mAmH5SHHq1_TS9wDTXX16KlQTG9tS_I11n--1grYTz5WondBoM14BJebDdcSF7nRWJ-I8CU_UYu6gcA
Claims:

{
  "iss": "https://47.52.197.163:32000",
  "sub": "Cgc4OTEyNTU3EgZnaXRodWI",
  "aud": "example-app",
  "exp": 1524020707,
  "iat": 1523934307,
  "at_hash": "9s2hoIsPte1ogsuJzdZoZg",
  "email": "fhtjob@hotmail.com",
  "email_verified": true,
  "name": "steven"
}
Refresh Token:

Chlrem12bDdmdGJ1dWNlYnk0b2llcWd0YzNqEhloNGhwbmlsZnByZ29mdWdsdWZ6bGp4cHhs
```

那么 恭喜你成功了, 这个token就是我们要的东西

### 验证tocken
```
 curl -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImJjOTU0NjdlM2I0OTE5YWE1OTEzZDNkMDU3NGM2ZTRjYjBjY2NhNzgifQ.eyJpc3MiOiJodHRwczovLzQ3LjUyLjE5Ny4xNjM6MzIwMDAiLCJzdWIiOiJDZ2M0T1RFeU5UVTNFZ1puYVhSb2RXSSIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxNTIzOTYyNjUyLCJpYXQiOjE1MjM4NzYyNTIsImF0X2hhc2giOiJFUXRWWm5ObE50c2hhWERfZ3N2UkNBIiwiZW1haWwiOiJmaHRqb2JAaG90bWFpbC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6InN0ZXZlbiJ9.vu0keGMoRGg6OAYpMZNN9zm4pnKXGyXDkZaRNj6MXDY9XsfnBDT4HnXkY17Lvm1ow0xPbq9cgVL3JBZT73jiddgFNAIXJffHfPejlVRSqXx9iF1uEcNIc5tDA1hUPtBrX8n_rzdz0sZsPMb4ZYMx3AdEylszpVrS_OelbB4I_2eLfO0KzwcEknOgV8cZZghCCITl6ZTOeeWEv5FPvJjRC2rpu_MkSY5tAf30SITwldFUMgF8ei3aPrZdojPLgqUWtxKaDmPpcHVLhYr0sLE_BnDZLjGP4ff8l5yy_EfDc7sQsrJR7StwZXRnK-n2omqaV3z-n5IxaUty85e_97FA1g" -k https://172.31.244.238:6443/api/v1/namespaces/default/pods
```
你会发现
```
{
  "kind": "Status",
  "apiVersion": "v1",
  "metadata": {

  },
  "status": "Failure",
  "message": "pods is forbidden: User \"fhtjob@hotmail.com\" cannot list pods in the namespace \"default\"",
  "reason": "Forbidden",
  "details": {
    "kind": "pods"
  },
  "code": 403
}
```
fhtjob@hotmail.com这个用户没有权限访问pods。我们给他创建一个角色绑定：
```
[root@master2 dex]# cat examples/k8s/role.yaml
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: read-secrets-global
subjects:
- kind: User
  name: "fhtjob@hotmail.com" # Name is case sensitive
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: cluster-admin  # 超级用户给他
  apiGroup: rbac.authorization.k8s.io
```
```
$ kubectl create -f examples/k8s/role.yaml
```
再次curl:
```

ot@master2 dex]# curl -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImJjOTU0NjdlM2I0OTE5YWE1OTEzZDNkMDU3NGM2ZTRjYjBjY2NhNzgifQ.eyJpc3MiOiJodHRwczovLzQ3LjUyLjE5Ny4xNjM6MzIwMDAiLCJzdWIiOiJDZ2M0T1RFeU5UVTNFZ1puYVhSb2RXSSIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxNTIzOTYyNjUyLCJpYXQiOjE1MjM4NzYyNTIsImF0X2hhc2giOiJFUXRWWm5ObE50c2hhWERfZ3N2UkNBIiwiZW1haWwiOiJmaHRqb2JAaG90bWFpbC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6InN0ZXZlbiJ9.vu0keGMoRGg6OAYpMZNN9zm4pnKXGyXDkZaRNj6MXDY9XsfnBDT4HnXkY17Lvm1ow0xPbq9cgVL3JBZT73jiddgFNAIXJffHfPejlVRSqXx9iF1uEcNIc5tDA1hUPtBrX8n_rzdz0sZsPMb4ZYMx3AdEylszpVrS_OelbB4I_2eLfO0KzwcEknOgV8cZZghCCITl6ZTOeeWEv5FPvJjRC2rpu_MkSY5tAf30SITwldFUMgF8ei3aPrZdojPLgqUWtxKaDmPpcHVLhYr0sLE_BnDZLjGP4ff8l5yy_EfDc7sQsrJR7StwZXRnK-n2omqaV3z-n5IxaUty85e_97FA1g" -k https://172.31.244.238:6443/api/v1/namespaces/default/pods
{
  "kind": "PodList",
  "apiVersion": "v1",
  "metadata": {
    "selfLink": "/api/v1/namespaces/default/pods",
    "resourceVersion": "333066"
  },
  "items": [
    {
      "metadata": {
        "name": "dex-578588c896-rsp9w",
        "generateName": "dex-578588c896-",
        "namespace": "default",
        "selfLink": "/api/v1/namespaces/default/pods/dex-578588c896-rsp9w",
```
成功
### 把tocken加入到证书中
最简单的方式：
```
[root@master2 dex]# cat ~/.kube/config
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUN5RENDQWJDZ0F3SUJBZ0lCQURBTkJna3Foa2lHOXcwQkFRc0ZBREFWTVJNd0VRWURWUVFERXdwcmRXSmwKY201bGRHVnpNQjRYRFRFNE1ETXlOekV6TURNMU9Gb1hEVEk0TURNeU5ERXpNRE0xT0Zvd0ZURVRNQkVHQTFVRQpBeE1LYTNWaVpYSnVaWFJsY3pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTnl1CldVdWQxNEJjcFgvbU4xQ3dtUnZzRDJyVFF5aGdOQ3FDTGQ5M2VVVzRqbHJCQ0pCUGkzcE9mOXNTVFJELzV3V0YKUUpzOTE5eVZuRmdOOXBHVVRHbjVieHVtNzgrS3lvY20xTnJ3Z1kxUUpnWlZxVUZhQ1I5VDJ1RThBM1lyKzdITAovS3FHMEZ3S05UV0w3Zy85VFAzdmpQMW5XR3NTTElmVHAxMjNaK0lxZHJaQlI0NUZDQzBOQzU4cmxEYUErVFdOCjRVQ2xJalBKRHJuV2M1Z2E4a3NVYXN4UkQ5clR2dm5iOG05V2NEYnhXdEViTlJyS0R6cUp5K00wT1BacDdIY1QKWkhHUnlXTVR3WVhRVGlmWFhxVTY1U2VkdEdjT3ZsOHl0ck1UZndCRkh2Y1NWQW9sNytYUnNicSs4bDRKc0M5OApPN0tOODAxT0dvaUFyWWNFYjRjQ0F3RUFBYU1qTUNFd0RnWURWUjBQQVFIL0JBUURBZ0trTUE4R0ExVWRFd0VCCi93UUZNQU1CQWY4d0RRWUpLb1pJaHZjTkFRRUxCUUFEZ2dFQkFLcFpoN0t5ZzRLM0dyK0NZRzdwbVlXUGR1eFUKMXcrY1BPZ053OE83MHNFWE5lbnFMbEZVdThtYjMxQTZ4RHZKWHZ4OXNiQ0o5YzdDeDFiaVNNYnFrWlBHRGs4UgpZSm5wZlB4WXBUWlBISmFkK1ZCb2tVY0J3QlluTTB1SXFzZGhvbHhPelVBUU9UbVo5M3UzSi9MeUNob3IwSmlJCm1uQ1hlaHlDaTZ1YTVvTldXNmVNWnlaRWxzRnpXcnlGdHkyNGRKVWkvQkd0SjR2ZStlRmtLTE9VTXpnMjBBU3YKZU1ldkdUL1FibWd1YXZhT1RCdjVDYW05RElSNEZSd2YvV2hwdHpMTVdCVFJXb2ROOUhkcE1tRDhxUmU1YnVWNgpiZ245VFgzejBncUhGMzFQZlBjcTJWRFRJWnJZK084MTBwOTQrbTA5YmxrcTQzSDdXQmhGTmdvditzUT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo=
    server: https://172.31.244.238:6443
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
current-context: kubernetes-admin@kubernetes
kind: Config
preferences: {}
users:
- name: kubernetes-admin
  user:
    token: eyJhbGciOiJSUzI1NiIsImtpZCI6ImJjOTU0NjdlM2I0OTE5YWE1OTEzZDNkMDU3NGM2ZTRjYjBjY2NhNzgifQ.eyJpc3MiOiJodHRwczovLzQ3LjUyLjE5Ny4xNjM6MzIwMDAiLCJzdWIiOiJDZ2M0T1RFeU5UVTNFZ1puYVhSb2RXSSIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxNTIzOTY3NDQzLCJpYXQiOjE1MjM4ODEwNDMsImF0X2hhc2giOiJMUzNKUVpiWDVuVnBuam5zSU5nNGZnIiwiZW1haWwiOiJmaHRqb2JAaG90bWFpbC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6InN0ZXZlbiJ9.KjKSkqqX1I21rkqF4t39x8YmEFx2yPlQSMFInVeAp4lCRACljMvTY07GSWycEez0SarPtO80dTqcM4buz7WMVPMRuSqg-HuCPB3DjzD4M84OiHZSFB_5xOJIUqP0dWLAuPTalu2T-le4Gp0gPXc863YfLEMzRm8cxbvdASwQrTZ5oKgoRVznDREW3NIgEONUU9A64bBeWi5xH1eyCbvh4l3Q-ZfkYG4A4w46FwAmfL4ClxCBiIkpZWhKv5GcN8bg7-msaNlvlejpvbSuVWpt5CLJzpCXHh1AqCUBkXzp8ObSGGIw1BfkVFnyH26bpho2kAzxbGtdwNx4TdGlu_XYlw
```
注意把user那的client-certificate-data client-key-data 删掉，加上token, 我这直接在/etc/kubernetes/admin.conf上修改的，也可以重新生成配置文件：

```
kubectl config set-credentials fanux \
--client-certificate=/etc/kubernetes/pki/ca.crt \
--client-key=/etc/kubernetes/pki/ca.key \
--token=eyJhbGciOiJSUzI1NiIsImtpZCI6ImJjOTU0NjdlM2I0OTE5YWE1OTEzZDNkMDU3NGM2ZTRjYjBjY2NhNzgifQ.eyJpc3MiOiJodHRwczovLzQ3LjUyLjE5Ny4xNjM6MzIwMDAiLCJzdWIiOiJDZ2M0T1RFeU5UVTNFZ1puYVhSb2RXSSIsImF1ZCI6ImV4YW1wbGUtYXBwIiwiZXhwIjoxNTIzOTYyNjUyLCJpYXQiOjE1MjM4NzYyNTIsImF0X2hhc2giOiJFUXRWWm5ObE50c2hhWERfZ3N2UkNBIiwiZW1haWwiOiJmaHRqb2JAaG90bWFpbC5jb20iLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6InN0ZXZlbiJ9.vu0keGMoRGg6OAYpMZNN9zm4pnKXGyXDkZaRNj6MXDY9XsfnBDT4HnXkY17Lvm1ow0xPbq9cgVL3JBZT73jiddgFNAIXJffHfPejlVRSqXx9iF1uEcNIc5tDA1hUPtBrX8n_rzdz0sZsPMb4ZYMx3AdEylszpVrS_OelbB4I_2eLfO0KzwcEknOgV8cZZghCCITl6ZTOeeWEv5FPvJjRC2rpu_MkSY5tAf30SITwldFUMgF8ei3aPrZdojPLgqUWtxKaDmPpcHVLhYr0sLE_BnDZLjGP4ff8l5yy_EfDc7sQsrJR7StwZXRnK-n2omqaV3z-n5IxaUty85e_97FA1g \
--embed-certs=true \
--kubeconfig=fanux.config

kubectl config set-context kubernetes \
--cluster=kubernetes \
--user=fanux \
--namespace=default \
--kubeconfig=fanux.config

kubectl config use-context kubernetes --kubeconfig=fanux.config

kubectl config set-cluster kubernetes --server=https://172.31.244.238:6443 --certificate-authority=/etc/kubernetes/pki/ca.key  --kubeconfig=fanux.config
```

验证：
```
$ kubectl get pod #正常
```
删除角色绑定再执行get pod
```
[root@master2 dex]# kubectl delete -f examples/k8s/role.yaml
clusterrolebinding.rbac.authorization.k8s.io "read-secrets-global" deleted
[root@master2 dex]# kubectl get pod
Error from server (Forbidden): pods is forbidden: User "fhtjob@hotmail.com" cannot list pods in the namespace "default"
```
已经无权限了。
至于给用户分配更细的权限，比较简单，读者门自己倒持去吧
