package driver

import (
	csicommon "github.com/QQGoblin/extrootfs/pkg/csi-common"
	"github.com/QQGoblin/extrootfs/pkg/utils/lock"
	"github.com/QQGoblin/extrootfs/pkg/utils/log"
	"github.com/container-storage-interface/spec/lib/go/csi"
)

const (
	DefaultDriverName    = "driver.extrootfs.io"
	defaultDriverVersion = "1"
	topologyKeyNode      = "topology.extrootfs.io/node"
)

type Driver struct {
	csiDriver              *csicommon.CSIDriver
	servers                *csicommon.Servers
	name                   string
	nodeid                 string
	endpoint               string
	basePath               string
	outputBase             string
	ctrlCapCreateAndDelete bool
}

// NewDriver returns new ceph driver.
func NewDriver(name, nodeid, endpoint, basePath, outputBase string, ctrlCapCreateAndDelete bool) *Driver {
	return &Driver{
		csiDriver:              csicommon.NewCSIDriver(name, nodeid, endpoint),
		servers:                &csicommon.Servers{},
		name:                   name,
		nodeid:                 nodeid,
		endpoint:               endpoint,
		basePath:               basePath,
		outputBase:             outputBase,
		ctrlCapCreateAndDelete: ctrlCapCreateAndDelete,
	}
}

func (r *Driver) NewServers() {

	var ctrlCap []csi.ControllerServiceCapability_RPC_Type
	if r.ctrlCapCreateAndDelete {
		ctrlCap = append(ctrlCap, csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME)
	}

	r.csiDriver.AddControllerServiceCapabilities(ctrlCap)

	r.csiDriver.AddNodeServiceCapabilities([]csi.NodeServiceCapability_RPC_Type{})

	r.servers.IS = csicommon.NewDefaultIdentityServer(r.csiDriver)

	r.servers.CS = &ControllerServer{
		DefaultControllerServer: csicommon.NewDefaultControllerServer(r.csiDriver),
		driverName:              r.name,
	}

	r.servers.NS = &NodeServer{
		DefaultNodeServer: csicommon.NewDefaultNodeServer(r.csiDriver, map[string]string{topologyKeyNode: r.nodeid}),
		driverName:        r.name,
		basePath:          r.basePath,
		outputBase:        r.outputBase,
		rootfsLock:        lock.NewVolumeLocks(),
	}

}

func (r *Driver) Run() {

	r.csiDriver = csicommon.NewCSIDriver(r.name, defaultDriverVersion, r.nodeid)
	if r.csiDriver == nil {
		log.FatalLogMsg("Failed to initialize CSI Driver.")
	}
	r.NewServers()
	s := csicommon.NewNonBlockingGRPCServer()
	s.Start(r.endpoint, *r.servers)
	s.Wait()
}
