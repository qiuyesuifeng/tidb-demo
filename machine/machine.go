package machine

import (
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/ngaut/log"
	"github.com/qiuyesuifeng/tidb-demo/pkg/utils"
)

const (
	shortIDLen    = 8
	machineDir    = ".machine"
	machineIDFile = "machineID"
)

type Machine interface {
	ID() string
	ShortID() string
	MatchID(ID string) bool
	Status() *MachineStatus
	Monitor(<-chan struct{})
}

type machine struct {
	machID     string
	hostName   string
	hostRegion string
	hostIDC    string
	publicIP   string
	stat       *MachineStat
	rwMutex    sync.RWMutex
}

func NewMachine(hostip, hostname, hostregion, hostidc string) (Machine, error) {
	machID, err := readLocalMachineID()
	if err != nil {
		log.Errorf("Read local machine ID error, %v", err)
		return nil, err
	}

	var publicIP string
	if len(hostip) > 0 {
		publicIP = hostip
	} else {
		if ipaddrs, err := utils.IntranetIP(); err != nil {
			return nil, err
		} else {
			log.Debugf("Get local IP addr: %v", ipaddrs)
			if len(ipaddrs) > 0 {
				publicIP = ipaddrs[0]
			}
		}
	}
	var hostName string
	if len(hostname) > 0 {
		hostName = hostname
	} else {
		hostName = publicIP
	}

	mach := &machine{
		machID:     machID,
		hostName:   hostName,
		hostRegion: hostregion,
		hostIDC:    hostidc,
		publicIP:   publicIP,
		stat: &MachineStat{
			LoadAvg:     []float64{},
			UsageOfDisk: []DiskUsage{},
		},
	}
	return mach, nil
}

// IsLocalMachineID returns whether the given machine ID is equal to that of the local machine
func IsLocalMachineID(mID string) bool {
	m, err := readLocalMachineID()
	return err == nil && m == mID
}

func readLocalMachineID() (string, error) {
	fullPath := filepath.Join(utils.GetDataDir(), machineDir, machineIDFile)
	if _, err := utils.CheckFileExist(fullPath); err != nil {
		return generateLocalMachineID()
	} else {
		// read the machine ID from file
		hash, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return "", err
		}
		machID := fmt.Sprintf("%X", hash)
		if len(machID) == 0 {
			return generateLocalMachineID()
		}
		return machID, nil
	}
}

// generate a new machine ID, and save it to file
func generateLocalMachineID() (string, error) {
	rand64 := string(utils.KRand(64, utils.KC_RAND_KIND_ALL))
	log.Debugf("Generated a randomized string with 64 runes, %s", rand64)
	t := sha1.New()
	io.WriteString(t, rand64)
	hash := t.Sum(nil)

	dir := filepath.Join(utils.GetDataDir(), machineDir)
	if _, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		// dir not exists, make it
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			return "", err
		}
	}

	file := filepath.Join(dir, machineIDFile)
	if err := ioutil.WriteFile(file, hash, os.ModePerm); err != nil {
		return "", err
	}
	machID := fmt.Sprintf("%X", hash)
	return machID, nil
}

func (m *machine) ID() string {
	return m.machID
}

func (m *machine) ShortID() string {
	if len(m.machID) <= shortIDLen {
		return m.machID
	}
	return m.machID[0:shortIDLen]
}

func (m *machine) MatchID(ID string) bool {
	return m.machID == ID || m.ShortID() == ID
}

func (m *machine) Status() *MachineStatus {
	return &MachineStatus{
		MachID:  m.machID,
		IsAlive: true,
		MachInfo: MachineInfo{
			HostName:   m.hostName,
			HostRegion: m.hostRegion,
			HostIDC:    m.hostIDC,
			PublicIP:   m.publicIP,
		},
		MachStat: m.getMachineStat(),
	}
}

func (m *machine) getMachineStat() MachineStat {
	m.rwMutex.RLock()
	defer m.rwMutex.RUnlock()
	return *m.stat
}
