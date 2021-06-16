# Snap Vault

Snap Vault is a project made to share, illustrate and discuss patterns and best practices for REST APIs and 
servers written in Go.

Snap Vault is a simple REST API that basically performs CRUD operations on galleries and images. The application incorporates
an authentication system built on top of the concept of users, keys and permissions. The focus here is not on the features
of the application but on the architecture of the project.

The repository contains also basic scripts and configuration files useful to deploy the REST API on a remote machine and to 
monitor the runtime behaviour of the application (with Prometheus + Grafana). 

The project is composed of:
- the Snap Vault API binary
- the Snap Vault CLI binary
- database migrations (postgres)
- deploy scripts and systemd units
- several in-code explanations

## Architecture 

![architecture of the application](./assets/architecture.svg "architecture")

The public interface of the architecture is Nginx which act as a reverse proxy. Nginx redirect HTTP requests to the
REST API, which is our Go application. The API have exclusive access to the Postgres database and writes data to the 
file system (in a specific _storage_ directory). The API binds to the loopback interface, so it is not publicly 
accessible. API responses will flow through Nginx to reach the clients.

Let's talk about monitoring. The private Prometheus instance scrapes metrics to the instrumented Go application and 
exposes fetched metrics to the Grafana server, which will periodically poll Prometheus. Nginx will redirect requests 
starting with _/grafana_ to the grafana dashboard (protected with its own auth system). Additionally, the _/metrics_ 
endpoint of the REST API is blocked by Nginx (it is used from Prometheus to poll the application).



## Project structure

The architectural pattern used in this project is influenced by the hexagonal architecture. In practice, it means, 
among several things, that the business logic should have no knowledge of transport-related concepts. Your core 
services shouldn’t know anything about HTTP headers, gRPC error codes or any other adapter used to expose them 
to the world. Applying the principle to the Go language, Go-kit was inspirational about this. I suggest taking
a look at it at https://gokit.io/. However, I decided to drastically reduce the complexity of go-kit by not following 
exactly the same patterns used there.

The project is laid out in two layers.

1. Transport layer. The transport layer is bound to concrete transports like HTTP or gRPC. No business logic
is implemented here, the goal of this layer is to expose your services to the world by creating transport specific 
adapters, like for HTTP, RPC, CLI, events, etc.

2. Service layer. This layer is where all of the business logic is implemented. Tipically, each service method
is exposed in a single transport endpoint. Services shouldn't have any knowledge about the transport layer.  

Both the layers could be wrapped with middlewares to add functionality, such as logging, rate limiting, 
metrics, authentication and so on. It’s common to chain multiple middlewares around an endpoint or service. 

The division in two layers and the middleware (decorator) pattern enforce a more strict separation of concerns and 
allows us to reuse code when needed. Adding a new transport for your services will be just a matter of writing some 
adapter functions. 

----------------- image here

## Services

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
package booking

// Define a service interface and some helper types.
type Service interface {
    ListRooms(ctx context.Context, page int) ([]Room, error)
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

type Room struct {
    ID      int64
    Name    string
    Prince  int
}
```

Then we provide at least one concrete implementation of the interface, here we hypothetically save the data in a 
database, and we contact some payment service.

```go
package booking
 
// Define a struct that holds shared dependencies...
type SimpleService struct {
    Store         store.Models
    Logger        log.Logger
    BankEndpoint  string
}

// ... and make sure it implements the Service interface.

func (ss *SimpleService) ListRooms(ctx context.Context, page int) ([]Room, error) {
    // List available rooms, retrieving data from the database.
    // ...
}

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
    // service (it could be anything from an internal microservice to a third-party service).  
    // ...
}
```

Finally, our service can be used by other parts of the application, mainly by the transport layer.

```go
// Define an interface variable...
var bookingService booking.Service

// ... and assign the concrete implementation to it.
bookingService = booking.SimpleService{store, logger, "https://bank-endpoint"}

// Later on...
res, err := bookingService.BookRoom(ctx, userID, roomID, people)
if err != nil {
    // Handle the error.
}
```

### Service middlewares
Above, we defined an interface to our booking service. We can create some middlewares to provide additional
functionalities to our service. Service middlewares will satisfy the same interface, so they can be chained 
together and wrap the core service.

Service middlewares should provide business-logic related features, while transport ones will be provided by
transport middlewares. 

```go
package booking 

// Note that we embed a Service interface in our middleware. This will provide interface the 
// interface implementation for free, but we can still override whatever method we want. 
type AuthMiddleware struct {
    Authenticator auth.Auther
    Service 
}
var (
    ErrUnauthenticated = errors.New("unauthenticated")
    ErrForbidden = errors.New("forbidden")
)


