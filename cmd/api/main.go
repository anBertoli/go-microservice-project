package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/anBertoli/snap-vault/pkg/auth"
	"github.com/anBertoli/snap-vault/pkg/mailer"
	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/services/galleries"
	"github.com/anBertoli/snap-vault/services/images"
	"github.com/anBertoli/snap-vault/services/users"
)

func main() {
	// Obtain the configuration parsed from the config file. If only the version
	// is asked, print it and exit immediately.
	cfg, err := parseConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if cfg.DisplayVersion {
		fmt.Printf("API version: %s\n", version)
		return
	}

	// Create the logger to be used throughout the application, specifying the
	// format of the logs.
	logger := makeLogger(cfg.Env == "dev").Sugar()
	logger.Infof("configuration %s", cfg.Expose())

	// Open a pool of connection to the database.
	db, err := openDB(cfg)
	if err != nil {
		logger.Fatalf("cannot open db connection: %v", err)
	}

	// Instantiate the store struct that will be used to perform operations on the database.
	// The store needs the connection pool created above and the path of the directory where
	// images will be stored.
	storage, err := store.New(db, cfg.Storage.Root)
	if err != nil {
		logger.Fatalw("creating storage", "err", err)
	}

	// The authenticator is used to authenticate requests in several auth middlewares
	// that wrap our core services.
	authenticator := auth.Authenticator{Store: storage}

	// Declare a users.Service interface variable, then assign to it the core service of the users
	// package (it is a concrete value assigned to an interface). Decorate the interface with the
	// middlewares of the same package (which respect the same interface). This will create a chain
	// of service middlewares that provides specialized functionalities and enforce separation
	// of concerns.
	var usersService users.Service
	usersService = &users.UsersService{Store: storage}
	usersService = &users.ValidationMiddleware{Service: usersService}
	usersService = &users.AuthMiddleware{Service: usersService, Auth: authenticator}

	// Repeat the same process for the galleries service.
	var galleriesService galleries.Service
	galleriesService = galleries.NewGalleriesService(storage, logger, 20)
	galleriesService = &galleries.StatsMiddleware{Store: storage.Stats, Service: galleriesService}
	galleriesService = &galleries.ValidationMiddleware{Service: galleriesService}
	galleriesService = &galleries.AuthMiddleware{Service: galleriesService, Auth: authenticator}

	// Repeat the same process for the images service.
	var imagesService images.Service
	imagesService = &images.ImagesService{Store: storage}
	imagesService = &images.StatsMiddleware{Store: storage.Stats, Service: imagesService, MaxBytes: cfg.Storage.MaxSpace}
	imagesService = &images.ValidationMiddleware{Service: imagesService}
	imagesService = &images.AuthMiddleware{Service: imagesService, Auth: authenticator}

	mailer := mailer.New(cfg.Smtp.Host, cfg.Smtp.Port, cfg.Smtp.Username, cfg.Smtp.Password, cfg.Smtp.Sender)

	// Create the application struct, the entity that represent our JSON API. It provides
	// the HTTP handlers as methods along several helper functions.
	app := application{
		users:     usersService,
		galleries: galleriesService,
		images:    imagesService,
		mailer:    mailer,
		logger:    logger,
		config:    cfg,
	}

	// Start listening of the address:port specified by the configs.
	err = app.serve()
	if err != nil {
		logger.Fatalw("shutting down server", "err", err)
	}
}

// Create a database connection pool and configure it.
func openDB(cfg config) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", cfg.Db.Dsn)
	if err != nil {
		return nil, err
	}

	// Set the maximum number of  (in - use + idle) connections in the pool.Note that
	// passing a value less than or equal to 0 will mean there is no limit.
	db.SetMaxOpenConns(cfg.Db.MaxOpenConns)

	// Set the maximum number of idle connections in the pool. Again, passing a value
	// less than or equal to 0 will mean there is no limit.
	db.SetMaxIdleConns(cfg.Db.MaxIdleConns)

	// Set the maximum idle timeout.
	db.SetConnMaxIdleTime(time.Duration(cfg.Db.MaxIdleTime) * time.Minute)

	// Ping the database to test the connection.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// Instantiate the appropriate logger based on the mode. The dev mode will result in colorized
// and more readable log entries, while production logs will be entirely JSON-formatted.
func makeLogger(dev bool) *zap.Logger {
	var zapLogger *zap.Logger
	if dev {
		config := zap.NewDevelopmentEncoderConfig()
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		zapLogger = zap.New(
			zapcore.NewCore(
				zapcore.NewConsoleEncoder(config), os.Stdout, zap.DebugLevel,
			),
		)
	} else {
		config := zap.NewProductionEncoderConfig()
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		zapLogger = zap.New(
			zapcore.NewCore(
				zapcore.NewJSONEncoder(config), os.Stdout, zap.DebugLevel,
			),
		)
	}
	return zapLogger
}
