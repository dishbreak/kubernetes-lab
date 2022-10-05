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
$ docker build -t value-service:v1
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
kubectl get deployments --namespace=value    
NAME   READY   UP-TO-DATE   AVAILABLE   AGE
api    1/1     1            1           24m
```

Of course, your `AGE` value may differ, but there it is! One pod, ready and available for value-add goodness. But, um...how are we going to reach it?

