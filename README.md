# Autonats
[![Build Status](https://drone.zyra.ca/api/badges/zyra/autonats/status.svg)](https://drone.zyra.ca/zyra/autonats)
[![Go Report Card](https://goreportcard.com/badge/github.com/zyra/autonats)](https://goreportcard.com/badge/github.com/zyra/autonats)

Generate NATS client code and server handlers from Go interfaces. 

Useful for projects with multiple services that share some source code.


## Usage

#### Enable code generation for an interface
```go
// Add the following comment to instruct Autonats
// to generate files for this interface:
//
// @nats:server User
type UserService interface {
  // Add as many methods in the interface
  //
  // Each method can take one or no parameters,
  // it can return one or no parameters, in addition to an error
  // 
  // Methods that do not return any values will be treated as 
  // fire and forget calls. Errors during transport will be logged
  // and the message would be considered as delivered as soon
  // as a server receives it regardless of the actual processing status.
  //
  
  // takes one param and returns one param + error
  GetById(id primitive.ObjectId) (*User, error)
 
  // takes no params
  GetAll() ([]*User, error)
  
  // takes no params, returns error only
  DeleteAll() error
 
  // takes nothing, returns nothing
  SendInvoices()
}
```

#### Run CLI tool
You can run the tool by downloading it from the releases page, or by using the docker image.

```shell script
# CLI tool
$ autonats g 

# Docker
$ docker run -it --rm -v $(pwd):/root/ harbor.zyra.ca/public/autonats g

# TODO: upload to dockerhub
```

#### Use the generated code

##### Server handler
```go
import (
  "github.com/nats-io/nats.go"
  "context"
)

var nc *natsgo.Connection // replace with a real connection
var server UserService // struct that implements the previously defined interface

_, err := NewUserHandler(context.TODO(), server, nc)

if err != nil {
  // the handler failed to subscribe with the provided NATS connection
  // handle the error
}

// server is up and running and is ready to process requests
// it will shut down automatically when context is done
```

##### Client code
```go
import (
  "github.com/nats-io/nats.go"
  "log"
)

var nc *natsgo.Connection // replace with a real connection
l := log.New(...) // or any logger with Printf

// create client
client := NewUserClient(nc, l)

// use it as defined in the interface
user, err := GetById(...)
```