# Snap Vault

Snap Vault is a project made to share, illustrate and discuss patterns and best practices for REST APIs and 
servers written in Go.

Snap Vault is a simple REST API that basically performs CRUD operations on galleries and images, with an authentication
system built on top of authentication keys and permissions. The focus here is not on the features of the application 
but on the architecture of the project.

The project is composed of:
- the Snap Vault API binary
- the Snap Vault CLI binary
- database migrations (postgres)
- deploy scripts and systemd units
- several in-code explanations

## Project structure

The architectural pattern used in this project is influenced by the hexagonal architecture. In practice, it means, 
among several things, that the business logic should have no knowledge of transport-related concepts. Your core 
services shouldn’t know anything about HTTP headers, gRPC error codes or any other adapter used to expose them 
to the world. Applying the principle to the Go language, Go-kit was inspirational about this. I suggest taking
a look at it at https://gokit.io/. However, I decided to drastically reduce the boilerplate and the complexity of
go-kit by not following exactly the same patterns used there.

The project is laid out in two layers.

1. Transport layer. The transport layer is bound to concrete transports like HTTP or gRPC. No business logic
is implemented here, the goal of this layer is to expose your services to the world creating transport specific 
adapters, like for HTTP, RPC, CLI, events, etc.

2. Service layer. This layer is where all of the business logic is implemented. Tipically, each service method
is exposed in a single transport endpoint. Services shouldn't have any knowledge about the transport layer.  

Both of two the layers could be wrapped with middlewares to add functionality, such as logging, rate limiting, 
metrics, authentication and so on. It’s common to chain multiple middlewares around an endpoint or service. 

The two layers division and the middleware (decorator) pattern enforce a more strict separation of concerns and 
help to reuse (business-logic) code when needed. Adding a new transport for your services will be just a matter 
of writing some adapter functions. 


----------------- image here

### Services

As anticipated above, services implement all the business logic of the application. They are agnostic of the concrete
transport method used to expose them to the world. In other words, you can reuse the same service to provide similar 
functionalities to a JSON REST API server, to a CLI, to an RPC server and so on. By using an interface, you enforce the 
fact that transport adapters couldn't introspect you business logic.

In practice, services of this project are modeled as concrete implementations of an interface defined specifically for the 
domain area (users, galleries and so on). Service middlewares also satisfy the same interface, so they can be chained 
together and with the core service to provide additional functionalities.

The following code snippets puts in practice the concept. It is only a trivial example, but it could help to grasp 
the idea. 

First of all we define an abstract interface for our service.

```go
package hotel

// Define a service interface and an helper type.
type Service interface { 
    BookRoom(ctx context.Context, userID, roomID int64, people int) (Reservation, error)
    UpdateReservation(ctx context.Context, reservationID int64, people int) error
    DeleteReservation(ctx context.Context, reservationID int64, people int) error
    ConfirmAndPay(ctx context.Context, reservationID int, bankAccount string) error
}

type Reservation struct {
    ID      int64
    UserID  int64
    RoomID  int64
    Price   int
    People  int
}
```

Then we provide at least one concrete implementation of the interface, here we hypothetically save the data in a 
database, and we contact some payment service.

```go
package hotel
 
// Define a struct that holds shared dependencies...
type SimpleService struct {
    Store         store.Models
    Logger        log.Logger
    BankEndpoint  string
}

// ... and make sure it implements the Service interface.

func (ss *SimpleService) BookRoom(ctx context.Context, userID, roomID int64, people int) (Reservation, error) {
    // Reserve the room for the user and return back reservation data. In practice, insert a  
    // reservation into the database. 
    // ...
}

func (ss *SimpleService) UpdateReservation(ctx context.Context, reservationID int64, people int) error {
    // Update the number of people for the reservation, making sure the room has enough space. 
    // In practice, update the reservation in the database. 
    // ...
}

func (ss *SimpleService) DeleteReservation(ctx context.Context, reservationID int64, people int) error {
    // Delete an existing reservation identified by the provided ID, the room will be available again. 
    // In practice, delete the reservation from the database. 
    // ...
}

func (ss *SimpleService) ConfirmAndPay(ctx context.Context, reservationID int, bankAccount string) error {
    // Confirm an existing reservation identified by the provided ID and charge the bill the user 
    // bank account. In practice, update the reservation in the database and contact the payment
    // service (it could be anyhting from an internal microservice to a third-party service).  
    // ...
}
```

### Transports


