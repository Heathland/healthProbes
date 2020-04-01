# Let's talk Probes

So I was wondering why I see so many cases where people create Deployments for Kubernetes and took the effort of 
defining the readinessProbe and livenessProbe. This is a good thing, since Kubernetes should be able to heal application 
once pods don't respond the way they should. 

However, what I do see, is that in most cases, the same endpoint is used for both the readiness as the liveness Probes.
And that, is a bit redundant. So after a talk on the Kubernetes Slack, I noticed that there is some confusion as to what 
the difference is between, and how they should be used.

As such, I decided to make a simple demo to show what the exact difference is between the two, and at the same time show 
a new Probe as well, named startupProbe, which was added as an Alpha feature in 1.16, and graduated to beta in 1.18.

## A simple Go web server

So for this example I created a simple web server which has 4 different endpoints.

Namely:
* /startupProbe
* /livenessProbe
* /readinessProbe
* /startJob

The functions name should be a clear enough indication as to what their functions are, apart from `/startjob` which I will 
elaborate on further up.

The endpoints `/startupProbe` and `/livenessProbe` will start with giving up 503's for a given duration. The extend of 
that duration can be determined by setting the appropriate ENV values, namely: `WAIT_STARTUP_TIME` and `WAIT_LIVENESS_TIME`.
After that duration has passed, they will give off 200's.

Then we also have the `/readinessProbe` endpoint, which will start off with giving off 200's. However, once a GET is done  
against `/startjob` the `/readinessProbe` endpoint will start to give off 503 for a given duration. The duration can be set 
by adjusting the  `JOB_DURATION_TIME` ENV variable, after the duration has passed, it will also start giving off 200's again.

The default values are:  
`ENV WAIT_STARTUP_TIME 20`  
`ENV WAIT_LIVENESS_TIME 35`  
`ENV JOB_DURATION_TIME 20`

## Testing it on Kubernetes

You can find a Deployment template in the `/Kubernetes` folder in this repo.  
With it, you will start 3 pods, each containing this container. They will use the default as given above.  
Applying it can be done using the following command: `kubectl apply -f Kubernetes/Deployment.yaml`

With the given defaults, you will see the following events, when describing a single pod:
```shell script
  Type     Reason     Age                 From               Message
  ----     ------     ----                ----               -------
  Normal   Scheduled  117s                default-scheduler  Successfully assigned default/health-probes-64c858bf4b-shfpr to minikube
  Normal   Pulling    116s                kubelet, minikube  Pulling image "heathland/health-probes:latest"
  Normal   Pulled     111s                kubelet, minikube  Successfully pulled image "heathland/health-probes:latest"
  Normal   Created    111s                kubelet, minikube  Created container health-probe
  Normal   Started    111s                kubelet, minikube  Started container health-probe
  Warning  Unhealthy  93s (x4 over 108s)  kubelet, minikube  Startup probe failed: HTTP probe failed with statuscode: 503
  Warning  Unhealthy  79s (x2 over 84s)   kubelet, minikube  Liveness probe failed: HTTP probe failed with statuscode: 503
```

As you can see, 111 seconds ago, the pod was created. After that StartupProbes succeeded eventually.
The difference between the times being 111-93 is 18 seconds. Which is quite correct with the given startup value we set (being 20 seconds).  
The next one being the LivenessProbe, which took 14 seconds. Looking at the default values, the startupTime is 20 sec and the liveness is 35. 
The difference being 15 secs. So that's correct as well.

:warning: **Do note!**: The startupProbe is something that's only come into Beta in Kubernetes version 1.18!
 
Then we can start and test the readinessProbe. To do that we do two things. First we run the following command:
`kubectl get pods -w`  
And we follow that up with a second command, namely: `kubectl port-forward deploy/health-probes 8080:8080`

Now we can access the pods trough localhost. So we go to http://localhost:8080/startJob and we see a message:
```
Pod (health-probes-64c858bf4b-ph6m2)
Starting job. Unavailable till: Wed Apr  1 21:51:15 2020
```
At the same time we see container `health-probes-64c858bf4b-ph6m2` losing it Ready state. And regaining it afterwards:
```shell script
health-probes-64c858bf4b-ld5x8   1/1     Running   0          12m
health-probes-64c858bf4b-ph6m2   1/1     Running   0          12m
health-probes-64c858bf4b-shfpr   1/1     Running   0          12m
health-probes-64c858bf4b-ph6m2   0/1     Running   0          14m
health-probes-64c858bf4b-ph6m2   1/1     Running   0          14m
``` 
This is expected behaviour. Looking at the events of this pods, we see the folling:
```shell script
  Warning  Unhealthy  2m3s (x4 over 2m18s)  kubelet, minikube  Readiness probe failed: HTTP probe failed with statuscode: 503
```
This proves the readinessProbe works!

## Ok... what are they for then ?
So let me explain what these probes are designed for.

By default Kubernetes will monitor a pod to see if it's doing what it's supposed to be doing... sorta. A docker container has an entrypoint 
and that will start an application. In this case, it will start `app`, which is the Golang webserver.
However, say the application should be able to communicate with a database as well in order to function fully. Kubernetes will not 
know by default that the container isn't doing anymore what you'd expect it to do. For that the Probes are made.

So let's start with `startupProbe`. We all know a few applications which can take quite a bit of time to start. StartupProbe is made 
specifically to know when the pod is fully started. It's handy to have this apart from, say livenessProbe, as this startupProbe is only used for the start.
Giving you the option to have the LivenessProbe with shorter times. After that, it won't do anything anymore during the container's livecycle. So this is a good one to set with somewhat larger times. In this example it 
it is set with a period of 5 sec, and a threshold of 30. Meaning it will poll for 30*5=150 seconds max. 

Then we have `livenessProbe`. This is for the example I gave above. The moment you have a container not responding as it should, you can use the livenessProbe 
to kill it, and restart it. A common endpoint name for this is `/healtz`. So make sure your application has a test endpoint which will output a 200 OK 
if the application has all resource it supposed to have. 

Lastly, we have `readinessProbe`. A common misconception is that it's function is that what startupProbe actually is, and that it only runs
 at the start of the container. This isn't so. The actual purpose of readiness is for containers who get a certain job to do, and then 
 can't handle any more extra work. They can then set the readinessProbe to, say 503, to indicate that it should be removed from the Service, so that 
 other pods will get the work, giving the container time to do the job it was given. The readinessProbe will not kill the container, as is designed, 
 otherwise you lose all the work you already done so far.
 

## Building the container

The Dockerfile is included in the repo. Building it, is simple. Simply run:  
`docker build . -t health-probes`

Or just use the container which is on my dockerhub:  
`docker run -P -p 8080:8080 heathland/health-probes`

Then you can access it using localhost:
* http://localhost:8080/readinessProbe
* http://localhost:8080/livenessProbe
* http://localhost:8080/startupProbe
* http://localhost:8080/startJob
