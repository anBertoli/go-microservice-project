package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"

	"github.com/anBertoli/snap-vault/pkg/auth"
	"github.com/anBertoli/snap-vault/pkg/tracing"
)

// The extractAuthKey middleware extracts the authentication key from the request 'Authorization"
// header and put it into the request context. The logic here is not meant to authenticate the user,
// but to provide transport-specific data extraction. The authentication is business logic
// and this will be handled by the service layer.
func (app *application) extractAuthKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Add the "Vary: Authorization" header to the response. This indicates to any
		// caches that the response may vary based on the value of the Authorization
		// header in the request.
		w.Header().Add("Vary", "Authorization")

		// Retrieve the value of the Authorization header from the request. If there is
		// no Authorization header found, call the next handler in the chain and return
		// without executing any of the code below.
		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Otherwise, we expect the value of the Authorization header to be in the format
		// "Bearer <key>". If the header isn't in the expected format we return a 401
		// response (unauthorized).
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		plainKey := headerParts[1]

		// Add the auth key information to the request context. This information will flow into
		// the next HTTP handlers and in each internal service that will receive the context.
		r = r.WithContext(auth.ContextSetKey(r.Context(), plainKey))

		// Proceed with next handler in the chain.
		next.ServeHTTP(w, r)
	})
}

// The tracing middleware puts a request trace into the request context. If a trace is
// already present the middleware acts as a no-op.
func (app *application) tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqTrace := tracing.TraceFromRequestCtx(r)
		if reqTrace.ID == tracing.AnonymousID {
			r = tracing.NewRequestWithTrace(r)
		}
		next.ServeHTTP(w, r)
	})
}

// The logging middleware is used to log incoming requests and related outgoing responses.
// Before passing the control to the next http handler the incoming request is logged.
// Another log is emitted for outgoing responses, using the (possibly) enriched
// request trace.
func (app *application) logging(next http.Handler) http.Handler {

	// Wrap the returned middleware in the tracing middleware, that is, before invoking
	// the function call the tracing function logic.
	return app.tracing(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTrace := tracing.TraceFromRequestCtx(r)

		if r.URL.Path == app.config.Metrics.MetricsEndpoint {
			next.ServeHTTP(w, r)
			return
		}

		// Perform the first log about the incoming request.
		ip, err := realIP(r)
		if err != nil {
			app.logger.Errorw("retrieving real IP",
				"id", requestTrace.ID,
				"err", err,
			)
		}

		app.logger.Infow("incoming request",
			"id", requestTrace.ID,
			"start_time", requestTrace.Start,
			"remote_addr", r.RemoteAddr,
			"real_ip", ip,
			"URL", r.URL,
			"method", r.Method,
		)

		// Pass the request to the next handler.
		next.ServeHTTP(w, r)

		// After the request handling produce another log. Note that some values could not
		// be present since is the responsibility of other http handlers to enrich the
		// trace, even if this is not mandatory. Logs are produced with different
		// severity based on the HTTP code of the response.
		end := time.Now().UTC()
		fields := []interface{}{
			"id", requestTrace.ID,
			"http_code", requestTrace.HttpCode,
			"end_time", end,
			"duration_ms", end.Sub(requestTrace.Start).Milliseconds(),
		}
		if requestTrace.PrivateErr != nil {
			fields = append(fields, "private_err", requestTrace.PrivateErr)
		}

		switch requestTrace.HttpCode / 100 {
		case 0, 1, 2, 3:
			app.logger.Infow("request completed", fields...)
		case 4:
			app.logger.Warnw("request completed", fields...)
		case 5:
			app.logger.Errorw("request error", fields...)
		}
	}))
}

// The metrics middleware is used to register metrics (scraped by Prometheus) of incoming HTTP
// requests. Currently two metrics are registered: the count of the HTTP requests (divided by
// path and HTTP code) and the latency of the responses (divided by path). The scraping
// endpoint itself is not monitored.
func (app *application) metrics(next http.Handler) http.Handler {

	// Declare and register the counter of HTTP requests.
	requestCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_http_request",
			Help: "Counter of HTTP requests.",
		},
		[]string{"path", "code"},
	)
	if err := prometheus.Register(requestCount); err != nil {
		panic(err)
	}

	// Declare and register the histogram of HTTP responses latency.
	requestsLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_http_requests_duration_milliseconds",
			Help:    "Histogram of latencies for HTTP requests",
			Buckets: []float64{0.1, 1, 10, 100, 250, 500, 1000, 2500, 5000, 10000},
		},
		[]string{"path"},
	)
	if err := prometheus.Register(requestsLatency); err != nil {
		panic(err)
	}

	// Wrap the returned middleware in the tracing middleware, that is, before invoking
	// the function call the tracing function logic.
	return app.tracing(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTrace := tracing.TraceFromRequestCtx(r)
		next.ServeHTTP(w, r)

		path := r.URL.Path
		if path == app.config.Metrics.MetricsEndpoint {
			return
		}

		requestCount.WithLabelValues(path, fmt.Sprintf("%d", requestTrace.HttpCode)).Inc()
		requestsLatency.WithLabelValues(path).Observe(float64(time.Since(requestTrace.Start).Milliseconds()))
	}))
}