// The pattern is the same for each method: extract the auth token from the context
// authenticate the user and make sure it has the necessary permissions. Return
// early if something goes wrong.

func (am *AuthMiddleware) BookRoom(ctx context.Context, userID, roomID int64, people int) (Reservation, error) {
    authData, err := am.Authenticator.Auth(ctx)
    if err != nil {
        return Reservation{}, ErrUnauthenticated
    }

    isAllowed := checkPermissions(authData.Permissions, "book-room-perm")
    if !isAllowed {
         return Reservation{}, ErrForbidden
    }

    return am.Service.BookRoom(ctx, userID, roomID, people)
}

// Implement other methods. We don't need to implement the ListRooms API.
// ...
```

Note the following two things.
- The 'core' service could be entirely skipped on behalf of the middleware. In our example if the user doesn't
have the necessary permissions the middleware returns early.

- The `ListRooms` method is not present on the middleware since it is a public API. The interface is still satisfied
since calls to this method will be directly delegathed to the embedded Service. 

Similarly, we can define a metrics middleware to record statistics about our service utilization.

```go
package booking 

type MetricsMiddleware struct {
    Metrics metrics.Recorder
    Service
}

func NewMetricsMiddleware(m metrics.Recorder, next Service) *MetricsMiddleware {
    // Create and register needed metrics, e.g. instrument the code with Prometheus
    // counters, gauges and histograms.
    // ...

    return &MetricsMiddleware {
        Metrics: metrics.Recorder,
        Service: next,
    }
}

func (mm *MetricsMiddleware) ListRooms(ctx context.Context, page int) ([]Room, error) {
    // Defer a function which records the time took by the API
    // to process the request.
    defer func(start time.Time) {
        mm.Metrics.Record("list-rooms-time", time.Now().Since(start))
    }(time.Now())
    
    // Increment the API counter.
    mm.Metrics.Add("list-rooms", 1)
    
    // Call the core service.
    return mm.Service.ListRooms(ctx, page)
}

// Implement all other methods.
// ...
``` 

Finally, we can wire everything together, typically in our main function. The order of the middlewares
could be changed based on your specific needs.

```go
var bookingService booking.Service

bookingService = booking.SimpleService{store, logger, "https://bank-endpoint"}
bookingService = booking.AuthMiddleware{authenticator, bookinService}
bookingService = booking.NewMetricsMiddleware(metrics, bookingService)

// The method call will pass through (in order): the metrics middleware, the
// authentication middleware and eventually to the core booking service. 
res, err := bookingService.BookRoom(ctx, userID, roomID, people)
if err != nil {
    // Handle the error.
}
```

## Transports

We defined our services and all related middlewares, now we have to expose th service to the outside.
The transport layer is related to concrete transports like JSON over HTTP or gRPC. No business logic
should be implemented here.

Each type of transport has its own peculiarities and nuances, but all implementation follow this pattern:
- a handler is defined for each service API (not strictly rule)
- the handler collects/extracts relevant data from the request
- the handler pass the collected data to the relevant service
- the returned output is sent to the client  

We will implement JSON over HTTP adapters to the service defined previously. This layer can be modelled 
using closures (which return HTTP handlers) or using a struct whose methods are HTTP handlers. The choice
is not so important. In the Snap Vault project the second approach was followed.

```go
package jsonapi 

// Define a jsonapi struct that holds the core services and some additional deps...
type jsonapi struct {
    booking booking.Service     // defined above
    users   users.Service       
    mailer  mailer.Mailer
    logger  log.Logger
}

// ... then in the same package define the HTTP handlers.

func (j *jsonapi) listRoomsHandler(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()
    page, err := strconv.Atoi(query.Get("page"))
    if err != nil {
        page = 0
    }

    rooms, err := j.booking.ListRooms(r.Context(), page)
    if err != nil {
        // Handle the error appropriately.
        return
    }

    roomsBytes, err := json.Marshal(rooms)
    if err != nil {
        // Handle the error appropriately.
        return
    }
    w.Write(roomsBytes)
}

// Other HTTP handlers implementations.
// ...
```

The last step is to register our routes and start the server. You can use the routing strategy 
you want, here I used the mux router.

```go
router := mux.NewRouter()

router.Methods(http.MethodGet).Path("/booking/rooms").HandlerFunc(api.listRoomsHandler)
router.Methods(http.MethodPost).Path("/booking/rooms/{id}").HandlerFunc(api.bookRoomHandler)
router.Methods(http.MethodPut).Path("/booking/rooms/{id}").HandlerFunc(api.updateReservationHandler)
router.Methods(http.MethodDelete).Path("/booking/rooms/{id}").HandlerFunc(api.deleteReservationHandler)
router.Methods(http.MethodPost).Path("/booking/rooms/confirm/{id}").HandlerFunc(api.confirmationHandler)

