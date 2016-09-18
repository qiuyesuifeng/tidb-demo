package registry

import (
	etcd "github.com/coreos/etcd/client"
	"path"
	"strings"
	"time"
	"github.com/qiuyesuifeng/tidb-demo/minion"
	"golang.org/x/net/context"
	"github.com/ngaut/log"
	"github.com/qiuyesuifeng/tidb-demo/pkg/utils"
)

const (
	jobPrefix = "job"

	// Occurs when any Process's target state is touched
	ProcessTargetStateChangeEvent = utils.Event("ProcessTargetStateChangeEvent")
	// Occurs when any Machine's state is touched
	MachineStateChangeEvent = utils.Event("MachineStateChangeEvent")
)

type EtcdEventStream struct {
	watcher    etcd.Watcher
	rootPrefix string
}

func NewEtcdEventStream(kapi etcd.KeysAPI, keyPrefix string) minion.EventStream {
	key := path.Join(keyPrefix, jobPrefix)
	opts := &etcd.WatcherOptions{
		AfterIndex: 0,
		Recursive:  true,
	}
	return &EtcdEventStream{
		watcher:    kapi.Watcher(key, opts),
		rootPrefix: keyPrefix,
	}
}

// Next returns a channel which will emit an Event as soon as one of interest occurs
func (es *EtcdEventStream) Next(timeout time.Duration) chan utils.Event {
	evchan := make(chan utils.Event)
	go func() {
		ctx, _ := context.WithTimeout(context.Background(), timeout)
		res, err := es.watcher.Next(ctx)
		if err != nil {
			if err == context.DeadlineExceeded {
				close(evchan)
			} else {
				log.Errorf("Some failure encountered while waiting for next etcd event, %v", err)
			}
		} else {
			if ev, ok := parse(res, es.rootPrefix); ok {
				evchan <- ev
			}
		}
		return
	}()
	return evchan
}

func parse(res *etcd.Response, prefix string) (ev utils.Event, ok bool) {
	if res == nil || res.Node == nil {
		return
	}
	if !strings.HasPrefix(res.Node.Key, path.Join(prefix, jobPrefix)) {
		return
	}
	switch path.Base(res.Node.Key) {
	case "process-state":
		ev = ProcessTargetStateChangeEvent
		ok = true
	case "machine-state":
		ev = MachineStateChangeEvent
		ok = true
	}
	return
}
