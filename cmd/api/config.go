package main

import (
	"encoding/json"
	"flag"
	"os"
)

var (
	version = "unknown"
)

// Define a config struct to hold all the configuration settings for our application.
// For now, the only configuration settings will be the network port that we want the
// server to listen on, and the name of the current operating environment for the
// application (development, staging, production, etc.). We will read in these
// configuration settings from command-line flags when the application starts.
type config struct {
	Port int    `json:"port"`
	Env  string `json:"env"`
	Db   struct {
		Dsn          string `json:"dsn"`
		MaxOpenConns int    `json:"max_open_conns"`
		MaxIdleConns int    `json:"max_idle_conns"`
		MaxIdleTime  int    `json:"max_idle_time"`
	} `json:"db"`
	RateLimit struct {
		Enabled bool    `json:"enabled"`
		PerIp   bool    `json:"per_ip"`
		Rps     float64 `json:"rps"`
		Burst   int     `json:"burst"`
	} `json:"rate-limit"`
	Smtp struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		Sender   string `json:"sender"`
	} `json:"smtp"`
	Storage struct {
		Root     string `json:"root"`
		MaxSpace int64  `json:"max_space"`
	} `json:"storage"`
	Cors struct {
		TrustedOrigins []string `json:"trusted_origins"`
	} `json:"cors"`
	DisplayVersion bool // not from config file
}

func parseConfig() (config, error) {
	var cfg config

	version := flag.Bool("version", false, "Display version and exit")
	configPath := flag.String("config", "./conf/api.dev.json", "Path to config file")
	flag.Parse()

	configBytes, err := os.ReadFile(*configPath)
	if err != nil {
		return config{}, err
	}
	err = json.Unmarshal(configBytes, &cfg)
	if err != nil {
		return config{}, err
	}
	cfg.DisplayVersion = *version

	return cfg, nil
}
