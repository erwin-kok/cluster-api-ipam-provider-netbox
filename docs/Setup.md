# Project Setup

The project was setup using the following commands:

```shell
go mod init github.com/erwin-kok/cluster-api-ipam-provider-netbox

kubebuilder init --domain cluster.x-k8s.io --repo github.com/erwin-kok/cluster-api-ipam-provider-netbox --project-name cluster-api-ipam-provider-netbox

kubebuilder create api --group ipam --version v1alpha1 --kind NetboxIPPool --controller=true --resource=true
kubebuilder create webhook --group ipam --version v1alpha1 --kind NetboxIPPool --programmatic-validation --defaulting

kubebuilder create api --group ipam --version v1alpha1 --kind NetboxGlobalIPPool --controller=true --resource=true
kubebuilder create webhook --group ipam --version v1alpha1 --kind NetboxGlobalIPPool --programmatic-validation --defaulting
```
