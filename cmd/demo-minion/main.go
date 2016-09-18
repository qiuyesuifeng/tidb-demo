package main

import (
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ngaut/log"
	"github.com/qiuyesuifeng/tidb-demo/minion"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UTC().UnixNano())

	// Parse configuration from command-line arguments, environment variables or the config file of "minion.conf"
	cfg, err := minion.ParseFlag()
	if err != nil {
		log.Fatalf("Parsing configuration flags failed: %v", err)
	}
	log.Debugf("Load configuration successfully, %v", cfg)

	// Initialize server
	if err := minion.Init(cfg); err != nil {
		log.Fatalf("Failed to initializing tidemo minion from configuration, %v", err)
	}

	// Start tidemo-minion as daemon
	if err := minion.Run(cfg); err != nil {
		log.Fatalf("Failed to run tidemo minion server, %v", err)
	}

	shutdown := func() {
		log.Infof("Gracefully shutting down")
		minion.Kill()
		minion.Purge()
		os.Exit(0)
	}

	restart := func() {
		log.Infof("Restarting server now")
		minion.Kill()
		minion.Purge()

		// reload configuration file
		cfg, err := minion.ParseFlag()
		if err != nil {
			log.Fatalf("Parsing configuration flags failed: %v", err)
		}
		if err := minion.Init(cfg); err != nil {
			log.Fatalf("Failed to initializing tidemo minion from configuration, %v", err)
		}
		if err := minion.Run(cfg); err != nil {
			log.Fatalf("Failed to run tidemo minion, %v", err)
		}
	}

	dumpStatus := func() {
		log.Infof("start dumping server status")
		status, err := minion.Dump(cfg)
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
