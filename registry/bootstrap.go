package registry

import (
	etcd "github.com/coreos/etcd/client"
	"github.com/ngaut/log"
)

const bootstrapPrefix = "bootstrapped"

func (r *EtcdRegistry) IsBootstrapped() bool {
	key := r.prefixed(bootstrapPrefix)
	opts := &etcd.GetOptions{
		Quorum: true,
	}
	ctx, cancel := r.ctx()
	defer cancel()
	resp, err := r.kAPI.Get(ctx, key, opts)
	if err != nil {
		if isEtcdError(err, etcd.ErrorCodeKeyNotFound) {
			// not bootstrapped yet
			log.Warnf("The etcd registry not bootstrapped yet")
			return false
		}
		log.Fatal(err)
	}
	if resp.Node.Dir {
		log.Fatalf("Node[%s] is a directory in etcd, which's unexpected", key)
	}
	// already bootstrapped
	return true
}

func (r *EtcdRegistry) Bootstrap() (err error) {
	if err = r.mustCreateNode(r.prefixed(processPrefix), "", true); err != nil {
		return
	}
	if err = r.mustCreateNode(r.prefixed(machinePrefix), "", true); err != nil {
		return
	}
	if err = r.mustCreateNode(r.prefixed(jobPrefix), "", true); err != nil {
		return
	}
	if err = r.mustCreateNode(r.prefixed(maxProcessID), "10000", false); err != nil {
		return
	}
	if err = r.mustCreateNode(r.prefixed(bootstrapPrefix), "bootstrapped", false); err != nil {
		return
	}
	return
}
