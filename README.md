# kubernetes-lab

A quick little proof of concept that lets you deploy a Redis-backed CRUD API on a Kubernetes cluster. Inspired by dishbreak/docker-compose-lab.

# Getting Started

## Before You Begin

Make sure you've downloaded and installed Docker Desktop and enabled Kubernetes. You can follow an official guide [here](https://docs.docker.com/desktop/kubernetes/).

The API in this lab is written in Go, but don't stress, you don't need it installed!

**This is not a production ready system**. You _could_ deploy all this on GKE or EKS, but I wouldn't vouch for it. If it makes you money, do yourself a favor and read up on productionizing k8s before deploying app.

# Step 1: Create a Namespace

_You've been tasked with adding value to your enterprise. So, naturally, you'll create a Value service that will set and retrieve an integer value. That's what they meant, right?_

To begin, let's checkout code at the start.

```
git checkout 1-my-first-k8s-svc
```

We're going to create a namespace for our Value service. This will let us keep all our pods and services nicely organized. We've got a namespace definition ready to go, let's take a look!

```shell
$ cat namespace.yml 
apiVersion: v1
kind: Namespace
metadata:
  name: value
```

There's not much to namespaces--they're really made to be referenced in other objects, as we'll see in a bit. Let's use `kubectl` to apply this file against our API server.

```shell
$ kubectl apply -f namespace.yml 
namespace/value created
```

Sweet! Let's check the API to see the namespace.

```shell
$ kubectl describe namespace value
Name:         value
Labels:       kubernetes.io/metadata.name=value
Annotations:  <none>
Status:       Active

No resource quota.

No LimitRange resource.
```

Sweet, we've now shown that we can interact with the API server and create Kubernetes objects!

# Step 2: Try out the Value API

Check out the source code in `api/`. It might be hard to parse if you don't know Go, but here's what you need to know about the service:

* It listens on port 8080
* POSTing an integer value to `/value` will set the value
* GETting `/value` will retrieve the value.

Let's try it out. Run the `docker build` command in the `api/` directory.

```shell
$ docker build -t value-api:v1
```

It'll take a minute to download everything. When it's done, go ahead and execute it.

```shell
$ docker run -P value-api:v1
2022/10/05 21:33:10 ready to listen
```

The `-P` flag publishes ports--that is, it automatically binds any ports named with the `EXPOSE` command within the Dockerfile to high-order ports on your machine. Let's figure out what port to use by running `docker ps` and checking the `PORTS` section.

```
$ docker ps  
CONTAINER ID   IMAGE              COMMAND   CREATED         STATUS         PORTS                     NAMES
a8df1c2e7068   value-service:v1   "app"     2 minutes ago   Up 2 minutes   0.0.0.0:55001->8080/tcp   serene_lehmann
```

So, on my machine, the container bound Port 8080 in the container to to port 55001 on the host. Your port will almost certainly be different.

Let's interact with this value service. In a separate terminal, let's post a value to the endpoint.

```
$ curl -X POST localhost:55001/value -d 343
```

Great. Now let's check to see if we can get that value back.

```
$ curl localhost:55001/value       
343
```

Awesome! We're ready to deploy this container.


# Step 3: Create a deployment