err := http.ListenAndServe("127.0.0.1:4000", router)
if err != nil {
    log.fatal(err)
}
```

In the same way we can define transport-specific adapters for a gRPC server, for a CLI, for a SQS polling
system and so on. 


### Transport middlewares

Transport middlewares are not modelled following an interface since they are tightly coupled with the 
concrete transport type. For HTTP middlewares there is a well-known pattern to create middlewares.

```go
func httpMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Do something before serving the request (or the next middleware).

        // Proceed with next handler in the chain.
        next.ServeHTTP(w, r)
    
        // Do something after the request was served.
	})
}
```

The pattern takes advantage of closures to return a wrapped version of the original HTTP handler. It is possible
to wrap a single endpoint or to the entire HTTP handler passed to the ListenAndServe function. 

```go
router := mux.NewRouter()

// Apply some middleware to a single endpoint...
listRoomsHandler := middleware1(api.listRoomsHandler)
listRoomsHandler = middleware2(listRoomsHandler)

router.Methods(http.MethodGet).Path("/booking/rooms").HandlerFunc(listRoomsHandler)


// ... or globally to the resulting HTTP handler.
handler := middleware3(router)
handler = middleware4(handler)

err := http.ListenAndServe("127.0.0.1:4000", handler)
if err != nil {
    log.fatal(err)
}
```

In the Snap Vault codebase several HTTP middlewares were used, all related to HTTP related issues:
- logging & tracing 
- rate-limiting
- CORS authorization
- auth key extraction

#### A note on authentication 

It is usual to perform authentication in an HTTP middleware, especially if the project doesn't enforce
the separation between transport and business logic (services). This pattern is not followed for Snap Vault.

The reason is that authentication & authorization are part of the business logic and should be done inside the
service layer or in a dedicated service middleware. This results in more code (not so much to be honest) but will 
result in a cleaner code, and a better separation of concerns. The transport middleware is still responsible to 
extract the auth key from a transport-specific location, i.e. the Authorization header for HTTP requests.

This point is debatable, and it is perfectly acceptable to perform authentication in a transport middleware.
Software engineering involves trade-offs, and valuable exceptions could be made. Note however that DRY code
is not always a cleaner code.    


## Data persistence

Storing and retrieving data is typically part of the business logic. For simple data manipulation
it is sufficient to pass a sql.DB pointer to the services concrete implementations. For more than 
trivial operations you usually want to create a storage package with a concrete Store type. This type
will hold the concrete db connection pool and provides operations on data implemented as methods. The 
defined storage type is provided to the services concrete implementations.

```go
package storage

type Store struct {
    logger log.Logger
    db     *sql.DB
}

func (s *Store) InsertReservation(ctx context.Context, res Reservation) error {
    row := s.db.QueryRow(`
        INSERT INTO reservation (userID, roomID, people) 
        VALUES ($1, $2, $3)
        RETURNING id, created_at
    `, res.UserID, res.RoomID, res.People)

    return row.Scan(
        &res.ID, 
        &res.CreatedAt,
    )
}
```

Even better, it is possible to define an interface also for persistence operations. Then a concrete type
can implement the interface. Each concrete implementation could support a different type of storage, e.g.
file system, S3, another storage microservice etc. In this case you provide an interface rather than a concrete
type to your services.

```go
package storage 

type Store interface {
	InsertReservation(ctx context.Context, reservation Reservation) error
	UpdateReservation(id reservation Reservation) error
	// ... other methods
}
```

Anyway, this type of abstraction could be an overkill especially if each concrete service use only one
concrete type of storage. In this case it is ok to use only concrete store types. 

## Running the binaries

Binaries are scoped under the cmd directory. The API reads the JSON config file location from the `config` flag 
(defaults to `./conf/api.dev.json`). You can find an example configuration file at `./conf/api.example.json`. This
file must be edited with valid values before starting the application.

The API could be directly started: 

```shell script
go run ./cmd/api -config <path/to/config/file>
```

or it can be compiled:

```shell script
make build
./bin/linux/snapvault-cli_<git_desc> -config <path/to/config/file>
```

Under the cmd directory there is a simple CLI. Currently, it supports only the `migrate` command, but in the future
it could be extended to support additional features. The migrate command uses the https://github.com/golang-migrate/migrate
embedded as a library.

```shell script
go run ./cmd/cli --help # obtain help for the CLI

go run ./cmd/cli migrate --help # obtain help for the migrate command

go run ./cmd/cli migrate \
  --action up  \
  --migrations-folder file://<path/to/migrations/folder>  \
  --database-url  postgres://localhost:5432/database?sslmode=disable
```


## Deploy