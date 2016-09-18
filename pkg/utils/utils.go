package utils

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/libcontainer/netlink"
	"github.com/ngaut/log"
)

const (
	KC_RAND_KIND_NUM   = 0
	KC_RAND_KIND_LOWER = 1
	KC_RAND_KIND_UPPER = 2
	KC_RAND_KIND_ALL   = 3
)

var (
	cmddir  string
	rootdir string
	datadir string
)

func init() {
	SetCmdDir()
	SetRootDir()
}

func SetCmdDir() {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	cmddir = filepath.Dir(path)

}

func SetRootDir() {
	path, _ := os.Getwd()
	if filepath.Base(path) == "bin" {
		rootdir = filepath.Dir(path)
	} else {
		rootdir = path
	}
}

func SetDataDir(d string) {
	if d == "" {
		datadir = rootdir
		return
	}
	datadir = d
}

func GetCmdDir() string {
	return cmddir
}

func GetRootDir() string {
	return rootdir
}

func GetDataDir() string {
	return datadir
}

func CheckFileExist(filepath string) (string, error) {
	fi, err := os.Stat(filepath)
	if err != nil {
		return "", err
	}
	if fi.IsDir() {
		return "", errors.New(fmt.Sprintf("filepath: %s, is a directory, not a file", filepath))
	}
	return filepath, nil
}

func KRand(size int, kind int) []byte {
	ikind, kinds, result := kind, [][]int{[]int{10, 48}, []int{26, 97}, []int{26, 65}}, make([]byte, size)
	is_all := kind > 2 || kind < 0
	for i := 0; i < size; i++ {
		if is_all { // random ikind
			ikind = rand.Intn(3)
		}
		scope, base := kinds[ikind][0], kinds[ikind][1]
		result[i] = uint8(base + rand.Intn(scope))
	}
	return result
}

type StringSlice []string

func NewStringSlice(value string) StringSlice {
	var s = []string{}
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimLeft(item, " [\"")
		item = strings.TrimRight(item, " \"]")
		s = append(s, item)
	}
	return StringSlice(s)
}

func (f StringSlice) String() string {
	return strings.Join(f.Value(), ",")
}

func (f StringSlice) Value() []string {
	return []string(f)
}

type Protocol string
type Port int32

const (
	ProtocolHttp = Protocol("http")
	ProtocolUnix = Protocol("unix")
)

func (p Port) Value() int32 {
	return int32(p)
}

func (p Protocol) String() string {
	return string(p)
}

type Endpoint struct {
	Protocol Protocol
	IPAddr   string
	Port     Port
}

func (e Endpoint) String() string {
	var ip = "0.0.0.0"
	if len(e.IPAddr) > 0 {
		ip = e.IPAddr
	}
	if len(e.Protocol) > 0 {
		return fmt.Sprintf("%s://%s:%d", e.Protocol.String(), ip, e.Port.Value())
	} else {
		return fmt.Sprintf("%s:%d", ip, e.Port.Value())
	}
}

func TrimAddrs(addrs []string) []string {
	res := []string{}
	for _, addr := range addrs {
		parts := strings.Split(addr, "://")
		if len(parts) == 2 {
			res = append(res, parts[1])
		} else {
			res = append(res, addr)
		}
	}
	return res
}

func ParseEndpoint(str string) (Endpoint, error) {
	var res Endpoint
	parts := strings.Split(str, "://")
	if len(parts) == 2 {
		sparts := strings.Split(parts[1], ":")
		if len(sparts) < 2 {
			return res, errors.New(fmt.Sprintf("Illegal endpoint string: %s", str))
		}
		res.Protocol = Protocol(parts[0])
		res.IPAddr = sparts[0]
		if port, err := strconv.Atoi(sparts[1]); err != nil {
			return res, errors.New(fmt.Sprintf("Illegal endpoint string: %s", str))
		} else {
			res.Port = Port(port)
		}
	} else {
		sparts := strings.Split(parts[0], ":")
		if len(sparts) < 2 {
			return res, errors.New(fmt.Sprintf("Illegal endpoint string: %s", str))
		}
		res.IPAddr = sparts[0]
		if port, err := strconv.Atoi(sparts[1]); err != nil {
			return res, errors.New(fmt.Sprintf("Illegal endpoint string: %s", str))
		} else {
			res.Port = Port(port)
		}
	}
	return res, nil
}

func ParseEndpoints(slice []string) ([]Endpoint, error) {
	res := []Endpoint{}
	for _, s := range slice {
		ep, err := ParseEndpoint(s)
		if err != nil {
			return nil, err
		}
		res = append(res, ep)
	}
	return res, nil
}

func EndpointsToStrings(endpoints map[string]Endpoint) []string {
	res := []string{}
	for _, ep := range endpoints {
		res = append(res, ep.String())
	}
	return res
}

type Event string

// None event produced by a watcher timeout
func (e Event) None() bool {
	if string(e) == "" {
		return true
	}
	return false
}

type EventStream interface {
	Next(timeout time.Duration) chan Event
}

// Method 1 to get local IP addr
func IntranetIP() (ips []string, err error) {
	ips = make([]string, 0)
	ifaces, e := net.Interfaces()
	if e != nil {
		return ips, e
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		// ignore docker and warden bridge
		if strings.HasPrefix(iface.Name, "docker") || strings.HasPrefix(iface.Name, "w-") {
			continue
		}
		addrs, e := iface.Addrs()
		if e != nil {
			return ips, e
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			ipStr := ip.String()
			if IsIntranet(ipStr) {
				ips = append(ips, ipStr)
			}
		}
	}
	return ips, nil
}

func IsIntranet(ipStr string) bool {
	if strings.HasPrefix(ipStr, "10.") || strings.HasPrefix(ipStr, "192.168.") {
		return true
	}
	if strings.HasPrefix(ipStr, "172.") {
		// 172.16.0.0-172.31.255.255
		arr := strings.Split(ipStr, ".")
		if len(arr) != 4 {
			return false
		}
		second, err := strconv.ParseInt(arr[1], 10, 64)
		if err != nil {
			return false
		}
		if second >= 16 && second <= 31 {
			return true
		}
	}
	return false
}

// Method 2 to get local IP addr
func GetLocalIP() (got string) {
	iface := getDefaultGatewayIface()
	if iface == nil {
		return
	}
	addrs, err := iface.Addrs()
	if err != nil || len(addrs) == 0 {
		return
	}
	for _, addr := range addrs {
		// Attempt to parse the address in CIDR notation
		// and assert that it is IPv4 and global unicast
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}
		if !usableAddress(ip) {
			continue
		}
		got = ip.String()
		break
	}
	return
}

func usableAddress(ip net.IP) bool {
	return ip.To4() != nil && ip.IsGlobalUnicast()
}

func getDefaultGatewayIface() *net.Interface {
	log.Debug("Attempting to retrieve IP route info from netlink")
	routes, err := netlink.NetworkGetRoutes()
	if err != nil {
		log.Debugf("Unable to detect default interface: %v", err)
		return nil
	}
	if len(routes) == 0 {
		log.Debug("Netlink returned zero routes")
		return nil
	}
	for _, route := range routes {
		if route.Default {
			if route.Iface == nil {
				log.Debugf("Found default route but could not determine interface")
			}
			log.Debugf("Found default route with interface %v", route.Iface.Name)
			return route.Iface
		}
	}
	log.Debugf("Unable to find default route")
	return nil
}
