package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/ovh/canary-ng/internal"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// AppVersion stores application version at compilation time
var AppVersion string

// AppName to store application name
var AppName string = "canary-ng"

// GitCommit to set git commit at compilation time (can be empty)
var GitCommit string

// GoVersion to set Go version at compilation time
var GoVersion string

func main() {
	quiet := flag.Bool("quiet", false, "quiet mode")
	verbose := flag.Bool("verbose", false, "print more logs")
	debug := flag.Bool("debug", false, "print even more logs")
	version := flag.Bool("version", false, "print version")
	configFile := flag.String("config", AppName+".yaml", "configuration file name")
	flag.Parse()

	if *version {
		if AppVersion == "" {
			AppVersion = "unknown"
		}
		showVersion()
		return
	}

	config, err := internal.NewConfig(*configFile)
	if err != nil {
		fmt.Printf("could not create configuration: %v\n", err)
		os.Exit(1)
	}

	// Parse log level from config
	var logLevel slog.Level
	switch config.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		fmt.Printf("invalid log level %s\n", config.LogLevel)
		os.Exit(1)
	}

	// Eventually override log level from command line arguments
	if *debug {
		logLevel = slog.LevelDebug
	}
	if *verbose {
		logLevel = slog.LevelInfo
	}
	if *quiet {
		logLevel = slog.LevelError
	}

	var h slog.Handler
	switch config.LogFormat {
	case "json":
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	case "text":
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	default:
		fmt.Printf("invalid log format %s\n", config.LogFormat)
		os.Exit(1)
	}
	slog.SetDefault(slog.New(h))

	if len(config.Jobs) == 0 {
		slog.Error("no job configured")
		os.Exit(1)
	}

	reg := prometheus.NewRegistry()
	metrics := internal.NewMetrics(reg, config)

	for _, jobConfig := range config.Jobs {
		jobs, err := internal.NewJobs(jobConfig, metrics, config.QueryLabels, config.JobLabelName)
		if err != nil {
			slog.Error("could not create job", slog.Any("job", jobConfig.Name), slog.Any("error", err))
			continue
		}
		for _, j := range jobs {
			go j.Run()
		}
	}

	slog.Info(fmt.Sprintf("serving to %s%s", config.ListenAddr, config.Route))

	http.Handle(config.Route, promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	if err = http.ListenAndServe(config.ListenAddr, nil); err != nil {
		slog.Error("could not listen and serve", slog.Any("error", err))
		os.Exit(1)
	}
}

func showVersion() {
	if GitCommit != "" {
		AppVersion = fmt.Sprintf("%s-%s", AppVersion, GitCommit)
	}
	fmt.Printf("%s version %s (compiled with %s)\n", AppName, AppVersion, GoVersion)
}
