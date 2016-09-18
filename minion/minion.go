package minion

import (
	"sync"
	"time"
	"fmt"
	"errors"
	svc "github.com/qiuyesuifeng/tidb-demo/service"
	etcd "github.com/coreos/etcd/client"
	"strings"
	"github.com/qiuyesuifeng/tidb-demo/registry"
	"github.com/ngaut/log"
	"github.com/qiuyesuifeng/tidb-demo/minion/proc"
	"github.com/qiuyesuifeng/tidb-demo/minion/machine"
)

const (
	shutdownTimeout = time.Minute
)

var (
	stopc   chan struct{}  // used to terminate all other goroutines
	wg      sync.WaitGroup // used to co-ordinate shutdown
	running bool           = false

	Agent      *Agent
	Reconciler *AgentReconciler
	Publisher  *ProcessStatePublisher
	Heartbeat  *AgentHeartbeat
)

func Init(cfg *Config) error {
	if IsRunning() {
		return errors.New("Not allowed to initialize a running server")
	}
	agentTTL, err := time.ParseDuration(cfg.AgentTTL)
	if err != nil {
		return err
	}

	// init registry driver of etcd
	etcdAddrs := strings.Join(TrimAddrs(cfg.EtcdServers), ",")
	etcdTimeout := time.Duration(cfg.EtcdRequestTimeout) * time.Millisecond
	etcdPrefix := cfg.EtcdKeyPrefix
	etcdCfg := etcd.Config{
		Endpoints: cfg.EtcdServers,
		Transport: etcd.DefaultTransport,
	}
	etcdClient, err := etcd.New(etcdCfg)
	if err != nil {
		return err
	}
	kAPI := etcd.NewKeysAPI(etcdClient)
	reg := registry.NewEtcdRegistry(kAPI, etcdPrefix, etcdTimeout, etcdAddrs)
	es := registry.NewEtcdEventStream(kAPI, etcdPrefix)

	// check whether or not the registry is bootstrapped
	if ok := reg.IsBootstrapped(cfg); !ok {
		if err := reg.Bootstrap(); err != nil {
			log.Fatalf("Bootstarp failed, error: %v", err)
		}
		log.Info("Etcd registry bootstrapped successfully")
	}

	// register services in cluster
	svc.RegisterServices()
	// init local processes manager
	procMgr := proc.NewProcessManager()
	// init this machine
	mach, err := machine.NewMachineFromConfig(cfg)
	if err != nil {
		return err
	}
	// create agent
	Agent = NewAgent(reg, procMgr, mach, agentTTL)

	// reconciler drives the local process's state towards the desired state
	// stored in the Registry.
	Reconciler = NewReconciler(reg, es, Agent)
	Publisher = NewProcessStatePublisher(reg, Agent, agentTTL)
	Heartbeat = NewAgentHeartbeat(reg, Agent, agentTTL)

	log.Infof("Server initialized successfully")
	return nil
}

func Run(cfg *Config) (err error) {
	if IsRunning() {
		err = errors.New("Server is already running, cannot call to run repeatly")
		return
	}

	if err := Agent.BirthCry(); err != nil {
		log.Fatalf("Start server failed, %v", err)
	}

	stopc = make(chan struct{})
	wg = sync.WaitGroup{}
	components := []func(){
		func() { Reconciler.Run(stopc) },
		func() { Publisher.Run(stopc) },
		func() { Heartbeat.Run(stopc) },
		func() { Agent.Mach.Monitor(stopc) },
	}

	for _, f := range components {
		f := f
		wg.Add(1)
		go func() {
			f()
			wg.Done()
		}()
	}

	log.Infof("Server started successfully")
	switchStateToRunning()
	return
}

func Kill() (err error) {
	if !IsRunning() || stopc == nil {
		err = errors.New("The server is not running, cannot be killed")
		return
	}

	close(stopc)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(shutdownTimeout):
		err = errors.New("Timed out waiting for server to shutdown")
		return
	}

	log.Infof("Tidemo minion stopped")
	switchStateToStopped()
	return
}

func Purge() {
}

func Dump(cfg *Config) (dumpinfo []byte, err error) {
	err = nil
	dumpinfo = []byte(fmt.Sprintf("%v", cfg))
	log.Infof("Finished dumping server status")
	return
}

func IsRunning() bool {
	return running
}

func switchStateToStopped() {
	running = false
}

func switchStateToRunning() {
	running = true
}
