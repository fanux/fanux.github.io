## calico网络策略
使用kubernetes NetworkPolicy让用户定义pod之间的访问策略，精细的控制哪些pod之间有相互访问的权利，如此网络更安全.

## 教程流程
* 创建nginx service
* 禁止所有入口流量
* 允许向内访问nginx
* 禁止所有出口流程
* 允许出口流量访问kube-dns

## 创建nginx service
```
kubectl create ns advanced-policy-demo
kubectl run --namespace=advanced-policy-demo nginx --replicas=2 --image=nginx
kubectl expose --namespace=advanced-policy-demo deployment nginx --port=80
``` 

现在nginx是完全可以被访问到的：
```
kubectl run --namespace=advanced-policy-demo access --rm -ti --image busybox \
wget -q --timeout=5 nginx -O -
```

## 禁止入口流量
```
kubectl create -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-ingress
  namespace: advanced-policy-demo
spec:
  podSelector:
    matchLabels: {}
  policyTypes:
  - Ingress
EOF
```

再去访问：
```
kubectl run --namespace=advanced-policy-demo access --rm -ti --image busybox \
wget -q --timeout=5 nginx -O -
wget: download timed out
```

## 允许所有pod访问nginx
```
kubectl create -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: access-nginx
  namespace: advanced-policy-demo
spec:
  podSelector:
    matchLabels:
      run: nginx  # 目标pods，给这些pods配置策略
  ingress:
    - from:
      - podSelector:
          matchLabels: {} # 允许所有pod访问nginx
EOF
```

```
kubectl run --namespace=advanced-policy-demo access --rm -ti --image busybox \
wget -q --timeout=5 nginx -O -
```

## 禁止所有出口流量
```
kubectl create -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-egress
  namespace: advanced-policy-demo
spec:
  podSelector:
    matchLabels: {}
  policyTypes:
  - Egress
EOF
```
这时就无法访问sina什么的了，也无法访问别的pod

在 busybox里面：
```
/ # nslookup nginx
Server:    10.96.0.10
Address 1: 10.96.0.10


nslookup: can't resolve 'nginx'
/ # wget -q --timeout=5 sina.com -O -
wget: bad address 'google.com'
```

## 允许访问DNS
因为DNS跑在kube-system这个namespace下，所以先给这个namespace贴个标签，
然后通过namespaceSelector选到这个namespace,允许本namespace下的pod访问kube-system下面的pod

```
kubectl label namespace kube-system name=kube-system
kubectl create -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-dns-access
  namespace: advanced-policy-demo
spec:
  podSelector:
    matchLabels: {}
  policyTypes:
  - Egress
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: UDP
      port: 53

EOF
```

如此busybox里面就能发现nginx了
```
/ # nslookup nginx
Server:    10.0.0.10
Address 1: 10.0.0.10 kube-dns.kube-system.svc.cluster.local
```

## 允许外部流量访问nginx
```
kubectl create -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-egress-to-advance-policy-ns
  namespace: advanced-policy-demo
spec:
  podSelector:
    matchLabels: {}
  policyTypes:
  - Egress
  egress:
  - to:
    - podSelector:
        matchLabels:
          run: nginx
EOF
```
这时我们的测试pods只能联通带有 run:nginx标签的pod，外部的DNS就无法访问了

## 删除所有的namespace
```
kubectl delete ns advanced-policy-demo
```
