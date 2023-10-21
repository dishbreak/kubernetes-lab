# kubernetes-lab

A quick little proof of concept that lets you deploy a Redis-backed CRUD API on a Kubernetes cluster. Inspired by dishbreak/docker-compose-lab.

# Getting Started

## Before You Begin

Make sure you've downloaded and installed Docker Desktop and Homebrew. Having an IDE with a Kubernetes plugin like VSCode can help, but it's not required. This guide assumes you're on macOS, you may have to adapt it for a Windows or Linux installation.

First, install the `kind` package from Homebrew, then use it to create a cluster.

```shell
$ brew install kind
$ kind create cluster
```

Next, install the `kubernetes-cli` package from Homebrew. This will set up the `kubectl` command, which we'll use throughout the lab to interact with our cluster.

```shell
$ brew install kubernetes-cli
```

Once you've done this, you'll need to delete the `kubectl` installation that Docker Desktop put on your PATH. 

```shell
$ sudo rm $(which kubectl)
```

At the time of writing, `kubectl version` should respond like so.

```shell
$ kubectl version
Client Version: v1.28.3
Kustomize Version: v5.0.4-0.20230601165947-6ce0bf390ce3
Server Version: v1.27.3
```

Lastly, let's set the context. This will aim our `kubectl` command at our kind cluster.

```shell
$ kubectl config set-context kind-kind
Context "kind-kind" modified.
```

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

```shell
$ docker ps  
CONTAINER ID   IMAGE              COMMAND   CREATED         STATUS         PORTS                     NAMES
a8df1c2e7068   value-service:v1   "app"     2 minutes ago   Up 2 minutes   0.0.0.0:55001->8080/tcp   serene_lehmann
```

So, on my machine, the container bound Port 8080 in the container to to port 55001 on the host. Your port will almost certainly be different.

Let's interact with this value service. In a separate terminal, let's post a value to the endpoint.

```shell
$ curl -X POST localhost:55001/value -d 343
```

Great. Now let's check to see if we can get that value back.

```shell
$ curl localhost:55001/value       
343
```

This container's looking good. Let's load the docker image into our kind cluster -- this will let us use it on the cluster.

