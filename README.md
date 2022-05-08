# Serverless

A serverless platform built on Kubernetes

## Installation and Usage

### To Run Serverless as Standalone

- Clone the repository
- Install skaffold if you haven't already
- Run `skaffold dev` to run serverless in dev mode or use `skaffold run` to run without auto-reload

### To run Cloudbase fully 

Checkout the Cloudbase-main [repo](https://github.com/Cloudbase-Project/cloudbase-main)

## Implementation

The system consists of 3 parts:
1) Build the image for the function
2) Deploy the image into the serverless environment
3) Monitor the functions for the autoscaler to make decisions on

Currently it supports only Nodejs runtime.

### Build Process

It uses Kaniko as its automated image builder in Kubernetes. Kaniko requires the build context with all the files required to build the image, to be present in its volumes. In order to add the function code and other config files into the kaniko volume, we first run an “Init Container” before running the kaniko image itself.

Init Containers are specialized containers that run before the app containers in a Pod. The init container must exit successfully for the next app container to start. These are generally used to run setup scripts which is exactly our requirement.

Our “Kaniko-Worker” pod that builds the image and pushes to the registry runs a total of 2 containers, one init container that contains a busybox image, and the other the kaniko image itself. BusyBox is a software suite that provides several Unix utilities in a single executable file. The init container runs a shell script that loads the code and config files given by the user onto the common volume that the two containers share.

Once the required registry credentials and other configurations are loaded onto kaniko, the build process starts.
And when the image is built and is pushed to the remote registry, the function is marked as read-to-deploy by the serverless service in the database. The required clean up work of deleting the dangling kaniko-worker pod after it is done is performed by the serverless service.


### Deployment Process

The next step is to deploy the function image onto the serverless environment. It makes use of native kubernetes resources to achieve this. It create a Kubernetes Deployment and ClusterIP service that put together the provisioning and scaling of the image and the networking for the container respectively.
Deployments are a higher level abstraction over pods and Replica Sets.

ClusterIP service is a type of Service resource in kubernetes that enables networking for the deployment within the cluster. Other types of services also exist for different use cases.

Once the function has been deployed and is available for use, it registers itself with the router service which takes care of routing external traffic to the corresponding function deployment.


## Future Scope

A number of improvements can be made to this existing Serverless implementation.

- Instead of using Deployments and Services offered as native kubernetes resource, moving to Custom Resource Definitions (CRDs) which provide greater control over the resource lifecycle.
- Using something like the Operator Framework offers finegrain control over everything that is happening to the resource. This prevents accidental deletes, node failure, provisioning control etc.
- Incremental changes can be made by using Kubernetes Admission Hooks which helps us keep in sync with the cluster.
- Another bottleneck in the system is the Router service which becomes a single point of failure as all
the function requests flow through it. Instead, something like Traeffik which is a Kubernetes-aware reverse proxy could be used. Traefik integrates with Kubernetes and configures itself automatically and dynamically. It listens to Kubernetes and instantly generates the routes for the function. This architecture is better because Traefik can be run in High availability mode which offers resiliency and stability.

**Event Processing system:** A lot of use cases for serverless functions involve event processing.
Cloudbase could use an internal message queue like RabbitMQ or NATS which works as our event bus. Using a specification for tagging messages/events, we could map these messages/events to one or more functions. Cloudbase should also divide the functions into two types - “HTTP” and “EVENT” based functions.

The above provided factors clearly present that there is a lot of scope for improvement in Cloudbase which is required to be used anywhere near a production environment. At the very least, it was a fun project which helped me learn a lot about developing applications in/for Kubernetes and the inner workings of a Serverless system. 
