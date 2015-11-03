package main

import (
    "log"

    "k8s.io/kubernetes/pkg/api"
    client "github.com/kubernetes/kubernetes/pkg/client/unversioned"
    "k8s.io/kubernetes/pkg/labels"
    "k8s.io/kubernetes/pkg/fields"
    "k8s.io/kubernetes/pkg/watch"

    "github.com/sroze/kubernetes-load-balancer-proxifier/reverseproxy"
    "encoding/json"
    "os"
    "strings"
)

func main() {
    clusterAddress := os.Getenv("CLUSTER_ADDRESS")
    rootDns := os.Getenv("ROOT_DNS_DOMAIN")
    if rootDns == "" {
        log.Fatalln("You need to precise your root DNS name with the `ROOT_DNS_DOMAIN` environment variable")
    }

    config := client.Config{
        Host: clusterAddress,
        Insecure: os.Getenv("INSECURE_CLUSTER") == "true",
    }
    
    c, err := client.New(&config)
    if err != nil {
        log.Fatalln("Can't connect to Kubernetes API:", err)
    }

    w, err := c.Services(api.NamespaceAll).Watch(labels.Everything(), fields.Everything(), api.ListOptions{})
    if err != nil {
        log.Fatalln("Unable to watch services:", err)
    }

    log.Println("Watching services")
    for event := range w.ResultChan() {
        service, ok := event.Object.(*api.Service)
        if !ok {
            log.Println("Got a non-service object")

            continue
        }

        if event.Type == watch.Added || event.Type == watch.Modified {
            err := ReviewService(c, service, rootDns)

            if err != nil {
                log.Println("An error occured while updating service")
            }
        }
    }
}

func ReviewService(client *client.Client, service *api.Service, rootDns string) (error) {
    if service.Spec.Type != api.ServiceTypeLoadBalancer {
        log.Println("Skipping service", service.ObjectMeta.Name, "as it is not a LoadBalancer")

        return nil
    }

    // If there's an IP and/or DNS address in the load balancer status, skip it
    if ServiceHasLoadBalancerAddress(service) {
        log.Println("Skipping service", service.ObjectMeta.Name, "as it already have a LoadBalancer address")

        return nil
    }

    log.Println("Service", service.ObjectMeta.Name, "needs to be reviewed")

    // Get existing proxy configuration
    var proxyConfiguration reverseproxy.Configuration
    if jsonConfiguration, found := service.ObjectMeta.Annotations["kubernetesReverseproxy"]; found {
        proxyConfiguration := reverseproxy.Configuration{}

        if err := json.Unmarshal([]byte(jsonConfiguration), &proxyConfiguration); err != nil {
            log.Println("Unable to unmarshal the configuration, keep the empty one")
        }
    } else {
        proxyConfiguration = reverseproxy.Configuration{}
    }

    // If configuration found, skip it
    if len(proxyConfiguration.Hosts) == 0 {
        // Create the expected hostname of the service
        host := strings.Join([]string{
            service.ObjectMeta.Name,
            service.ObjectMeta.Namespace,
            rootDns,
        }, ".")

        // Append the new host to the configuration
        proxyConfiguration.Hosts = append(proxyConfiguration.Hosts, reverseproxy.Host{
            Host: host,
            Port: 80,
        })

        jsonConfiguration, err := json.Marshal(proxyConfiguration)
        if err != nil {
            log.Println("Unable to JSON-encode the proxy configuration: ", err)

            return err
        }

        if service.ObjectMeta.Annotations == nil {
            service.ObjectMeta.Annotations = map[string]string{}
        }

        service.ObjectMeta.Annotations["kubernetesReverseproxy"] = string(jsonConfiguration)

        // Update the service
        log.Println("Adding the `kubernetesReverseproxy` annotation to service and the loadbalancer status")
        updated, err := client.Services(service.ObjectMeta.Namespace).Update(service)
        if err != nil {
            log.Println("Error while updated the service:", err)

            return err
        }

        log.Println("Successfully added the reverse proxy configuration", updated)
    } else {
        // Updating service load-balancer status
        log.Println("Updating the service load-balancer status")
        service.Status = api.ServiceStatus{
            LoadBalancer: api.LoadBalancerStatus{
                Ingress: []api.LoadBalancerIngress{
                    api.LoadBalancerIngress{
                        Hostname: proxyConfiguration.Hosts[0].Host,
                    },
                },
            },
        }

        updated, err := client.Services(service.ObjectMeta.Namespace).Update(service)
        if err != nil {
            log.Println("Error while updated the service:", err)

            return err
        }

        log.Println("Successfully updated the service status", updated)
    }

    return nil
}

func ServiceHasLoadBalancerAddress(service *api.Service) bool {
    if len(service.Status.LoadBalancer.Ingress) == 0 {
        return false
    }

    for _, ingress := range service.Status.LoadBalancer.Ingress {
        if ingress.IP != "" || ingress.Hostname != "" {
            return true
        }
    }

    return false
}
