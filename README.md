# Autonats
[![Build Status](https://drone.zyra.ca/api/badges/zyra/autonats/status.svg)](https://drone.zyra.ca/zyra/autonats)
[![Go Report Card](https://goreportcard.com/badge/github.com/zyra/autonats)](https://goreportcard.com/badge/github.com/zyra/autonats)

Generates a simple service mesh that runs over [NATS](https://nats.io) by parsing Go interface declarations. The genearted code simplifies the process of creating publishers and subscribers and make it easy for services to communicate.

The project is still in early stages and supports basic use cases, see [Project info](#project-info) and [Ideas](#ideas) below to learn more about the project goals and vision.

<br><br>

## Usage

#### Enable code generation for an interface
```go
// Add the following comment to instruct Autonats
// to generate files for this interface:
//
// @nats:server User
type UserService interface {
  // Add as many methods in this interface
  // 
  // Methods that do not return any values will be treated as 
  // fire and forget calls. Errors during transport will be logged
  // and the message would be considered as delivered as soon
  // as a server receives it regardless of the actual processing status.
  
  // takes one param and returns one param + error
  GetById(ctx context.Context, id string) (*User, error)
 
  // takes no params
  GetAll(ctx.Context) ([]*User, error)
  
  // takes no params, returns error only
  DeleteAll(ctx.Context) error
}
```

#### Run CLI tool
You can run the tool by downloading it from the releases page, or by using the docker image.

```shell script
# CLI tool
$ autonats g 

# Docker
$ docker run -it --rm -v $(pwd):/root/ docker.pkg.github.com/zyra/autonats/autonats:v1.0.1 g

# TODO: upload to dockerhub
```

#### Use the generated code

##### Server handler
```go
import (
  "github.com/nats-io/nats.go"
  "context"
  "os"
)

// dummy struct
type User struct {}

// dummy service that implements UserService interface
type UserService struct {}
func (s *UserService) GetAll(ctx context.Context) ([]*User, error) {
  // logic here
  return []*User{}, nil
}

func (s *UserService) GetById(ctx context.Context, id string) (*User, error) {
  // logic here
  return &Use{}, nil
}

func (s *UserService) DeleteAll(ctx context.Context) error {
  // logic here
  return nil
}

func main() {
  var nc *natsgo.Connection // replace with a real connection
  
  svc := &UserService{}
  h := NewUserHandler(nc)
  
  if err := h.Run(ctx); err != nil {
    panic(err)
  }
  // server is up and running and is ready to process requests
  // it will shut down automatically when context is done
   

  // add a blocking code (chan os.Signal or anything else)
  sCh := make(chan os.Signal)
	signal.Notify(sCh, syscall.SIGINT, syscall.SIGTERM)
  <-sCh
  
  h.Shutdown() // Shutdown will unsubscribe from the related NATS topics. It's not required to call it before exiting if you are using context.Context
}
```

##### Client code
```go
import (
  "github.com/nats-io/nats.go"
  "log"
)

var nc *nats.Conn // replace with a real NATS connection

// create client
client := NewUserClient(nc)

// use it as defined in the interface
user, err := GetById(ctx, "someId")
```

#### Tracing
To enable tracing, add the `--tracing` flag when generating your code. This will generate code to create spans when sending and handling service calls. 

Tracing is currently handled using the OpenTracing SDK and [not.go](https://github.com/nats-io/not.go). Spans are created on servers (handlers) and clients. Operation names use the following format: `autonats:<ServiceName><Server|Client>:<MethodName>`. For example, the `User` service in the usage docs above would create a span on the client side with the name `autonats:UserClient:GetById` and `autonats:UserServer:GetById` on the handler side.


#### Timeouts
Default timeout for each method is 5 seconds. You can override this value when using the `--timeout` CLI flag.

Timeout value is used to create a context with a timeout when sending/receiving requests over NATS.

#### Concurrency
Default concurrency for each method is 5. You can override this value using the `--concurrency` CLI flag.

The concurrency option allows limiting the number of concurrent requests that a process can handle at the same time. This is useful to avoid a crash that disrupts multiple requests due to a panic/memory leak...etc. There is no recommended value to use, it depends on how confident you are with the handler code, if you have panic recovery logic in place, and if you have retry logic for critical requests. 



<br><br>

## Project info
This project aims to provide a simple service mesh implementation to allow various backend services to communicate. [NATS](https://nats.io) was picked as the transport layer since it provides a reliable messaging system with various features that can help this project grow without adding much complexity. 

**Service discovery**: Implementing a service mesh over NATS doesn't require having a service discovery logic, configuration, or external service. Topics represent a service and a method and the client doesn't need to find or even know which handler is responding to a request. 

**Load balancing**: NATS architecture provides load balancing out of the box and allows you to run your handlers anywhere as long as they are able to connect to NATS and subscribe to the relevant topics.



<br><br>

## Ideas
The concepts below are just rough ideas and aren't planned for development yet. Most ideas are aimed to provide similar funcionality to alternative methods of creating service meshes, while keeping all components as modular as possible, and without adding much complexity.

<details>
	<summary><b>Versioning</b></summary>
	versioning services is useful specially for larger projects that can't always be updated at the same time. This is currently possible by simply adding new methods (e.g `GetByIdV2(...)`) but this might get messy. Ideally services and clients would be configured with a specific version, and NATS topics can be used to specify what version to connect to. Example: currently an Autonats generated service would use a topic similar to `autonats.user.GetById`, with versioning the topics would be prefixed with the service version: `autonats.user.v1.GetByID`
</details>
<details>
	<summary><b>Metrics</b></summary>
	 when deploying an Autonats service handler on *Kuberenetes*, it would be useful to have metrics that can trigger a *HorizontalPodAutoscaler* to scale up or down the Deployment. This can be done by exporting Kuberenetes Metrics API compatible metrics that indicate the current or average capacity. For example, with this metric value we can create an HPA that automatically scales a service when its average capacity is `2` or less since that indicates that the service is starting to become very busy.
</details>
<details>
	<summary><b>Multi language support</b></summary>
	currently Autonats is designed to create service meshes that connect Go services together. However, it can use the same parsed interfaces to generate TypeScript code, protobuf spec... etc. Alternatively it can support various inputs/outputs to allow defining servies in various ways and generating code for multiple languages.
</details>
<details>
	<summary><b>Circuit breaking</b></summary>
	This feature can be implemented in a distributed way *(i.e service handlers will automatically shutdown when error rate is above accepted threshold)* or it can be implemented with an external service *(e.g Kubernetes Operator)*. An external service would require that each service handler exports relevant metrics *(e.g error rate, avg req time)* to make decisions and then kill/restart the service based on the environment *(e.g restart docker container, delete k8s pod)*.
</details>

<details>
	<summary><b>Kubernetes Operator</b></summary>
	Build an operator that works alongside [NATS Operator](https://github.com/nats-io/nats-operator) to automatically deploy and configure services. The operator can manage NATS resource definitions to configure access for each service.
	
	Examples:
```yaml
---
# define a service with it's methods
# can be generated from the same source code
apiVersion: autonats.zyra.ca/v1
kind: Service
metadata:
  name: user
spec:
  # methods allow the operator to know what methods does each service expose
  # which can allow for fine grained access control 
  methods:
  - "GetById"
  - "GetAll"
  
---
# operator would create credentials/tls certs for this service and inject them automatically
# then it will configure the NATS cluster to allow these credentials to only publish / subscribe to the relevant channels based on the service name + version
apiVersion: autonats.zyra.ca/v1
kind: Handler
metadata:
  name: user
spec:
  service: user # refers to the service defined above
  version: v1
  natsCluster: my-nats # refers to a NATS cluster resource created by NATS operator
  capacity: 10 # configure capacity/concurrency from here
  autoscaling:
    enabled: true
    minCapacity: 3
  tracing: true
  template: # pod template
    spec:
      image: my-handler-image:latest
      resources:
        requests:
          cpu: 100m
          memory: 128Mi
        limits:
          cpu: "1"
          memory: 1Gi
  
---
# Deployments / Pods / Daemonsets.. etc can use annotations to configure access
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user
  annotations:
    # comma separated list of services used by this deployment, optionally specifying version
    autonats.zyra.ca/uses-services: "user.v1,account,image"
    # specify where the NATS TLS certificates should be mounted (where your app expects it to be at)
    autonats.zyra.ca/tls-path: "/path/to/tls/certs" 
spec: {} # your spec goes here
```
</details>
