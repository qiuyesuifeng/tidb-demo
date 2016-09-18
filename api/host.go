package api

import (
	"encoding/json"

	"github.com/qiuyesuifeng/tidb-demo/machine"
	"github.com/qiuyesuifeng/tidb-demo/master"
	"github.com/qiuyesuifeng/tidb-demo/schema"
)

type HostController struct {
	baseController
}

func (c *HostController) FindAllHosts() {
	status, err := master.Agent.ListAllMachines()
	if err != nil {
		c.ServeError(500, err.Error())
	}
	hosts := []*schema.Host{}
	for _, s := range status {
		hosts = append(hosts, buildHostModel(s))
	}
	c.Data["json"] = hosts
	c.ServeJSON()
}

func (c *HostController) FindHost() {
	machID := c.Ctx.Input.Param(":machID")
	if len(machID) == 0 {
		c.Abort("400")
	}
	m, err := master.Agent.ListMachine(machID)
	if err != nil {
		c.ServeError(500, err.Error())
	}
	c.Data["json"] = buildHostModel(m)
	c.ServeJSON()
}

func (c *HostController) SetHostMetaInfo() {
	machID := c.Ctx.Input.Param(":machID")
	if len(machID) == 0 {
		c.Abort("400")
	}
	m, err := master.Agent.ListMachine(machID)
	if err != nil {
		c.ServeError(500, err.Error())
	}
	var meta schema.HostMeta
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &meta); err != nil {
		c.ServeError(500, err.Error())
	}
	if len(meta.Region) == 0 || len(meta.Datacenter) == 0 {
		c.ServeError(500, "Request parameters 'Region' or 'Datacenter' is necessary")
	}
	// TODO: implement update of host regoin and host IDC infomation, if SetHostMetaInfo is not deprecated in future
	c.Data["json"] = buildHostModel(m)
	c.ServeJSON()
}

func buildHostModel(s *machine.MachineStatus) *schema.Host {
	h := &schema.Host{
		MachID:   s.MachID,
		HostName: s.MachInfo.HostName,
		HostMeta: schema.HostMeta{
			Region:     s.MachInfo.HostRegion,
			Datacenter: s.MachInfo.HostIDC,
		},
		PublicIP: s.MachInfo.PublicIP,
		IsAlive:  s.IsAlive,
		Machine: schema.Machine{
			MachID:      s.MachID,
			UsageOfCPU:  s.MachStat.UsageOfCPU,
			TotalMem:    int32(s.MachStat.TotalMem),
			UsedMem:     int32(s.MachStat.UsedMem),
			TotalSwp:    int32(s.MachStat.TotalSwp),
			UsedSwp:     int32(s.MachStat.UsedSwp),
			LoadAvg:     s.MachStat.LoadAvg,
			UsageOfDisk: transformDiskUsage(s.MachStat.UsageOfDisk),
			ClockOffset: s.MachStat.ClockOffset,
		},
	}
	return h
}

func transformDiskUsage(disks []machine.DiskUsage) []schema.DiskUsage {
	res := []schema.DiskUsage{}
	for _, disk := range disks {
		res = append(res, schema.DiskUsage{
			Mount:     disk.Mount,
			TotalSize: int32(disk.TotalSize),
			UsedSize:  int32(disk.UsedSize),
		})
	}
	return res
}
