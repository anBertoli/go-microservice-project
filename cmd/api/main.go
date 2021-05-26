package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/anBertoli/snap-vault/pkg/mailer"
	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/services/galleries"
	"github.com/anBertoli/snap-vault/services/images"
	"github.com/anBertoli/snap-vault/services/users"
)

func main() {
	cfg, err := parseConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if cfg.DisplayVersion {
		fmt.Printf("API version: %s\n", version)
		return
	}

	logger := makeLogger(cfg.Env == "dev").Sugar()

	db, err := openDB(cfg)
	if err != nil {
		logger.Fatalf("cannot open db connection: %v", err)
	}
	storage, err := store.New(db, cfg.Storage.Root)
	if err != nil {
		logger.Fatalw("creating storage", "err", err)
	}

	var usersService users.Service
	usersService = &users.UsersService{Store: storage}
	usersService = &users.ValidationMiddleware{Next: usersService}
	usersService = &users.AuthMiddleware{
		Next:  usersService,
		Store: storage,
	}

	var galleriesService galleries.Service
	galleriesService = galleries.NewGalleriesService(storage, logger, 20)
	galleriesService = &galleries.StatsMiddleware{Store: storage.Stats, Next: galleriesService}
	galleriesService = &galleries.ValidationMiddleware{Next: galleriesService}
	galleriesService = &galleries.AuthMiddleware{
		Next:  galleriesService,
		Store: storage,
	}

	var imagesService images.Service
	imagesService = &images.ImagesService{Store: storage}
	imagesService = &images.StatsMiddleware{Store: storage.Stats, Next: imagesService, MaxBytes: cfg.Storage.MaxSpace}
	imagesService = &images.ValidationMiddleware{Next: imagesService}
	imagesService = &images.AuthMiddleware{Next: imagesService}

	mailer := mailer.New(cfg.Smtp.Host, cfg.Smtp.Port, cfg.Smtp.Username, cfg.Smtp.Password, cfg.Smtp.Sender)

	app := application{
		users:     usersService,
		galleries: galleriesService,
		images:    imagesService,
		mailer:    mailer,
		store:     storage,
		logger:    logger,
		config:    cfg,
	}

	err = app.serve()
	if err != nil {
		logger.Fatalw("shutting down server", "err", err)
	}
}

func openDB(cfg config) (*sqlx.DB, error) {
	// Create an empty connection pool, using the DSN from the config struct.
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

	// Create a context with a 5-second timeout deadline.
	// Use PingContext() to establish a new connection to the database, passing in the
	// context we created above as a parameter. If the connection couldn't be
	// established successfully within the 5 second deadline, then this will return an
	// error.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}

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