```shell
$ kind load docker-image value-api:v1
Image: "value-api:v1" with ID "sha256:e9b048efbd923e74c875c90a7e928d1d90b7616ee7c9226fab3153ca6915ec5f" not yet present on node "kind-control-plane", loading...
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

```shell
$ kubectl apply -f api/deployment.yml 
deployment.apps/api created
```

Now let's go find it!

```shell
$ kubectl get deployments
No resources found in default namespace.
```

Uhhh...what? We just created it, right? Where is it? Remember that we created the deployment in our `value` namespace. In order to see it via `kubectl`, we need to pass the `--namespace` flag.

```shell
$ kubectl get deployments --namespace=value
NAME   READY   UP-TO-DATE   AVAILABLE   AGE
api    1/1     1            1           24m
```

This is going to get old really fast. We can add the namespace to our context and let `kubectl` use it for subsequent commands.

```shell
$ kubectl config set-context kind-kind --namespace value
```

Now, running the command without the `--namespace` flag works!

```shell
$ kubectl get deployments
NAME   READY   UP-TO-DATE   AVAILABLE   AGE
api    1/1     1            1           24m
```

As a reminder, **deployments aren't pods -- they just specify how to create them**. We can use `kubectl` to find the running pods, too.

```shell
$ kubectl get pods
NAME                   READY   STATUS    RESTARTS   AGE
api-58776c686c-b2rr9   1/1     Running   0          73m
```

Of course, your `AGE` value may differ, but there it is! One pod, ready and available for value-add goodness. But, um...how are we going to reach it? We can make use of a debug container. Here we're using the `curlimages/curl:latest` container and the pod (`api-58776c686c-b2rr9` in this case). This will drop us into a shell in a container that's running in the pod.

```shell
$ kubectl debug --image curlimages/curl:latest -it api-58776c686c-b2rr9 -- sh
Defaulting debug container name to debugger-d2tsj.
If you don't see a command prompt, try pressing enter.
~ $  
```

You should be able to use `curl` within this debug container much like you did while playing with the container image using `docker run`. Making requests against `localhost:8080` will hit our value API.

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

Nice! We've got the same behavior that we saw locally, but inside our cluster.

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

Now I'll use `exec` to drop into an interactive `redis-cli` session in the redis pod.

```shell
$ kubectl exec -it redis-84d6945f5c-qwgj5 --namespace=value -- sh
# redis-cli
127.0.0.1:6379> get my-value
"457"
```

The API writes data to the redis cluster using the key `my-value`. We can see the data we POSTed to the API in the Redis cluster, so this shows that the two systems are working together!

# Step 6: Add a PersistentVolume

Remember how we lost data in the API when we restarted its deployment? Surprise, we still have that problem!

To demonstrate, let's use `redis-cli` to show the value again.

```shell
$ kubectl exec -it redis-84d6945f5c-qwgj5  --namespace=value -- redis-cli get my-value 
"457"
```

Right, so my-value has the value `"457"`. Now, let's restart the deployment.

```shell
$ kubectl rollout restart deployment redis --namespace=value                      
deployment.apps/redis restarted
```

Remember, the restart creates a new pod, so we'll need to find the new pod's name.

```shell
$ kubectl get pods --namespace=value                                                 
NAME                    READY   STATUS    RESTARTS   AGE
api-6bc69489d-fjzbz     1/1     Running   0          11h
redis-78bc567b9-9tvxx   1/1     Running   0          24s
```

And if we execute the same command against the new redis pod...uh, oops.

```shell
$ kubectl exec -it redis-78bc567b9-9tvxx --namespace=value -- redis-cli get my-value
(nil)
```

What happened? The new pod has none of the data that got written to the old pod! Remember, **pods are ephemeral**. Just like with the API pod, any data on the pod gets lost when the pod shuts down. 

In order to save data, we'll need a persistent volume and a persistent volume claim. Let's step forward in time again:

```shell
git checkout 3-persistent-volume-for-redis
```

First, check out the file `redis/pv.yml`. This file sets up a PersistentVolume in the cluster, in the form of a 500 MB file located in `/mnt/data` on the cluster. **Note there's no mention of namespace**. That's because the PersistentVolume belongs to the cluster.

Let's apply it.

```shell
$ kubectl apply -f redis/pv.yml                                                           
persistentvolume/redis-data created
```

Next, we'll create a PersistentVolumeClaim, which is essentially a "ticket" that a pod can use to get a PersistentVolume and mount. Check out the file `redis/pvc.yml`. Note that, unlike the PersistentVolume, the PersistentVolumeClaim **does** have a namespace attached. Additionally, there's nothing linking the PersistentVolumeClaim to the PersistentVolume--it's up to the cluster to find a PersistentVolume that's appropriate for the PersistentVolumeClaim.

Let's apply this.

```shell
$ kubectl apply -f redis/pvc.yml
persistentvolumeclaim/redis-pvc created
```

Now, let's take a look at the redis deployment. Under the `spec`, we now see a `volumes` section:

```yml
      volumes:
        - name: redis-storage
          persistentVolumeClaim:
            claimName: redis-pvc
```

This defines a volume for the pod that gets backed by the PersistentVolumeClaim. Next, in the `volumeMounts:` section of the container, we mount the volume appropriately to the `/data/` directory within the container.

```yml
        volumeMounts:
          - mountPath: "/data"
            name: redis-storage
