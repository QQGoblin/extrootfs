package csicommon

import (
	"context"

	"github.com/QQGoblin/extrootfs/pkg/utils/log"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DefaultControllerServer points to default driver.
type DefaultControllerServer struct {
	csi.UnimplementedControllerServer
	Driver *CSIDriver
}

// ControllerGetCapabilities implements the default GRPC callout.
// Default supports all capabilities.
func (cs *DefaultControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	log.TraceLog(ctx, "Using default ControllerGetCapabilities")
	if cs.Driver == nil {
		return nil, status.Error(codes.Unimplemented, "Controller server is not enabled")
	}

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.Driver.capabilities,
	}, nil
}
