package main

import (
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ngaut/log"
	"github.com/qiuyesuifeng/tidb-demo/api"
	"github.com/qiuyesuifeng/tidb-demo/master"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UTC().UnixNano())

	// Parse configuration from command-line arguments, environment variables or the config file of "master.conf"
	cfg, err := master.ParseFlag()
	if err != nil {
		log.Fatalf("Parsing configuration flags failed: %v", err)
	}
	log.Debugf("Load configuration successfully, %v", cfg)

	// Initialize server
	if err := master.Init(cfg); err != nil {
		log.Fatalf("Failed to initializing tidemo master from configuration, %v", err)
	}

	// Start tidemo-master as daemon
	if err := master.Run(cfg); err != nil {
		log.Fatalf("Failed to run tidemo master server, %v", err)
	}

	// Start HTTP server for a set of REST APIs
	go api.ServeHttp(cfg.APIPort)
	log.Infof("API server listening at port: %d", cfg.APIPort)

	shutdown := func() {
		log.Infof("Gracefully shutting down")
		master.Kill()
		master.Purge()
		os.Exit(0)
	}

	restart := func() {
		log.Infof("Restarting server now")
		master.Kill()
		master.Purge()

		// reload configuration file
		cfg, err := master.ParseFlag()
		if err != nil {
			log.Fatalf("Parsing configuration flags failed: %v", err)
		}
		if err := master.Init(cfg); err != nil {
			log.Fatalf("Failed to initializing tidemo master from configuration, %v", err)
		}
		if err := master.Run(cfg); err != nil {
			log.Fatalf("Failed to run tidemo master, %v", err)
		}
	}

	dumpStatus := func() {
		log.Infof("start dumping server status")
		status, err := master.Dump(cfg)
		if err != nil {
			log.Errorf("Failed to dump server status: %v", err)
			return
		}
		if _, err := os.Stdout.Write(status); err != nil {
			log.Errorf("Failed to dump server status: %v", err)
			return
		}
		os.Stdout.Write([]byte("\n"))
	}

	signals := map[os.Signal]func(){
		syscall.SIGHUP:  restart,
		syscall.SIGTERM: shutdown,
		syscall.SIGINT:  shutdown,
		syscall.SIGUSR1: dumpStatus,
	}
	sigchan := make(chan os.Signal, 1)
	for k := range signals {
		signal.Notify(sigchan, k)
	}

	for true {
		sig := <-sigchan
		if handler, ok := signals[sig]; ok {
			handler()
		}
	}
}