```

Let's apply these changes.

```shell
$ kubectl apply -f redis/deployment.yml 
deployment.apps/redis configured
```

Now let's test the changes! Note that your pod names will be different from mine, use the following command to find your pod names:

```shell
$ kubectl get pods --namespace=value
```

First, let's touch a file in the pod, using the exec command:

```shell
$ kubectl exec -it redis-78bc567b9-9tvxx --namespace=value -- touch /data/my-file   
$ kubectl exec -it redis-78bc567b9-9tvxx --namespace=value -- ls /data           
my-file
```

Now, let's restart the deployment, to see if a new container has the same files.

```shell
$ kubectl rollout restart deployment redis --namespace=value
deployment.apps/redis restarted
```

If we did things right, the new pod will still have the file we created, and it does!

```shell
$ kubectl exec -it redis-566c8cd794-j268v --namespace=value -- ls /data
dump.rdb  my-file
```

Hold on though. There's still a problem here. To demonstrate, let's post to the API and restart the redis deployment.

```shell
$ curl -X POST localhost:31000/value -d 457
$ kubectl rollout restart deployment redis --namespace=value
```

If we check the newly created redis pod, there's a problem: there's no data there!

```shell
$  kubectl exec redis-f86675ffd-2m7n5 --namespace=value -- redis-cli get my-value

$
```

So, we've got a volume where the redis pod can persis data, but the redis software isn't making use of it yet. Hmm. What to do?

# Step 7: Configure Redis with a ConfigMap

By default, redis isn't much different from our API--it stores values in memory. It _does_ have support for something called [Append-only File (AOF) mode](https://redis.io/docs/manual/persistence/#append-only-file). In short, this mode journals writes to the filesystem, which lets redis rebuild the database after a halt and restart.

In order to turn on AOF mode, we need the following line in a redis.cfg file:

```
appendonly yes
```

So...how are we going to get this file into the filesystem of the container? We _could_ build a Docker container image that adds the config file to a `redis:7` container image, but that's got a number of drawbacks. We'd need to keep our homegrown container up to date and it's going to be hard to keep the redis system maintained. Additionaly, config changes will require us to publish and deploy new containers.

Enter [ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/)! From the Kubernetes docs (emphasis added):

> A ConfigMap is an API object used to store non-confidential data in key-value pairs. Pods can consume ConfigMaps as environment variables, command-line arguments, or as **configuration files in a volume**.

Managing configuration files in the container thru the kubernetes API sounds like _exactly_ what we need. Here's what our ConfigMap in `redis/configmap.yml` looks like:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis
  namespace: value
data:
  redisConfig: |
    appendonly yes
```

The pipe character (`|`) after the `redisConfig` key lets us create a multiline value. Let's apply it.

```shell
$ kubectl apply -f redis/configmap.yml 
configmap/redis created
```

Now, we need to modify our deployment. `redis/deployment.yml` is up-to-date, so let's take a quick tour of all our changes.

First, we've added the ConfigMap as a volume for the pod. This snippet creates a `config` volume, and writes the value attached to `redisConfig` into a file `redis.cfg` within the volume.

```yaml
      volumes:
        ...
        - name: config
          configMap:
            name: redis
            items:
              - key: redisConfig
                path: redis.conf
```

Next, we mount the `config` volume in the `volumeMounts` section. This will make the config file available at `/redis-master/redis.conf`. 

```yaml
        volumeMounts:
          ...
          - mountPath: "/redis-master"
            name: config
```

Finally we need to update the container's command to pass the config file path.

```yaml
      containers:
      - name: redis
        image: redis:7
        command:
          - "redis-server"
          - "/redis-master/redis.conf"
```

We'll update the deployment.

```shell
$  kubectl apply -f redis/deployment.yml
deployment.apps/redis configured
```

Now, let's try the same test again: post a value to the API.

```shell
$ curl -X POST localhost:31000/value -d 457
```

Next, look up the redis pod, and check the redis backend. Restart the deployment.

```shell
$ kubectl exec redis-566c8cd794-j268v --namespace=value -- redis-cli get my-value
457
$ kubectl rollout restart deployment redis --namespace=value
deployment.apps/redis restarted
```

Look up the redis pod name post deployment and check the backend.
```shell
$ kubectl exec redis-78fbfdfc85-fgk4t --namespace=value -- redis-cli get my-value
457
```

Excellent. Now the API _and_ the redis backend are able to tolerate pod restarts. This lets our system be more fault tolerant and enables us to deploy software without taking downtime. Booyah!


