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
	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/services/galleries"
	"github.com/anBertoli/snap-vault/services/images"
	"github.com/anBertoli/snap-vault/services/users"
)

type application struct {
	users     users.Service
	images    images.Service
	galleries galleries.Service
	mailer    mailer.Mailer
	store     store.Store
	logger    *zap.SugaredLogger
	bgTasks   sync.WaitGroup
	config    config
}

func (app *application) handler() http.Handler {
	router := mux.NewRouter()

	router.Methods(http.MethodPost).Path("/v1/users/register").HandlerFunc(app.registerUserHandler)
	router.Methods(http.MethodGet).Path("/v1/users/activate").HandlerFunc(app.activateUserHandler)
	router.Methods(http.MethodGet).Path("/v1/users/me").HandlerFunc(app.getUserAccountHandler)
	router.Methods(http.MethodGet).Path("/v1/users/stats").HandlerFunc(app.getUserStatsHandler)

	router.Methods(http.MethodGet).Path("/v1/users/keys").HandlerFunc(app.listUserKeysHandler)
	router.Methods(http.MethodPost).Path("/v1/users/keys").HandlerFunc(app.addUserKeyHandler)
	router.Methods(http.MethodPut).Path("/v1/users/keys/{id}").HandlerFunc(app.editKeyPermissionsHandler)
	router.Methods(http.MethodDelete).Path("/v1/users/keys/{id}").HandlerFunc(app.deleteUserKeyHandler)
	router.Methods(http.MethodPost).Path("/v1/users/recover-keys").HandlerFunc(app.genKeyRecoveryTokenHandler)
	router.Methods(http.MethodGet).Path("/v1/users/recover-keys").HandlerFunc(app.recoverKeyHandler)

	router.Methods(http.MethodGet).Path("/v1/galleries").HandlerFunc(app.listGalleriesHandler)
	router.Methods(http.MethodGet).Path("/v1/galleries/{id}").HandlerFunc(app.getGalleryHandler)
	router.Methods(http.MethodPost).Path("/v1/galleries").HandlerFunc(app.createGalleriesHandler)
	router.Methods(http.MethodPut).Path("/v1/galleries/{id}").HandlerFunc(app.updateGalleryHandler)
	router.Methods(http.MethodDelete).Path("/v1/galleries/{id}").HandlerFunc(app.deleteGalleryHandler)

	router.Methods(http.MethodGet).Path("/v1/galleries/{gallery-id}/images").HandlerFunc(app.listImagesHandler)
	router.Methods(http.MethodGet).Path("/v1/galleries/images/{image-id}").HandlerFunc(app.getImageHandler)
	router.Methods(http.MethodPost).Path("/v1/galleries/{gallery-id}/images").HandlerFunc(app.createImageHandler)
	router.Methods(http.MethodPut).Path("/v1/galleries/images/{image-id}").HandlerFunc(app.editImageHandler)
	router.Methods(http.MethodDelete).Path("/v1/galleries/images/{image-id}").HandlerFunc(app.deleteImageHandler)

	router.Methods(http.MethodGet).Path("/v1/public/galleries").HandlerFunc(app.listPublicGalleriesHandler)
	router.Methods(http.MethodGet).Path("/v1/public/galleries/{id}").HandlerFunc(app.getPublicGalleryHandler)
	router.Methods(http.MethodGet).Path("/v1/public/images").HandlerFunc(app.listPublicImagesHandler)
	router.Methods(http.MethodGet).Path("/v1/public/images/{image-id}").HandlerFunc(app.getPublicImageHandler)

	router.Methods(http.MethodGet).Path("/v1/healthcheck").HandlerFunc(app.healthcheckHandler)
	router.Methods(http.MethodGet).Path("/v1/permissions").HandlerFunc(app.listPermissionsHandler)

	router.NotFoundHandler = http.HandlerFunc(app.routeNotFoundHandler)
	router.MethodNotAllowedHandler = http.HandlerFunc(app.methodNotAllowedHandler)

	handler := app.authenticate(router)

	if app.config.RateLimit.Enabled {
		if app.config.RateLimit.PerIp {
			handler = app.ipRateLimit(handler)
		} else {
			handler = app.globalRateLimit(handler)
		}
	}

	handler = app.enableCORS(handler)
	handler = app.recoverPanic(handler)
	handler = app.logging(handler)
	return handler
}

func (app *application) serve() error {
	// Declare a HTTP server using the same settings as in our main() function.
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.Port),
		Handler:      app.handler(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit
		app.logger.Infow("shutting down server", "signal", s.String())

		// Call Shutdown() on our server, passing in the context we just made.
		// Shutdown() will return nil if the graceful shutdown was successful, or an
		// error (which may happen because of a problem closing the listeners, or
		// because the shutdown didn't complete before the 5-second context deadline is
		// hit). We relay this return value to the shutdownError channel.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)

		// Call Wait() to block until our WaitGroup counter is zero --- essentially
		// blocking until the background goroutines have finished. Then we return nil on
		// the shutdownError channel, to indicate that the shutdown completed without
		// any issues.
		app.bgTasks.Wait()

		shutdownError <- err
	}()

	// Likewise log a "starting server" message.
	app.logger.Infow("starting server",
		"addr", srv.Addr,
		"env", app.config.Env,
	)

	// Calling Shutdown() on our server will cause ListenAndServe() to immediately
	// return a http.ErrServerClosed error. So if we see this error, it is actually a
	// good thing and an indication that the graceful shutdown has started. So we check
	// specifically for this, only returning the error if it is NOT http.ErrServerClosed.
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Otherwise, we wait to receive the return value from Shutdown() on the
	// shutdownError channel. If return value is an error, we know that there was a
	// problem with the graceful shutdown and we return the error.
	err = <-shutdownError
	if err != nil {
		return err
	}

	// At this point we know that the graceful shutdown completed successfully and we
	// log a "stopped server" message.
	app.logger.Infow("stopped server", "addr", srv.Addr)
	return nil
}

// The background() helper accepts an arbitrary function as a parameter.
func (app *application) background(fn func()) {
	// Launch a background goroutine, which it will execute
	// the arbitrary function that we passed as the parameter.
	app.bgTasks.Add(1)
	go func() {
		defer app.bgTasks.Done()
		fn()
	}()
}
