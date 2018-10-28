# kubediscovery

A Kubernetes Aggregated API Server to retrieve dynamic composition tree of Kubernetes resources/kinds in your cluster.


## What is it?

kubediscovery is a tool that helps you find dynamic composition trees of Kubernetes Objects.
In Kubernetes there are top-level resources which are composed of other resources.
For example, a Deployment is composed of a ReplicaSet which in turn is composed of one or more Pods. 
kubediscovery is a Kubernetes Aggregated API Server that helps you find 
the entire composition trees of Kubernetes objects in your cluster.


## How it works?

You provide a YAML file that defines static composition relationship between different Resources/Kinds.
Using this information kubediscovery API Server builds the dynamic composition trees by 
continuously querying the Kubernetes API for various Objects that are created in your cluster.

The YAML file can contain both in-built Kinds (such as Deployment, Pod, Service), and
Custom Resource Kinds (such as Postgres or EtcdCluster). An example YAML file is provided (kind_compositions.yaml). There is also kind_compositions.yaml.with-etcd which shows definition for the EtcdCluster custom resource. Use this YAML only after you deploy the [Etcd Operator](https://github.com/coreos/etcd-operator) (Rename this file to kind_compositions.yaml before deploying the API server).

kubedsicovery API server registers following REST endpoint in your cluster:
`/apis/kubeplus.cloudark.io/v1/describe`

kubediscovery supports two query parameters: `kind` and `instance` on this endpoint.

To retrieve dynamic composition tree for a particular Kind you would use following call:

```kubectl get --raw "/apis/kubeplus.cloudark.io/v1/describe?kind=Deployment&instance=nginx-deployment```

The value for `kind` query parameter should be the exact name of a Kind such as 'Deployment' and not 'deployment' or 'deployments'.

The value for `instance` query parameter should be the name of the instance. 
A special value of `*` is supported for the `instance` query parameter to retrieve 
composition trees for all instances of a particular Kind.

The dynamic composition information is currently collected for the "default" namespace only.
The work to support all namespaces is being tracked [here](https://github.com/cloud-ark/kubediscovery/issues/16).

Constructed dynamic composition trees are currently stored in memory.
If the kubediscovery pod is deleted this information will be lost.
But it will be recreated once you redeploy kubediscovery API Server.

In building this API server we tried several approaches. You can read about our experience  
[here](https://medium.com/@cloudark/our-journey-in-building-a-kubernetes-aggregated-api-server-29a4f9c1de22).


## How is it different than..

```
kubectl get all
```

1) Using kubediscovery you can find dynamic composition trees for native Kinds and Custom Resources alike.

2) You can find dynamic composition trees for a specific object or all objects of a particular Kind.


## Try it:

Download Minikube
- You will need VirtualBox installed
- Download appropriate version of Minikube for your platform
- Kubediscovery has been tested with Minikube-0.25 and Minikube-0.28.
  It should work with other versions as well. Please file an Issue if it does not.

1) Start Minikube

2) Deploy the API Server in your cluster:

   `$ ./deploy-discovery-artifacts.sh`

3) Check that API Server is running:

   `$ kubectl get pods -n discovery`    

4) Deploy Nginx Pod:

   `$ kubectl apply -f nginx-deployment.yaml`


5) Get dynamic composition for nginx deployment

```
kubectl get --raw "/apis/kubeplus.cloudark.io/v1/describe?kind=Deployment&instance=nginx1-deployment" | python -mjson.tool
```

![alt text](https://github.com/cloud-ark/kubediscovery/raw/master/docs/nginx1-deployment.png)


6) Get dynamic composition for all deployments

```
kubectl get --raw "/apis/kubeplus.cloudark.io/v1/describe?kind=Deployment&instance=*" | python -mjson.tool
```

![alt text](https://github.com/cloud-ark/kubediscovery/raw/master/docs/all-dep-1.png)


7) Get dynamic composition for all replicasets

```
kubectl get --raw "/apis/kubeplus.cloudark.io/v1/describe?kind=ReplicaSet&instance=*" | python -mjson.tool
```

![alt text](https://github.com/cloud-ark/kubediscovery/raw/master/docs/all-replicasets.png)


8) Get dynamic composition for all pods

```
kubectl get --raw "/apis/kubeplus.cloudark.io/v1/describe?kind=Pod&instance=*" | python -mjson.tool
```

![alt text](https://github.com/cloud-ark/kubediscovery/raw/master/docs/all-pod.png)


9) Delete nginx deployment

```
kubectl delete -f nginx-deployment.yaml
```

10) Try getting dynamic compositions for various kinds again (repeat steps 5-8)


You can use above style of commands with all the Kinds that you have defined in kind_compositions.yaml



## Development

1) Start Minikube 

2) Allow Minikube to use local Docker images: 

   `$ eval $(minikube docker-env)`

3) Install/Vendor in dependencies:

   `$ dep ensure`

4) Build the API Server container image:

   `$ ./build-local-discovery-artifacts.sh`

5) Deploy the API Server in your cluster:

   `$ ./deploy-local-discovery-artifacts.sh`

6) Follow steps 3-10 listed under Try it section above.




## Troubleshooting tips:

1) Check that the API server Pod is running: 

   `$ kubectl get pods -n discovery`

2) Get the Pod name from output of above command and then check logs of the container.
   For example:

   `$ kubectl logs -n discovery kube-discovery-apiserver-kjz7p  -c kube-discovery-apiserver`


### Clean-up:

  `$ ./delete-discovery-artifacts.sh`



### Issues/Suggestions:

Issues and suggestions for improvement are welcome. 
Please file them [here](https://github.com/cloud-ark/kubediscovery/issues)



