package proc

import (
	"errors"
	"sync"

	"github.com/ngaut/log"
	"github.com/qiuyesuifeng/tidb-demo/pkg/utils"
)

type ProcMgr interface {
	CreateProcess(*ProcessStatus, map[string]string) (Proc, error)
	DestroyProcess(string) error
	StartProcess(string, map[string]string) error
	StopProcess(string) error
	AllProcess() map[string]Proc
	AllActiveProcess() map[string]Proc
	TotalProcess() int
	TotalActiveProcess() int
	FindByProcID(string) Proc
	FindBySvcName(string) map[string]Proc
}

type processManager struct {
	procs   map[string]Proc
	rwMutex sync.RWMutex
}

func NewProcessManager() ProcMgr {
	return &processManager{
		procs: make(map[string]Proc),
	}
}

func buildProcessMeta(target *ProcessStatus) map[string]string {
	meta := make(map[string]string)
	meta["HOST_NAME"] = target.RunInfo.HostName
	meta["HOST_IP"] = target.RunInfo.HostIP
	meta["HOST_REGION"] = target.RunInfo.HostRegion
	meta["HOST_IDC"] = target.RunInfo.HostIDC
	meta["SERVICE"] = target.SvcName
	return meta
}

func (pm *processManager) CreateProcess(target *ProcessStatus, endpoints map[string]string) (Proc, error) {
	meta := buildProcessMeta(target)
	// TODO: stdout and stderr filepath should be assigned from client
	proc, err := NewProcess(target.ProcID, target.SvcName, target.RunInfo.Executor, target.RunInfo.Command, target.RunInfo.Args,
		"$SERVICE_$PROCID_$RUN.out", "$SERVICE_$PROCID_$RUN.err", target.RunInfo.Environment, meta, utils.GetDataDir())
	if err != nil {
		log.Errorf("Failed to create new local process, procID: %s, error: %v", target.ProcID, err)
		return nil, err
	}
	// if process's desiredstate is 'started', then start it
	if target.DesiredState == StateStarted {
		if err := proc.Start(endpoints); err != nil {
			log.Errorf("Failed to start local process, procID: %s, error: %v", target.ProcID, err)
			return nil, err
		}
	}
	pm.rwMutex.Lock()
	defer pm.rwMutex.Unlock()
	pm.procs[target.ProcID] = proc
	return proc, nil
}

func (pm *processManager) DestroyProcess(procID string) (err error) {
	pm.rwMutex.RLock()
	proc, ok := pm.procs[procID]
	pm.rwMutex.RUnlock()
	if !ok {
		return errors.New("Failed to destroy a local process, the procID not exists: " + procID)
	}

	if proc.State() == StateStarted {
		if err := proc.Stop(); err != nil {
			log.Errorf("Failed to stop local process, procID: %s, error: %v", procID, err)
			return err
		}
	}

	pm.rwMutex.Lock()
	delete(pm.procs, procID)
	pm.rwMutex.Unlock()
	return nil
}

func (pm *processManager) StartProcess(procID string, endpoints map[string]string) error {
	pm.rwMutex.RLock()
	proc, ok := pm.procs[procID]
	pm.rwMutex.RUnlock()
	if !ok {
		return errors.New("Failed to start a local process, the procID not exists: " + procID)
	}
	if proc.State() == StateStopped {
		if err := proc.Start(endpoints); err != nil {
			log.Errorf("Failed to start local process, procID: %s, error: %v", procID, err)
			return err
		}
	} else {
		log.Warnf("Process is already started, no need to be stopped, procID: %s", procID)
	}
	return nil
}

func (pm *processManager) StopProcess(procID string) error {
	pm.rwMutex.RLock()
	proc, ok := pm.procs[procID]
	pm.rwMutex.RUnlock()
	if !ok {
		return errors.New("Failed to start a local process, the procID not exists: " + procID)
	}
	if proc.State() == StateStarted {
		if err := proc.Stop(); err != nil {
			log.Errorf("Failed to stop local process, procID: %s, error: %v", procID, err)
			return err
		}
	} else {
		log.Warnf("Process is already started, no need to be stopped, procID: %s", procID)
	}
	return nil
}

func (pm *processManager) AllProcess() map[string]Proc {
	res := make(map[string]Proc)
	pm.rwMutex.RLock()
	defer pm.rwMutex.RUnlock()
	for k, v := range pm.procs {
		res[k] = v
	}
	return res
}

func (pm *processManager) AllActiveProcess() map[string]Proc {
	res := make(map[string]Proc)
	pm.rwMutex.RLock()
	defer pm.rwMutex.RUnlock()
	for k, v := range pm.procs {
		if v.IsActive() {
			res[k] = v
		}
	}
	return res
}

func (pm *processManager) TotalProcess() int {
	pm.rwMutex.RLock()
	defer pm.rwMutex.RUnlock()
	return len(pm.procs)
}

func (pm *processManager) TotalActiveProcess() int {
	res := 0
	pm.rwMutex.RLock()
	defer pm.rwMutex.RUnlock()
	for _, v := range pm.procs {
		if v.IsActive() {
			res++
		}
	}
	return res
}

func (pm *processManager) FindByProcID(procID string) Proc {
	pm.rwMutex.RLock()
	defer pm.rwMutex.RUnlock()
	if proc, ok := pm.procs[procID]; ok {
		return proc
	} else {
		return nil
	}
}

func (pm *processManager) FindBySvcName(svcName string) map[string]Proc {
	res := make(map[string]Proc)
	pm.rwMutex.RLock()
	defer pm.rwMutex.RUnlock()
	for k, v := range pm.procs {
		if v.GetSvcName() == svcName {
			res[k] = v
		}
	}
	return res
}
