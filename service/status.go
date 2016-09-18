package service

import (
	"github.com/qiuyesuifeng/tidb-demo/pkg/utils"
)

type ServiceStatus struct {
	SvcName      string
	Version      string
	Executor     []string
	Command      string
	Args         []string
	Environments map[string]string
	Endpoints    map[string]utils.Endpoint
}

type TiDBPerfMetrics struct {
	TPS         int64  `json:"tps"`
	Connections int    `json:"connections"`
	Version     string `json:"version"`
}
