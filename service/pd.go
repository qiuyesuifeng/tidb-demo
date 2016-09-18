package service

import (
	"flag"
	"regexp"
	"strconv"
	"strings"
	"github.com/qiuyesuifeng/tidb-demo/minion"
	"github.com/qiuyesuifeng/tidb-demo/pkg/utils"
)

const PD_SERVICE = "PD"

type PDService struct {
	service
}

func NewPDService() Service {
	return &PDService{
		service{
			svcName:      PD_SERVICE,
			version:      "1.0.0",
			executor:     []string{},
			command:      "bin/pd-server",
			args:         []string{"--addr", "0.0.0.0:1234", "--advertise-addr", "$HOST_IP:1234", "--etcd", "$ETCD_ADDR", "--pprof", ":6060", "-L", "debug", "--cluster-id", "1", "--max-peer-count", "3"},
			environments: map[string]string{},
			endpoints: map[string]utils.Endpoint{
				"PD_ADDR": utils.Endpoint{
					Port: minion.Port(1234),
				},
				"PD_ADVERTISE_ADDR": utils.Endpoint{
					Port: minion.Port(1234),
				},
				"PD_PPROF_ADDR": utils.Endpoint{
					Port: minion.Port(6060),
				},
			},
		},
	}
}

func (s *PDService) ParseEndpointFromArgs(args []string) map[string]utils.Endpoint {
	var res = make(map[string]utils.Endpoint)
	argset := flag.NewFlagSet(PD_SERVICE, flag.ContinueOnError)
	argset.String("addr", "127.0.0.1:1234", "server listening address")
	argset.String("advertise-addr", "", "server advertise listening address [127.0.0.1:1234] for client communication")
	argset.String("etcd", "127.0.0.1:2379", "Etcd endpoints, separated by comma")
	argset.String("root", "/pd", "pd root path in etcd")
	argset.Int64("lease", 3, "Leader lease time (second)")
	argset.String("L", "debug", "log level: info, debug, warn, error, fatal")
	argset.String("pprof", ":6060", "pprof HTTP listening address")
	argset.Uint64("cluster-id", 0, "Cluster ID")
	argset.Uint("max-peer-count", 3, "max peer number for the region")
	if err := argset.Parse(args); err != nil {
		// handle error
		return s.endpoints
	}

	for k, v := range s.endpoints {
		switch k {
		case "PD_ADDR":
			if flag := argset.Lookup("addr"); flag != nil {
				addrParts := strings.Split(flag.Value.String(), ":")
				if len(addrParts) > 1 {
					if p, err := strconv.Atoi(addrParts[1]); err == nil {
						v.Port = minion.Port(p)
					}
				}
			}
		case "PD_ADVERTISE_ADDR":
			if flag := argset.Lookup("advertise-addr"); flag != nil {
				addrParts := strings.Split(flag.Value.String(), ":")
				if len(addrParts) > 1 {
					if ok, _ := regexp.MatchString(`^(\d+)\.(\d+)\.(\d+)\.(\d+)$`, addrParts[0]); ok {
						v.IPAddr = addrParts[0]
					}
					if p, err := strconv.Atoi(addrParts[1]); err == nil {
						v.Port = minion.Port(p)
					}
				}
			}
		case "PD_PPROF_ADDR":
			if flag := argset.Lookup("pprof"); flag != nil {
				addrParts := strings.Split(flag.Value.String(), ":")
				if len(addrParts) > 1 {
					if p, err := strconv.Atoi(addrParts[1]); err == nil {
						v.Port = minion.Port(p)
					}
				}
			}
		}
		res[k] = v
	}
	return res
}
