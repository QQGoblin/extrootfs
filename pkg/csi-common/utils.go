package csicommon

import (
	"context"
	"fmt"
	"k8s.io/mount-utils"
	"runtime/debug"
	"strings"
	"sync/atomic"

	"github.com/QQGoblin/extrootfs/pkg/util/log"

	"github.com/container-storage-interface/spec/lib/go/csi"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

func parseEndpoint(ep string) (string, string, error) {
	if strings.HasPrefix(strings.ToLower(ep), "unix://") || strings.HasPrefix(strings.ToLower(ep), "tcp://") {
		s := strings.SplitN(ep, "://", 2)
		if s[1] != "" {
			return s[0], s[1], nil
		}
	}

	return "", "", fmt.Errorf("invalid endpoint: %v", ep)
}

// NewVolumeCapabilityAccessMode returns volume access mode.
func NewVolumeCapabilityAccessMode(mode csi.VolumeCapability_AccessMode_Mode) *csi.VolumeCapability_AccessMode {
	return &csi.VolumeCapability_AccessMode{Mode: mode}
}

// NewDefaultNodeServer initializes default node server.
func NewDefaultNodeServer(d *CSIDriver, topology map[string]string) *DefaultNodeServer {
	d.topology = topology

	return &DefaultNodeServer{
		Driver:  d,
		Mounter: mount.NewWithoutSystemd(""),
	}
}

// NewDefaultIdentityServer initializes default identity server.
func NewDefaultIdentityServer(d *CSIDriver) *DefaultIdentityServer {
	return &DefaultIdentityServer{
		Driver: d,
	}
}

// NewDefaultControllerServer initializes default controller server.
func NewDefaultControllerServer(d *CSIDriver) *DefaultControllerServer {
	return &DefaultControllerServer{
		Driver: d,
	}
}

// NewControllerServiceCapability returns controller capabilities.
func NewControllerServiceCapability(ctrlCap csi.ControllerServiceCapability_RPC_Type) *csi.ControllerServiceCapability {
	return &csi.ControllerServiceCapability{
		Type: &csi.ControllerServiceCapability_Rpc{
			Rpc: &csi.ControllerServiceCapability_RPC{
				Type: ctrlCap,
			},
		},
	}
}

// NewNodeServiceCapability returns node capabilities.
func NewNodeServiceCapability(nodeCap csi.NodeServiceCapability_RPC_Type) *csi.NodeServiceCapability {
	return &csi.NodeServiceCapability{
		Type: &csi.NodeServiceCapability_Rpc{
			Rpc: &csi.NodeServiceCapability_RPC{
				Type: nodeCap,
			},
		},
	}
}

// NewMiddlewareServerOption creates a new grpc.ServerOption that configures a
// common format for log messages and other gRPC related handlers.
func NewMiddlewareServerOption() grpc.ServerOption {
	middleWare := []grpc.UnaryServerInterceptor{contextIDInjector, logGRPC, panicHandler}

	return grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(middleWare...))
}

func getReqID(req interface{}) string {
	// if req is nil empty string will be returned
	reqID := ""
	switch r := req.(type) {
	case *csi.CreateVolumeRequest:
		reqID = r.Name

	case *csi.DeleteVolumeRequest:
		reqID = r.VolumeId

	case *csi.CreateSnapshotRequest:
		reqID = r.Name
	case *csi.DeleteSnapshotRequest:
		reqID = r.SnapshotId

	case *csi.ControllerExpandVolumeRequest:
		reqID = r.VolumeId

	case *csi.NodeStageVolumeRequest:
		reqID = r.VolumeId
	case *csi.NodeUnstageVolumeRequest:
		reqID = r.VolumeId

	case *csi.NodePublishVolumeRequest:
		reqID = r.VolumeId
	case *csi.NodeUnpublishVolumeRequest:
		reqID = r.VolumeId

	case *csi.NodeExpandVolumeRequest:
		reqID = r.VolumeId
	}

	return reqID
}

var id uint64

func contextIDInjector(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	atomic.AddUint64(&id, 1)
	ctx = context.WithValue(ctx, log.CtxKey, id)
	if reqID := getReqID(req); reqID != "" {
		ctx = context.WithValue(ctx, log.ReqID, reqID)
	}

	return handler(ctx, req)
}

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.ExtendedLog(ctx, "GRPC call: %s", info.FullMethod)
	log.TraceLog(ctx, "GRPC request: %s", protosanitizer.StripSecrets(req))

	resp, err := handler(ctx, req)
	if err != nil {
		klog.Errorf(log.Log(ctx, "GRPC error: %v"), err)
	} else {
		log.TraceLog(ctx, "GRPC response: %s", protosanitizer.StripSecrets(resp))
	}

	return resp, err
}

//nolint:nonamedreturns // named return used to send recovered panic error.
func panicHandler(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			klog.Errorf("panic occurred: %v", r)
			debug.PrintStack()
			err = status.Errorf(codes.Internal, "panic %v", r)
		}
	}()

	return handler(ctx, req)
}
