package main

import (
	"encoding/json"
	"flag"
	"os"
)

var (
	version = "<unknown>"
)

// Define a config struct to hold all the configuration settings for our application.
// We will read in these configuration settings from a config file when the
// application starts.
type config struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
	Env     string `json:"env"`
	Db      struct {
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
	Metrics struct {
		MetricsEndpoint string `json:"metrics-endpoint"`
	} `json:"metrics"`
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
	PublicHostname string `json:"public_hostname"`
	DisplayVersion bool   // not from config file
}

// Parse command line flags and read in the config file at the provided path.
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

	// This is not from the config file.
	cfg.DisplayVersion = *version

	return cfg, nil
}