// This middleware is a wrapper around the two possibles rate-limiting middlewares.
// App configuration will dictate which strategy is applied. It is a no-op if
// rate-limiting is not enabled.
func (app *application) rateLimit(next http.Handler) http.Handler {
	if !app.config.RateLimit.Enabled {
		return next
	}

	if app.config.RateLimit.PerIp {
		return app.ipRateLimit(next)
	} else {
		return app.globalRateLimit(next)
	}
}

// The globalRateLimit middleware applies a rate limit control mechanism to the provided
// http handler. Rate limiting requests is particularly important to avoid server overloads.
// Different strategies could be used depending on how the app is deployed. Rate-limiting
// could be performed globally (this middleware) or per-IP (take a look below).
func (app *application) globalRateLimit(next http.Handler) http.Handler {

	// Initialize a new rate limiter which allows an average of 'n' requests
	// per second, with a maximum of 'm' requests in a single burst. Then
	// return a closure that can access the limiter variable.
	limiter := rate.NewLimiter(
		rate.Limit(app.config.RateLimit.Rps),
		app.config.RateLimit.Burst,
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Call limiter.Allow() to see if the request is permitted, and if
		// it's not return a 429 Too Many Requests response.
		if !limiter.Allow() {
			app.rateLimitExceededResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Using the per-IP rate-limiting pattern will only makes sens if your API application is directly
// exposed to clients on a single machine. If your infrastructure is distributed, with your app
// running on multiple servers behind a load balancer/reverse-proxy, another approach must be
// used. As an example, HAProxy or Nginx could take care of rate limiting directly. Alternatively,
// you could use a fast database like Redis to maintain a request count for clients, running on
// a server which all your application servers can communicate with.
func (app *application) ipRateLimit(next http.Handler) http.Handler {

	// Define a ipLimiter struct to hold the rate limiter and last seen time for each
	// client.
	type ipLimiter struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	// We keep in memory a map of IPs -> ipLimiters, the map must be accessed with a
	// mutex to avoid concurrency issues. Additionally, a background goroutine is
	// started, which removes old entries from the clients map once every minute.
	var (
		mu      sync.Mutex
		clients = make(map[string]*ipLimiter)
	)

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
			clients[ip] = &ipLimiter{
				limiter: rate.NewLimiter(
					rate.Limit(app.config.RateLimit.Rps),
					app.config.RateLimit.Burst,
				),
			}
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

func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Set a Vary: Origin response header to warn any caches that the
		// response may be different based on different origins. The same is
		// true for Vary: Access-Control-Request-Method.
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")

		// CORS requests have the Origin header set. If it is not present the
		// request is not CORS so proceed normally. Note however that if the
		// request is a CORS one but the origin is not included in our trusted
		// origins list, the request will be served as usual.
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		for _, trustedOrigin := range app.config.Cors.TrustedOrigins {
			if origin != trustedOrigin {
				continue
			}

			// Set this header to communicate to the browser that it's ok to read the response. A
			// wildcard (*) could be used here, but not in the case where the credentials are allowed
			// (like below).
			w.Header().Set("Access-Control-Allow-Origin", origin)

			// If your API endpoint requires credentials (cookies or HTTP basic authentication) you
			// should also set an Access-Control-Allow-Credentials: true header in your responses.
			// If you don’t set this header, then the web browser will prevent any cross-origin
			// responses with credentials from being read by JavaScript.
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			// Check if the request has the HTTP method OPTIONS and contains the "Access-Control-Request-Method"
			// header. If it does, then we treat it as a CORS preflight request (and normally it is).
			// If the request doesn't have them, it is a simple CORS request. The purpose of preflight
			// requests is to determine whether the real cross-origin request will be permitted or not.
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {

				// For preflight requests we must authorize non-CORS safe headers and HTTP methods
				// not allowed for simple CORS requests. Also the 'Access-Control-Allow-Origin' is
				// vital for preflight requests, but we have already set it previously.
				w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST, PUT, PATCH, DELETE")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.WriteHeader(http.StatusOK)
				return
			}
		}

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
