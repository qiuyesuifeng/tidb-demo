package api

import (
	"encoding/json"

	"github.com/qiuyesuifeng/tidb-demo/master"
	"github.com/qiuyesuifeng/tidb-demo/pkg/utils"
	"github.com/qiuyesuifeng/tidb-demo/proc"
	"github.com/qiuyesuifeng/tidb-demo/schema"
)

type ProcessController struct {
	baseController
}

func (c *ProcessController) FindAllProcesses() {
	status, err := master.Agent.ListAllProcesses()
	if err != nil {
		c.ServeError(500, err.Error())
	}
	procs := []*schema.Process{}
	for _, s := range status {
		procs = append(procs, buildProcessModel(s))
	}
	c.Data["json"] = procs
	c.ServeJSON()
}

func (c *ProcessController) StartNewProcess() {
	var body schema.Process
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &body)
	if err != nil {
		c.ServeError(500, err.Error())
	}
	if len(body.SvcName) == 0 || len(body.MachID) == 0 {
		c.ServeError(500, "Request parameters 'svcName' or 'MachID' is necessary")
	}
	runinfo := &proc.ProcessRunInfo{
		Executor:    body.Executor,
		Command:     body.Command,
		Args:        body.Args,
		Environment: transformEnvironmentsToMap(body.Environments),
	}
	if err := master.Agent.StartNewProcess(body.MachID, body.SvcName, runinfo); err != nil {
		c.ServeError(500, err.Error())
	}
	// TODO: return the last status of process
	c.Data["json"] = body
	c.ServeJSON()
}

func (c *ProcessController) FindByHost() {
	machID := c.GetString("machID")
	if len(machID) == 0 {
		c.Abort("400")
	}
	status, err := master.Agent.ListProcessesByMachID(machID)
	if err != nil {
		c.ServeError(500, err.Error())
	}
	procs := []*schema.Process{}
	for _, s := range status {
		procs = append(procs, buildProcessModel(s))
	}
	c.Data["json"] = procs
	c.ServeJSON()
}

func (c *ProcessController) FindByService() {
	svcName := c.GetString("svcName")
	if len(svcName) == 0 {
		c.Abort("400")
	}
	status, err := master.Agent.ListProcessesBySvcName(svcName)
	if err != nil {
		c.ServeError(500, err.Error())
	}
	procs := []*schema.Process{}
	for _, s := range status {
		procs = append(procs, buildProcessModel(s))
	}
	c.Data["json"] = procs
	c.ServeJSON()
}

func (c *ProcessController) FindProcess() {
	procID := c.Ctx.Input.Param(":procID")
	if len(procID) == 0 {
		c.Abort("400")
	}
	s, err := master.Agent.ListProcess(procID)
	if err != nil {
		c.ServeError(500, err.Error())
	}
	c.Data["json"] = buildProcessModel(s)
	c.ServeJSON()
}

func (c *ProcessController) DestroyProcess() {
	procID := c.Ctx.Input.Param(":procID")
	if len(procID) == 0 {
		c.Abort("400")
	}
	err := master.Agent.DestroyProcess(procID)
	if err != nil {
		c.ServeError(500, err.Error())
	}
	c.Data["json"] = &schema.Process{
		ProcID: procID,
	}
	c.ServeJSON()
}

func (c *ProcessController) StartProcess() {
	procID := c.Ctx.Input.Param(":procID")
	if len(procID) == 0 {
		c.Abort("400")
	}
	err := master.Agent.StartProcess(procID)
	if err != nil {
		c.ServeError(500, err.Error())
	}
	c.Data["json"] = &schema.Process{
		ProcID:       procID,
		DesiredState: proc.StateStarted.String(),
	}
	c.ServeJSON()
}

func (c *ProcessController) StopProcess() {
	procID := c.Ctx.Input.Param(":procID")
	if len(procID) == 0 {
		c.Abort("400")
	}
	err := master.Agent.StopProcess(procID)
	if err != nil {
		c.ServeError(500, err.Error())
	}
	c.Data["json"] = &schema.Process{
		ProcID:       procID,
		DesiredState: proc.StateStopped.String(),
	}
	c.ServeJSON()
}

func buildProcessModel(s *proc.ProcessStatus) *schema.Process {
	p := &schema.Process{
		ProcID:       s.ProcID,
		SvcName:      s.SvcName,
		MachID:       s.MachID,
		DesiredState: s.DesiredState.String(),
		CurrentState: s.CurrentState.String(),
		IsAlive:      s.IsAlive,
		Endpoints:    utils.EndpointsToStrings(s.RunInfo.Endpoints),
		Executor:     s.RunInfo.Executor,
		Command:      s.RunInfo.Command,
		Args:         s.RunInfo.Args,
		Environments: transformMapToEnvironments(s.RunInfo.Environment),
		PublicIP:     s.RunInfo.HostIP,
		HostName:     s.RunInfo.HostName,
		HostMeta: schema.HostMeta{
			Region:     s.RunInfo.HostRegion,
			Datacenter: s.RunInfo.HostIDC,
		},
		Port:     0,
		Protocol: "",
	}
	return p
}
