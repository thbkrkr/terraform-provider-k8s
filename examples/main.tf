resource "k8s_resources" "apish" {
  config = "/ops/clusters/c1/kubectl.cfg"
  dir = "./manifests"
}