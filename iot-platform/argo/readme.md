ArgoCD访问地址：https://ad5bef3262ea0487fa4ce74498d336b2-607992651.us-east-1.elb.amazonaws.com
登录凭据：

用户名：admin
密码：jByFZxoMmDbHwjS-


kubectl get svc -n monitoring prometheus-stack-grafana
NAME                       TYPE           CLUSTER-IP      EXTERNAL-IP                                                              PORT(S)
   AGE
prometheus-stack-grafana   LoadBalancer   172.20.156.58   a91c8df1f1b2544149b02c7d4c93177e-528337716.us-east-1.elb.amazonaws.com   80:30786/TCP
   2m13s
