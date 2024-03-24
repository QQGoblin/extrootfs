package csicommon

import (
	"github.com/QQGoblin/extrootfs/pkg/utils/log"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

// CSIDriver stores driver information.
type CSIDriver struct {
	name                   string
	nodeID                 string
	version                string
	topology               map[string]string
	capabilities           []*csi.ControllerServiceCapability
	controllerCapabilities []*csi.ControllerServiceCapability
	nodeCapabilities       []*csi.NodeServiceCapability
	vc                     []*csi.VolumeCapability_AccessMode
}

// NewCSIDriver Creates a NewCSIDriver object. Assumes vendor
// version is equal to driver version &  does not support optional
// driver plugin info manifest field. Refer to CSI spec for more details.
func NewCSIDriver(name, v, nodeID string) *CSIDriver {
	if name == "" {
		klog.Errorf("Driver name missing")

		return nil
	}

	if nodeID == "" {
		klog.Errorf("NodeID missing")

		return nil
	}
	// TODO version format and validation
	if v == "" {
		klog.Errorf("Version argument missing")

		return nil
	}

	driver := CSIDriver{
		name:    name,
		version: v,
		nodeID:  nodeID,
	}

	return &driver
}

// ValidateControllerServiceRequest validates the controller
// plugin capabilities.
//
//nolint:interfacer // c can be of type fmt.Stringer, but that does not make the API clearer
func (d *CSIDriver) ValidateControllerServiceRequest(c csi.ControllerServiceCapability_RPC_Type) error {
	if c == csi.ControllerServiceCapability_RPC_UNKNOWN {
		return nil
	}

	for _, capability := range d.capabilities {
		if c == capability.GetRpc().GetType() {
			return nil
		}
	}

	return status.Error(codes.InvalidArgument, c.String())
}

// AddControllerServiceCapabilities stores the controller capabilities
// in driver object.
func (d *CSIDriver) AddControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) {
	csc := make([]*csi.ControllerServiceCapability, 0, len(cl))

	for _, c := range cl {
		log.DefaultLog("Enabling controller service capability: %v", c.String())
		csc = append(csc, NewControllerServiceCapability(c))
	}

	d.capabilities = csc
}

func (d *CSIDriver) AddNodeServiceCapabilities(cl []csi.NodeServiceCapability_RPC_Type) {
	for _, c := range cl {
		log.DefaultLog("Enabling controller service capability: %v", c.String())
		d.nodeCapabilities = append(d.nodeCapabilities, NewNodeServiceCapability(c))
	}
}

// AddVolumeCapabilityAccessModes stores volume access modes.
func (d *CSIDriver) AddVolumeCapabilityAccessModes(vc []csi.VolumeCapability_AccessMode_Mode) []*csi.VolumeCapability_AccessMode {
	vca := make([]*csi.VolumeCapability_AccessMode, 0, len(vc))
	for _, c := range vc {
		log.DefaultLog("Enabling volume access mode: %v", c.String())
		vca = append(vca, NewVolumeCapabilityAccessMode(c))
	}
	d.vc = vca

	return vca
}

// GetVolumeCapabilityAccessModes returns access modes.
func (d *CSIDriver) GetVolumeCapabilityAccessModes() []*csi.VolumeCapability_AccessMode {
	return d.vc
}
