package agent

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ngaut/log"
	"github.com/qiuyesuifeng/tidb-demo/machine"
	"github.com/qiuyesuifeng/tidb-demo/pkg/utils"
	"github.com/qiuyesuifeng/tidb-demo/proc"
	"github.com/qiuyesuifeng/tidb-demo/registry"
	"github.com/qiuyesuifeng/tidb-demo/service"
)

type Agent struct {
	Reg        registry.Registry
	ProcMgr    proc.ProcMgr
	Mach       machine.Machine
	publishch  chan []string
	procsCache map[string]*proc.ProcessStatus
	cacheMutex sync.RWMutex
}

func (a *Agent) Subscribe(procIDs []string) {
	if procIDs != nil && len(procIDs) > 0 {
		a.publishch <- procIDs
	}
}

func (a *Agent) Publish() chan []string {
	return a.publishch
}

func (a *Agent) SaveProcsToCache(procs map[string]*proc.ProcessStatus) {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()
	a.procsCache = procs
}

func (a *Agent) GetProcsFomeCache() map[string]*proc.ProcessStatus {
	a.cacheMutex.RLock()
	defer a.cacheMutex.RUnlock()
	return a.procsCache
}

func NewAgent(reg registry.Registry, pm proc.ProcMgr, m machine.Machine) *Agent {
	return &Agent{
		Reg:        reg,
		ProcMgr:    pm,
		Mach:       m,
		publishch:  make(chan []string, 10),
		procsCache: make(map[string]*proc.ProcessStatus),
	}
}

func (a *Agent) StartNewProcess(machID, svcName string, runinfo *proc.ProcessRunInfo) error {
	var hostIP string
	var hostName string
	var hostRegion string
	var hostIDC string
	var executor []string
	var command string
	var args []string
	var envs map[string]string
	var endpoints = map[string]utils.Endpoint{}

	// retrieve machine infomation from etcd
	if mach, err := a.Reg.Machine(machID); err == nil {
		hostIP = mach.MachInfo.PublicIP
		hostName = mach.MachInfo.HostName
		hostRegion = mach.MachInfo.HostRegion
		hostIDC = mach.MachInfo.HostIDC
		// check if the target machine is offline
		if !mach.IsAlive {
			e := fmt.Sprintf("Should not start new processes on a offline host, machID: %s, svcName: %s", machID, svcName)
			log.Error(e)
			return errors.New(e)
		}
	} else {
		return err
	}

	if svc, ok := service.Registered[svcName]; ok {
		ss := svc.Status()
		if len(runinfo.Executor) > 0 {
			executor = runinfo.Executor
		} else {
			executor = ss.Executor
		}
		if len(runinfo.Command) > 0 {
			command = runinfo.Command
		} else {
			command = ss.Command
		}
		if len(runinfo.Args) > 0 {
			args = runinfo.Args
		} else {
			args = ss.Args
		}
		if len(runinfo.Environment) > 0 {
			envs = runinfo.Environment
		} else {
			envs = ss.Environments
		}
		parsedEndpoints := svc.ParseEndpointFromArgs(args)
		for k, v := range parsedEndpoints {
			if len(v.IPAddr) == 0 {
				v.IPAddr = hostIP
			}
			endpoints[k] = v
		}
	} else {
		e := fmt.Sprintf("Unregistered service: %s", svcName)
		log.Error(e)
		return errors.New(e)
	}

	if err := a.Reg.NewProcess(machID, svcName, hostIP, hostName, hostRegion, hostIDC,
		executor, command, args, envs, endpoints); err != nil {
		e := fmt.Sprintf("Create new process failed in etcd, %s, %s, %v", machID, svcName, err)
		log.Error(e)
		return errors.New(e)
	}
	return nil
}

func (a *Agent) DestroyProcess(procID string) error {
	_, err := a.Reg.DeleteProcess(procID)
	if err != nil {
		log.Errorf("Delete process failed in etcd, %s, %v", procID, err)
	}
	return err
}

func (a *Agent) StartProcess(procID string) error {
	err := a.Reg.UpdateProcessDesiredState(procID, proc.StateStarted)
	if err != nil {
		log.Errorf("Change desired state of process to started failed, %s, %v", procID, err)
	}
	return err
}

func (a *Agent) StopProcess(procID string) error {
	err := a.Reg.UpdateProcessDesiredState(procID, proc.StateStopped)
	if err != nil {
		log.Errorf("Change desired state of process to stopped failed, %s, %v", procID, err)
	}
	return err
}

func (a *Agent) ListAllProcesses() (res map[string]*proc.ProcessStatus, err error) {
	res, err = a.Reg.Processes()
	if err != nil {
		log.Errorf("List all processes failed, %v", err)
	}
	return
}

func (a *Agent) ListProcessesByMachID(machID string) (res map[string]*proc.ProcessStatus, err error) {
	res, err = a.Reg.ProcessesOnMachine(machID)
	if err != nil {
		log.Errorf("List processes on specified machine, %s, %v", machID, err)
	}
	return
}

func (a *Agent) ListProcessesBySvcName(svcName string) (res map[string]*proc.ProcessStatus, err error) {
	res, err = a.Reg.ProcessesOfService(svcName)
	if err != nil {
		log.Errorf("List processes of specified service, %s, %v", svcName, err)
	}
	return
}

func (a *Agent) ListProcess(procID string) (res *proc.ProcessStatus, err error) {
	res, err = a.Reg.Process(procID)
	if err != nil {
		log.Errorf("List specified process failed, %s, %v", procID, err)
	}
	return
}

func (a *Agent) ListAllMachines() (res map[string]*machine.MachineStatus, err error) {
	res, err = a.Reg.Machines()
	if err != nil {
		log.Errorf("List all machines in cluster failed, %v", err)
	}
	return
}

func (a *Agent) ListMachine(machID string) (res *machine.MachineStatus, err error) {
	res, err = a.Reg.Machine(machID)
	if err != nil {
		log.Errorf("List specified machines infomation failed, %s, %v", machID, err)
	}
	return
}

func (a *Agent) BirthCry() error {
	status := a.Mach.Status()
	if err := a.Reg.RegisterMachine(status.MachID, status.MachInfo.HostName, status.MachInfo.HostRegion,
		status.MachInfo.HostIDC, status.MachInfo.PublicIP); err != nil {
		log.Errorf("Register machine status into etcd failed, %v", err)
		return err
	}
	return nil
}

func (a *Agent) ShowTiDBRealPerfermance() *service.TiDBPerfMetrics {
	// fetch a list of all processes exists in Ti-Cluster from agent's cache, which updated since last reconciling
	var cachedProcs = a.GetProcsFomeCache()
	if cachedProcs == nil {
		// no process exists in Ti-Cluster, or the agent just started a moment ago
		return &service.TiDBPerfMetrics{}
	}
	tidbService := service.Registered[service.TiDB_SERVICE].(*service.TiDBService)
	return tidbService.RetrieveRealPerformance(cachedProcs)
}
