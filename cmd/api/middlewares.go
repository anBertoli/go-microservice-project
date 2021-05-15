package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/pkg/tracing"
)

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Add the "Vary: Authorization" header to the response. This indicates to any
		// caches that the response may vary based on the value of the Authorization
		// header in the request.
		w.Header().Add("Vary", "Authorization")

		// Retrieve the value of the Authorization header from the request. This will
		// return the empty string "" if there is no such header found. If there is no
		// Authorization header found, call the next handler in the chain and return
		// without executing any of the code below.
		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Otherwise, we expect the value of the Authorization header to be in the format
		// "Bearer <key>". We try to split this into its constituent parts, and if the
		// header isn't in the expected format we return a 401 Unauthorized response
		// using the invalidAuthenticationTokenResponse() helper.
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Extract the actual authentication (plain) key from the header parts.
		plainKey := headerParts[1]

		// Retrieve the details of the user associated with the authentication key,
		// along with the key being used and all related permissions. We will call
		// calling the invalidAuthenticationTokenResponse() helper if no matching
		// record was found.
		user, err := app.store.Users.GetForKey(plainKey)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		keys, err := app.store.Keys.GetForPlainKey(plainKey)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		permissions, err := app.store.Permissions.GetAllForKey(plainKey, false)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		// Call the RequestSetAuth() helper to add the user information to the request
		// context. This information will flow into the real HTTP handler and in each
		// internal service that will receive the context.
		r = r.WithContext(store.ContextSetAuth(r.Context(), &store.Auth{
			User:        user,
			Keys:        keys,
			Permissions: permissions,
		}))

		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}

// The logging middleware is used to create a request trace and to log incoming and ending requests.
// A request trace is created and put into the request context. Before passing the control to the next
// http handler the incoming request is logged. Another log is made after the requests handling, which
// uses request trace data that could be enriched by other context users.
func (app *application) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Create a request trace, and put it into the request context. Note that a
		// pointer is used, so functions that retrieve the trace could simply modify
		// in place the value pointed to.
		requestTrace := &tracing.RequestTrace{
			ID:    tracing.GenRequestID(20),
			Start: time.Now().UTC(),
		}
		r = tracing.TraceToRequestCtx(r, requestTrace)

		// Perform the first log about the incoming request.
		ip, _ := realIP(r)
		app.logger.Infow("incoming request",
			"id", requestTrace.ID,
			"start_time", requestTrace.Start,
			"remote_addr", r.RemoteAddr,
			"real_ip", ip,
			"URL", r.URL,
			"method", r.Method,
		)

		next.ServeHTTP(w, r)

		// After the request handling produce another log. Note that some values could not
		// be present since is responsibility of other http handlers to enrich the trace,
		// but it is not mandatory. Logs are produced with different severity based on the
		// HTTP code.
		end := time.Now().UTC()
		fields := []interface{}{
			"id", requestTrace.ID,
			"http_status", requestTrace.HttpStatus,
			"end_time", end,
			"duration", end.Sub(requestTrace.Start),
		}
		if requestTrace.Err != nil {
			fields = append(fields, "err", requestTrace.Err)
		}

		switch requestTrace.HttpStatus / 100 {
		case 0, 1, 2, 3:
			app.logger.Infow("request completed", fields...)
		case 4:
			app.logger.Warnw("request completed", fields...)
		case 5:
			app.logger.Errorw("request error", fields...)
		}

	})
}

