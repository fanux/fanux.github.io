# iptables教程

## 内核空间中的五个包处理位置，和五个函数钩子（规则链）
* PREROUTING 数据包刚进入网络层 , 路由之前
* INPUT 路由判断，流入用户空间
* OUTPUT 用户空间发出，后接路由判断出口的网络接口
* FORWARD 路由判断不进入用户空间，只进行转发
* POSTROUTING 数据包通过网络接口出去

```
                      应用层
                   -------------
                    ^        |
                    |        v
                  INPUT    OUTPUT
                    |        |
-->PREROUTING-------->FORWARD--->POSTROUTING---> 
```
这就是五个内置链，可以在链里面添加规则

## 四个表来定义区分各种不同功能和处理方式
表可以作用在多个链上，同样一个链也可以配置多个表

* Filter表 一般的数据包过滤
* Nat表 网络地址转换
* Mangle表 修改数据包的原数据，一般用于防火墙标记
* raw表 用于配置免除

chain/table|Filter | Nat | Mangle | Raw
-----------|-------|-----|--------|---
PREROUTING | false | true| true   | true
INPUT      | true  | false| true  | false
FORWARD    |true   | false| true  |false
OUTPUT     |true   |true  |true   |true
POSTROUTING |false | true|true    |false

## 创建一个自定义链
```
iptables -t filter -N newchain # 创建链
iptables -t filter -A newchain -s 192.168.75.9 -j DROP # 往链中添加规则
iptables -A INPUT -j newchain # 创建的链在INPUT链中生效,创建的链往哪接
```

## 命令结构
```
iptables [-t table]  # 指定表名
         command     # 对链操作命令
         [chain]     # 链名
         [rules]     # 规则，包是否匹配该条规则
         [-j target] # 符合规则的数据包采取什么动作
```

## neutron中的自定义链
