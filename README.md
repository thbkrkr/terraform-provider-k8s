# Terraform Kubernetes Provider

That handles directories of raw Kubernetes resources manifests.

```hcl
provider "k8s" {
  kubeconfig = "c1/kubectl.conf"
}

resource "k8s_resources" "myapp" {
  dir = "./manifests"
}

```


Rely on `kubectl apply|get|delete -f <dir>`.

Strongly inspired by [ericchiang/terraform-provider-k8s](https://github.com/ericchiang/terraform-provider-k8s).