func (app *application) globalRateLimit(next http.Handler) http.Handler {

	// Initialize a new rate limiter which allows an average of 2 requests
	// per second, with a maximum of 4 requests in a single ‘burst’
	limiter := rate.NewLimiter(
		rate.Limit(app.config.RateLimit.Rps),
		app.config.RateLimit.Burst,
	)

	// The function we are returning is a closure, which can access the limiter variable.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Call limiter.Allow() to see if the request is permitted, and if it's not,
		// then we call the rateLimitExceededResponse() helper to return a 429 Too Many
		// Requests response.
		if !limiter.Allow() {
			app.rateLimitExceededResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

/*
 * Using the per-IP rate-limiting pattern will only makes sens if your API application is
 * running on a single-machine. If your infrastructure is distributed, with your
 * application running on multiple servers behind a load balancer, then you’ll need
 * to use an alternative approach. If you’re using HAProxy or Nginx as a load balancer
 * or reverse proxy, both of these have built-in functionality for rate limiting that
 * it would probably be sensible to use. Alternatively, you could use a fast database
 * like Redis to maintain a request count for clients, running on a server which all
 * your application servers can communicate with.
 */

func (app *application) ipRateLimit(next http.Handler) http.Handler {

	// Define a ipLimiter struct to hold the rate limiter and last seen time for each
	// client.
	type ipLimiter struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	// We keep in memory a map of IPs -> ipLimiters, the map must be accessed with a
	// mutex to avoid concurrency issues.
	var (
		mu      sync.Mutex
		clients = make(map[string]*ipLimiter)
	)

	// Launch a background goroutine which removes old entries from the clients map once
	// every minute.
	go func() {
		for {
			time.Sleep(time.Minute)
			// Lock the mutex to prevent any rate limiter checks from happening while
			// the cleanup is taking place.
			mu.Lock()
			// Loop through all clients. If they haven't been seen within the last three
			// minutes, delete the corresponding entry from the map.
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			// Importantly, unlock the mutex when the cleanup is complete.
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, err := realIP(r)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		mu.Lock()

		// Create and add a new ipLimiter struct to the map if it doesn't already exist,
		// then set lastSeen time to now().
		_, found := clients[ip]
		if !found {
			clients[ip] = &ipLimiter{limiter: rate.NewLimiter(
				rate.Limit(app.config.RateLimit.Rps),
				app.config.RateLimit.Burst,
			)}
		}
		clients[ip].lastSeen = time.Now()

		// Call the Allow() method on the rate limiter for the current IP address. If
		// the request isn't allowed, unlock the mutex and send a 429 Too Many Requests
		// response, just like with the global rate limiting strategy.
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			app.rateLimitExceededResponse(w, r)
			return
		}

		// Very importantly, unlock the mutex before calling the next handler in the
		// chain. Notice that we DON'T use defer to unlock the mutex, as that would mean
		// that the mutex isn't unlocked until all the handlers downstream of this
		// middleware have also returned.
		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

func realIP(r *http.Request) (string, error) {
	addr := r.Header.Get("X-Real-Ip")
	if addr == "" {
		addr = r.Header.Get("X-Forwarded-For")
		if addr == "" {
			addr = r.RemoteAddr
		}
	}
	ip, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	return ip, nil
}

/*
 * If your API endpoint requires credentials (cookies or HTTP basic authentication)
 * you should also set an Access-Control-Allow-Credentials: true header in your responses.
 * If you don’t set this header, then the web browser will prevent any cross-origin responses
 * with credentials from being read by JavaScript. Importantly, you must never use the
 * wildcard Access-Control-Allow-Origin: * header in conjunction with Access-Control-Allow-Credentials: true,
 * as this would allow any website to make a credentialed cross-origin request to your API.
 */

func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")

		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		for _, trustedOrigin := range app.config.Cors.TrustedOrigins {
			if origin != trustedOrigin {
				continue
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Check if the request has the HTTP method OPTIONS and contains the
			// "Access-Control-Request-Method" header. If it does, then we treat
			// it as a preflight request.
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				// Set the necessary preflight response headers
				w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				// Write the headers along with a 200 OK
				// status and return with no further action.
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// The default behaviour, if a panics happens inside our handlers, is: unwind the stack for
// the affected goroutine (calling any deferred functions along the way), close the underlying
// HTTP connection, and log an error message and stack trace. This is ok, but it would be
// nicer to recover from the panic and send a proper HTTP 500 error.
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
