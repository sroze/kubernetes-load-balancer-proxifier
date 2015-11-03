# Kubernetes Load Balancer Proxifier

When dealing with your own Kubernetes Cluster (not hosted in GCE or AWS) you will have trouble to exposes automatically
public services. With these cloud providers, we simply have to create a `LoadBalancer` service and just wait a bit to
have an address automatically added in the status of the service.

This _load-balancer profixier_ brig this awesome workflow to self-hosted Kubernetes cluster, by automatically create
[kubernetes-reverse-proxy](https://github.com/darkgaro/kubernetes-reverseproxy) configurations for `LoadBalancer`
services that do not have this configuration. Once the configuration is created, it updates the service status to
add the expected DNS address.

## Getting started

First of all, **if not already did** setup [kubernetes-reverse-proxy](https://github.com/darkgaro/kubernetes-reverseproxy) on one or more of
your cluster nodes. The easiest way is to simply run the following command:

```
docker run -d -t \
    -e CONFD_ETCD_NODE=[YOUR-ETCD-IP]:4001 \
    -p 80:80 \
    --restart=always \
    --name=reverse-proxy \
    darkgaro/kubernetes-reverseproxy
```

Then, start the profixier on one of your nodes (the master might be the more relevant one):
```
docker run -d \
    --restart=always \
    --name=load-balancer-proxifier \
    -e ROOT_DNS_DOMAIN=any.wildcarded.dns.address \
    -e CLUSTER_ADDRESS=https://username:password@your.master.cluster.address \
    -e INSECURE_CLUSTER=true \
    sroze/kubernetes-load-balancer-proxifier
```

## How is it working

The _proxifier_ is listening on all events related to services. Once one is created or updated, if no
`kubernetesReverseproxy` annotation is found, it'll automatically create one based on the root DNS domain you choose
and send this created DNS address back to the service status.
