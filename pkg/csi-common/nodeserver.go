package csicommon

import (
	"context"
	"github.com/QQGoblin/extrootfs/pkg/utils/log"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/mount-utils"
)

// DefaultNodeServer stores driver object.
type DefaultNodeServer struct {
	csi.UnimplementedNodeServer
	Driver     *CSIDriver
	Mounter    mount.Interface
	NodeLabels map[string]string
}

// NodeGetInfo returns node ID.
func (ns *DefaultNodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	log.TraceLog(ctx, "Using default NodeGetInfo")

	csiTopology := &csi.Topology{
		Segments: ns.Driver.topology,
	}

	return &csi.NodeGetInfoResponse{
		NodeId:             ns.Driver.nodeID,
		AccessibleTopology: csiTopology,
	}, nil
}

// NodeGetCapabilities returns RPC unknown capability.
func (ns *DefaultNodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	log.TraceLog(ctx, "Using default NodeGetCapabilities")

	if ns.Driver == nil {
		return nil, status.Error(codes.Unimplemented, "Controller server is not enabled")
	}

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: ns.Driver.nodeCapabilities,
	}, nil
}
