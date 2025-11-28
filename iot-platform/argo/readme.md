ArgoCD访问地址：https://ad5bef3262ea0487fa4ce74498d336b2-607992651.us-east-1.elb.amazonaws.com
登录凭据：

用户名：admin
密码：jByFZxoMmDbHwjS-


kubectl get svc -n monitoring prometheus-stack-grafana
NAME                       TYPE           CLUSTER-IP      EXTERNAL-IP                                                              PORT(S)
   AGE
prometheus-stack-grafana   LoadBalancer   172.20.156.58   a91c8df1f1b2544149b02c7d4c93177e-528337716.us-east-1.elb.amazonaws.com   80:30786/TCP
   2m13s
   
   
   
基本信息:
docker --version
aws --version
kubectl version --client
Docker version 28.5.1, build e180ab8
aws-cli/2.32.5 Python/3.13.9 Windows/10 exe/AMD64
Client Version: v1.34.1
Kustomize Version: v5.7.1

Administrator@WIN-20241127NBZ MINGW64 /d/code2025/eks/iot-platform/mqtt-kafka-bridge (main)
$ aws configure
AWS Access Key ID [****************GUHF]:
AWS Secret Access Key [****************RRLv]:
Default region name [us-east-1]:
Default output format [json]:


---
4bafb949f16: Pushed
ddafe03b37da: Pushed
latest: digest: sha256:474148bcfc85e1e3068871a517b74e804f9132d6129c0c97178b14e68e1eb1d0 size: 856
✓ 镜像推送完成
📝 [5/5] 更新 Kubernetes 配置...
✓ 配置更新完成

=========================================
✅ 部署准备完成！
=========================================

📋 镜像信息:
   Repository: 645890933537.dkr.ecr.us-east-1.amazonaws.com/mqtt-kafka-bridge
   Tag: latest

📝 下一步操作:

1. 编辑 ArgoCD 配置文件:
   vim deployments/kubernetes/argocd-app.yaml
   修改 spec.source.repoURL 为你的 Git 仓库地址

2. 提交到 Git:
   git add .
   git commit -m 'Add MQTT-Kafka Bridge'
   git push origin main

3. 部署到 EKS:
   kubectl apply -f deployments/kubernetes/argocd-app.yaml

4. 查看状态:
   kubectl get pods -n iot-bridge
   kubectl logs -f -n iot-bridge -l app=mqtt-kafka-bridge
