package api

import (
	"github.com/qiuyesuifeng/tidb-demo/master"
	"github.com/qiuyesuifeng/tidb-demo/schema"
	"math/rand"
)

type MonitorController struct {
	baseController
}

func (c *MonitorController) TiDBPerformanceMetrics() {
	// TODO: implement it
	var status = master.Agent.ShowTiDBRealPerfermance()
	c.Data["json"] = schema.PerfMetrics{
		Tps:   int32(status.TPS),
		Qps:   int32(0),
		Iops:  int32(0),
		Conns: int32(status.Connections),
	}
	c.ServeJSON()
}

func (c *MonitorController) TiKVStorageMetrics() {
	// TODO: implement it
	c.Data["json"] = schema.StorageMetrics{
		Usage:    int64(randInt(535, 565)),
		Capacity: 8192,
	}
	c.ServeJSON()
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}