Our Kubernetes app will run on [pods](https://kubernetes.io/docs/concepts/workloads/pods/), where each pod is a set of Docker containers that are networked together.

We don't manage pods directly, though--we let the cluster handle that. Instead, we create a [deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) that will manage our pods for us. 

Take a look at `api/deployment.yml`. The YAML syntax is a little verbose, but there are some important things to note.

* This deployment is in our `value` namespace, according to lines 3-5
* The deployment manages any pods tagged with the `app: api` label (lines 7-9)
* The deployment ensures that any pods it launch get tagged with the `app: api` label (lines 10-13)
* The `api` container within the pod uses the Docker image we just built (line 17)
* The `api` container uses port 8080 (line 23)
* The `IfNotPresent` pull policy allows us to use containers without publishing them to a registry (line 24)

Alright, let's deploy this, uh, deployment. From the repo root:

```
$ kubectl apply -f api/deployment.yml 
deployment.apps/api created
```

Now let's go find it!

```
$ kubectl get deployments
No resources found in default namespace.
```

Uhhh...what? We just created it, right? Where is it? Remember that we created the deployment in our `value` namespace. In order to see it via `kubectl`, we need to pass the `--namespace` flag.

```
$ kubectl get deployments --namespace=value
NAME   READY   UP-TO-DATE   AVAILABLE   AGE
api    1/1     1            1           24m
```

Of course, your `AGE` value may differ, but there it is! One pod, ready and available for value-add goodness. But, um...how are we going to reach it?


# Step 4: Create a Service

Deployments are essential, we need them to ensue that the cluster is running pods with our desired container image and config. But if we want to form a network connection to them, we'll need to set up a Kubernetes [Service](https://kubernetes.io/docs/concepts/services-networking/service). Per the docs:

> In Kubernetes, a Service is an abstraction which defines a logical set of Pods and a policy by which to access them (sometimes this pattern is called a micro-service). The set of Pods targeted by a Service is usually determined by a selector. 

Specifically, we'll use a service to make our API reachable outside the cluster. We've defined an api service in `api/service.yml`. Take a look, and note the following:

* The service is in the `value` namespace (line 5)
* The service is a `NodePort` service -- more on this later (line 7)
* The service selects pods tagged with the label `app: api` (lines 8-9)
* The service routes traffic sent to the node on port 31000 to the container's port 31000

The selector is crucial here. It lets us make the pods that our deployment creates turn into endpoints for our service.

Let's deploy it, and then describe the service.

```shell
$ kubectl apply -f api/service.yml  
service/api created

$ kubectl describe service api --namespace=value
Name:                     api
Namespace:                value
Labels:                   <none>
Annotations:              <none>
Selector:                 app=api
Type:                     NodePort
IP Family Policy:         SingleStack
IP Families:              IPv4
IP:                       10.110.233.21
IPs:                      10.110.233.21
LoadBalancer Ingress:     localhost
Port:                     traffic  8080/TCP
TargetPort:               8080/TCP
NodePort:                 traffic  31000/TCP
Endpoints:                10.1.0.18:8080
Session Affinity:         None
External Traffic Policy:  Cluster
Events:                   <none>
```

Awesome! The `Endpoints` shows the Pod IP address for the pod we created with our deployment. Now, let's see if we can visit the service.

```shell
$ curl -X POST localhost:31000/value -d 457
$ curl localhost:31000/value
457
```

Nice! We've got the same behavior that we saw locally, but inside our cluster. Now let's 

# Step 5: Add Redis backing

There's a small problem with our service. To demonstrate, let's use the `rollout` command.

```shell
$ kubectl rollout restart deployment api --namespace=value
deployment.apps/api restarted
```

Don't let the `restart` fool you. What's happening here is the deployment is killing and replacing the pod with a brand new one. Try getting the value again, and you'll see an effect of this:

```shell
$ curl localhost:31000/value                              
0
```

Hm. The value we POSTed back in Step 4 is gone! That's because the value lived in memory within the pod that we just terminated. The value is gone with that pod. If we're going to want to keep our value, we'll need to persist it somewhere.

Let's step forward in time, and deploy a Redis service that will handle persisting data for us.

```shell
$ git checkout 2-using-redis-backend
```

Your repo now has a `redis/` directory. This directory has YAML files but no source code, because it's relying on the public redis docker image. Take a look at the files, and then apply them to the kubernetes cluster. 

```shell
$ kubectl apply -f ./redis
deployment.apps/redis created
service/redis created
```

Your API service got an upgrade, too! It now can store the value in Redis instead of keeping it memory. Check out `api/controller/value.go` to see the details on that. In the `api/` directory, run `docker build` to create a new container image.

```shell
$ docker build -t value-api:v2 .
```

This build should go much faster than the first one, since most of the image layers are cached.

Finally, check out `api/deployment.yml`. You should see a new piece of config in there:

```yml
        env:
          - name: USE_REDIS_BACKEND
            value: "1"
```

This sets the `USE_REDIS_BACKEND` environment variable in your container, which configures the API to use a Redis client to store and retrieve the data. Go ahead and apply the file against your cluster.

```shell
$ kubectl apply -f api/deployment.yml
deployment.apps/api configured
```

Now, let's try the same sequence again: POST a value, GET the value, restart the pod, and GET the value again.

```shell
$ curl -X POST localhost:31000/value -d 457

$ curl localhost:31000/value               
457                                                                                                                                                                                             
$ kubectl rollout restart deployment api --namespace=value
deployment.apps/api restarted

$ curl localhost:31000/value                              
457
```

Woo! The value persisted thru the restart. Awesome.

For extra fun, let's run some commands inside the redis cluster and see if we have the data there. First, let's find the pod running redis.

```shell
$ kubectl get pods --namespace=value
NAME                     READY   STATUS    RESTARTS   AGE
api-6bc69489d-fjzbz      1/1     Running   0          3m28s
redis-84d6945f5c-qwgj5   1/1     Running   0          14m
```

Now I'll use `exec` to drop into a shell in the redis pod.

```shell
$ kubectl exec -it redis-84d6945f5c-qwgj5 --namespace=value -- sh
# redis-cli
127.0.0.1:6379> get my-value
"457"
```
