# test Rokku Sts operator

# building 

```
$ operator-sdk build docker.io/your_id/sts-operator

$ operator-sdk generate k8s
$ operator-sdk generate crds
```

# installing 

```
$ kubectl create -f deploy/service_account.yaml 
$ kubectl create -f deploy/role.yaml 
$ kubectl create -f deploy/role_binding.yaml 
$ kubectl create -f deploy/crds/rokku-sts.wbaa.ing.com_rokkusts_crd.yaml 
$ kubectl create -f deploy/operator.yaml 
$ kubectl create -f deploy/crds/rokku-sts.wbaa.ing.com_v1alpha1_rokkusts_cr.yaml
```