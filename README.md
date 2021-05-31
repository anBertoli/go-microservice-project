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

The following code puts in practice the concept. First of all we define an abstract interface for our service.
```go
package hotel

// Define a service interface
type Service interface { 
    BookRoom(ctx context.Context, userID, roomID int) (int, error)
    UpdateBooking(ctx context.Context, reservationID int) error
    DeleteBooking(ctx context.Context, reservationID int) error
    ConfirmAndPay(ctx context.Context, reservationID int, bankAccount string) error
}

```
 

