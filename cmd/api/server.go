package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/anBertoli/snap-vault/pkg/mailer"
	"github.com/anBertoli/snap-vault/services/galleries"
	"github.com/anBertoli/snap-vault/services/images"
	"github.com/anBertoli/snap-vault/services/users"
)

// The application struct represents the API and holds all necessary dependencies for
// our endpoints. HTTP handlers are defined as methods on this struct.
type application struct {
	users     users.Service
	images    images.Service
	galleries galleries.Service
	mailer    mailer.Mailer
	logger    *zap.SugaredLogger
	bgTasks   sync.WaitGroup
	config    config
}

// The handler() method returns the server handler, that is, it registers all the HTTP API
// endpoints on the router and returns it. The function is also useful as a 'map' of our
// web service.
func (app *application) handler() http.Handler {
	router := mux.NewRouter()

	router.Methods(http.MethodPost).Path("/v1/users/register").HandlerFunc(app.registerUserHandler)
	router.Methods(http.MethodGet).Path("/v1/users/activate").HandlerFunc(app.activateUserHandler)
	router.Methods(http.MethodGet).Path("/v1/users/me").HandlerFunc(app.getUserAccountHandler)
	router.Methods(http.MethodGet).Path("/v1/users/stats").HandlerFunc(app.getUserStatsHandler)
	router.Methods(http.MethodPost).Path("/v1/users/recover-keys").HandlerFunc(app.genKeyRecoveryTokenHandler)
	router.Methods(http.MethodGet).Path("/v1/users/recover-keys").HandlerFunc(app.recoverKeyHandler)

	router.Methods(http.MethodGet).Path("/v1/users/keys").HandlerFunc(app.listUserKeysHandler)
	router.Methods(http.MethodPost).Path("/v1/users/keys").HandlerFunc(app.addUserKeyHandler)
	router.Methods(http.MethodPut).Path("/v1/users/keys/{id}").HandlerFunc(app.editKeyPermissionsHandler)
	router.Methods(http.MethodDelete).Path("/v1/users/keys/{id}").HandlerFunc(app.deleteUserKeyHandler)

	router.Methods(http.MethodGet).Path("/v1/galleries").HandlerFunc(app.listGalleriesHandler)
	router.Methods(http.MethodGet).Path("/v1/galleries/{id}").HandlerFunc(app.getGalleryHandler)
	router.Methods(http.MethodPost).Path("/v1/galleries").HandlerFunc(app.createGalleriesHandler)
	router.Methods(http.MethodPut).Path("/v1/galleries/{id}").HandlerFunc(app.updateGalleryHandler)
	router.Methods(http.MethodDelete).Path("/v1/galleries/{id}").HandlerFunc(app.deleteGalleryHandler)

	router.Methods(http.MethodGet).Path("/v1/galleries/{gallery-id}/images").HandlerFunc(app.listGalleryImagesHandler)
	router.Methods(http.MethodGet).Path("/v1/galleries/images/{image-id}").HandlerFunc(app.getImageHandler)
	router.Methods(http.MethodPost).Path("/v1/galleries/{gallery-id}/images").HandlerFunc(app.createImageHandler)
	router.Methods(http.MethodPut).Path("/v1/galleries/images/{image-id}").HandlerFunc(app.editImageHandler)
	router.Methods(http.MethodDelete).Path("/v1/galleries/images/{image-id}").HandlerFunc(app.deleteImageHandler)

	router.Methods(http.MethodGet).Path("/v1/public/galleries").HandlerFunc(app.listPublicGalleriesHandler)
	router.Methods(http.MethodGet).Path("/v1/public/galleries/{gallery-id}").HandlerFunc(app.getPublicGalleryHandler)
	router.Methods(http.MethodGet).Path("/v1/public/galleries/{gallery-id}/images").HandlerFunc(app.listPublicGalleryImagesHandler)
	router.Methods(http.MethodGet).Path("/v1/public/images").HandlerFunc(app.listPublicImagesHandler)
	router.Methods(http.MethodGet).Path("/v1/public/images/{image-id}").HandlerFunc(app.getPublicImageHandler)

	router.Methods(http.MethodGet).Path("/v1/healthcheck").HandlerFunc(app.healthcheckHandler)
	router.Methods(http.MethodGet).Path("/v1/permissions").HandlerFunc(app.listPermissionsHandler)

	router.NotFoundHandler = http.HandlerFunc(app.routeNotFoundHandler)
	router.MethodNotAllowedHandler = http.HandlerFunc(app.methodNotAllowedHandler)

	// Apply middlewares to the global handler. Here the order matters, e.g. the enableCORS
	// middleware should be triggered before the rate limiting one. This because we want to
	// avoid the circumstance of a pre-flight request allowed and a 'real' request blocked
	// due to the rate limiting threshold reached.
	handler := app.extractKey(router)
	handler = app.rateLimit(handler)
	handler = app.enableCORS(handler)
	handler = app.logging(handler)
	return handler
}

func (app *application) serve() error {
	// Declare a HTTP server setting sensible default for different timeouts.
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", app.config.Address, app.config.Port),
		Handler:      app.handler(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdownError := make(chan error, 1)

	// This goroutine will block waiting for signals from the environment and/or the
	// command line. It will handle SIGINT and SIGTERM in order to gracefully
	// shutdown the server.
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit
		app.logger.Infow("shutting down server", "signal", s.String())

		// Call Shutdown() on our server, with a timeout context. Shutdown() will return nil
		// if the graceful shutdown was successful, or an error (which may happen because of
		// a problem closing the listeners, or because the shutdown didn't complete before
		// the 5-second context deadline is hit).
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)

		// Call Wait() to block until all background tasks are ended. This is a blocking
		// operation. Then send any error encountered during the previous shutdown in the
		// dedicated channel. After this, the shutdown is completed.
		app.bgTasks.Wait()

		shutdownError <- err
	}()

	app.logger.Infow("starting server",
		"addr", srv.Addr,
		"env", app.config.Env,
	)

	// Calling Shutdown() on our server will cause ListenAndServe() to immediately return
	// a http.ErrServerClosed error. So if we see this error, it is actually a good thing
	// and an indication that the graceful shutdown has started. So we check specifically
	// for this, only returning the error if it is NOT http.ErrServerClosed.
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Otherwise, we wait to receive the return value from Shutdown() on the shutdownError
	// channel. If the return value is an error, we know that there was a problem with
	// the graceful shutdown and we return the error.
	err = <-shutdownError
	if err != nil {
		return err
	}

	// At this point we know that the graceful shutdown completed successfully and we
	// log a "stopped server" message.
	app.logger.Infow("stopped server", "addr", srv.Addr)

	return nil
}

// The background() helper accepts an arbitrary function as a parameter
// and runs it as a background goroutine.
func (app *application) background(fn func()) {
	app.bgTasks.Add(1)
	go func() {
		defer app.bgTasks.Done()
		fn()
	}()
}
