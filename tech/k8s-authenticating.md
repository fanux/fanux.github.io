# kubernetes对接第三认证
广告： [安装包地址](https://market.aliyun.com/products/57742013/cmxz025618.html?spm=5176.730005.0.0.TFKV5K#sku=yuncode1961800000) 
[原文地址]()

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

## 环境介绍
采用云服务器进行该实验，Floatingip是
