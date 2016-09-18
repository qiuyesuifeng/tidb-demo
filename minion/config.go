package minion

import (
	"errors"
	"flag"
	"path"

	"github.com/ngaut/log"
	"github.com/qiuyesuifeng/tidb-demo/pkg/utils"
	"github.com/rakyll/globalconf"
)

const (
	// TTL to use with all state pushed to Registry
	DefaultTTL = "10s"
	// If an environment variable with the EnvPrefix is set, it will take precedence over values
	// in the configuration file. Command line flags will override the environment variables.
	EnvConfigPrefix = "TIDEMO_"
	// First try to load configuration file in $(PWD), if not exist then check /etc/tiadm/tiadm.conf
	DefaultConfigFile = "tidemo.conf"
	DefaultConfigDir  = "/etc/tidemo"
	DefaultKeyPrefix  = "/_pingcap.com/tidemo"
)

type Config struct {
	EtcdServers        []string
	EtcdKeyPrefix      string
	EtcdRequestTimeout int
	MonitorInterval    int
	HostIP             string
	HostName           string
	HostRegion         string
	HostIDC            string
	AgentTTL           string
}

func ParseFlag() (*Config, error) {
	etcdServers := flag.String("etcd", "http://127.0.0.1:2379,http://127.0.0.1:4001", "List of etcd endpoints, default 'http://127.0.0.1:2379'")
	etcdKeyPrefix := flag.String("etcd-prefix", DefaultKeyPrefix, "Namespace for tidemo registry in etcd")
	etcdRequestTimeout := flag.Int("etcd-timeout", 2500, "Amount of time in milliseconds to allow a single etcd request before considering it failed.")
	monitorInterval := flag.Int("interval", 2000, "Interval at which the monitor should check and report the cluster status in etcd periodically.")
	hostIP := flag.String("ip", "", "IP address which this host advertises")
	hostName := flag.String("name", "", "The identifier of this machine in cluster")
	hostRegion := flag.String("region", "", "Geographical region where this machine located")
	hostIDC := flag.String("idc", "", "The IDC which this machine placed physically")
	agentTTL := flag.String("ttl", DefaultTTL, "TTL in seconds of machine state in etcd")
	logLevel := flag.String("log-level", "debug", "Log level: info, debug, warn, error, fatal")
	dataDir := flag.String("data-dir", "", "The path of data directory in which program's logs and storage data will be placed")

	opts := globalconf.Options{EnvPrefix: EnvConfigPrefix}
	if file, err := pathToConfigFile(); err == nil {
		opts.Filename = file
	}
	if gconf, err := globalconf.NewWithOptions(&opts); err == nil {
		gconf.ParseAll()
	} else {
		return nil, err
	}

	log.SetLevelByString(*logLevel)
	utils.SetDataDir(*dataDir)

	cfg := &Config{
		EtcdServers:        utils.NewStringSlice(*etcdServers),
		EtcdKeyPrefix:      *etcdKeyPrefix,
		EtcdRequestTimeout: *etcdRequestTimeout,
		MonitorInterval:    *monitorInterval,
		HostIP:             *hostIP,
		HostName:           *hostName,
		HostRegion:         *hostRegion,
		HostIDC:            *hostIDC,
		AgentTTL:           *agentTTL,
	}
	return cfg, nil
}

func pathToConfigFile() (string, error) {
	cd := utils.GetCmdDir()
	rd := utils.GetRootDir()

	if path, err := utils.CheckFileExist(path.Join(cd, DefaultConfigFile)); err == nil {
		return path, nil
	}
	if path, err := utils.CheckFileExist(path.Join(rd, DefaultConfigFile)); err == nil {
		return path, nil
	}
	if path, err := utils.CheckFileExist(path.Join(rd, "conf", DefaultConfigFile)); err == nil {
		return path, nil
	}
	if path, err := utils.CheckFileExist(path.Join(DefaultConfigDir, DefaultConfigFile)); err == nil {
		return path, nil
	}
	return "", errors.New("Not configuration file found")
}
