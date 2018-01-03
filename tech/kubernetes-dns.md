## dns之锅
```
kubectl run --namespace=kube-system access -it --image busybox -- /bin/sh
[root@fortest1513671663-master-00 ~]# kubectl exec access-79f4758b79-qwl8s nslookup kubernetes-dashboard.kube-system.svc -n kube-system
Server:    10.96.0.10
Address 1: 10.96.0.10 kube-dns.kube-system.svc.cluster.local

Name:      kubernetes-dashboard.kube-system.svc
Address 1: 10.110.146.248 kubernetes-dashboard.kube-system.svc.cluster.local
[root@fortest1513671663-master-00 ~]# kubectl get svc kubernetes-dashboard -n kube-system
NAME                   TYPE       CLUSTER-IP       EXTERNAL-IP   PORT(S)         AGE
kubernetes-dashboard   NodePort   10.110.146.248   <none>        443:30089/TCP   27m
```
