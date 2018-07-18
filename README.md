# Syringe

# Resources

https://blog.csnet.me/blog/building-a-go-api-grpc-rest-and-openapi-swagger.1/

https://github.com/grpc-ecosystem/grpc-gateway

Features:
- lab resources
- persistence
- scheduling (knowing when to configure what)
- API - requesting lab info


The infratructure services within antidote run as you would expect. Pods, services, ingresses. Front-ended by a GCE load balancer for port translation. These are almost all web-based services so we can stick with the native Kubernetes workflow for managing these.

The lab stuff is a bit different. We need a way to provision labs that goes a bit against what Kubernetes is good at. For instance, a "lab" for us is typically made up of several pods. These pods need to be managed as a group, but not in the traditional replicaset way, as each pod in a lab will be different. We'll also need to manage how they're networked. We also need an abstraction that a) provides a way for people to simply specify the parameters of a lab without getting into the details of pod and service definitions (and also provide a way for us to deploy labs on something other than kubernetes in the future).

Also we may want to do some preprovisioning

Syringe provides this functionality. It will accept lab definitions and take care of scheduling them

This will be a public-facing API, so it needs to be secured, and rate-limited. Need to track RPS as well as unique users (either via cookie and/or IP)