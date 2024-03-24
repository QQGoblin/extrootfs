package driver

import (
	"context"
	csicommon "github.com/QQGoblin/extrootfs/pkg/csi-common"
	"github.com/QQGoblin/extrootfs/pkg/utils/lock"
	"github.com/QQGoblin/extrootfs/pkg/utils/log"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NodeServer struct {
	*csicommon.DefaultNodeServer
	driverName string
	basePath   string
	rootfsLock *lock.VolumeLocks
}

func (ns *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {

	if err := ns.validateNodePublishVolumeRequest(req); err != nil {
		return nil, err
	}

	// 由于 UnpublishVolume 只会传 PV 名称，因此这里使用 PV 的名称作为 rootfs 的 ID
	rootfsID := req.VolumeId

	if acquired := ns.rootfsLock.TryAcquire(rootfsID); !acquired {
		return nil, status.Errorf(codes.Aborted, "an operation with the given Volume ID %s already exists", rootfsID)
	}
	defer ns.rootfsLock.Release(rootfsID)

	rootfs, err := NewRootFS(rootfsID, req.VolumeContext[RootFSTypeKey], ns.basePath, req.VolumeContext)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "New RootFS %s failed: %v", rootfsID, err)
	}

	if err := rootfs.Allocate(); err != nil {
		return nil, status.Errorf(codes.Internal, "Allocate RootFS %s failed: %v", rootfsID, err)
	}

	//TODO: 判断 rootfs 是否在使用？

	if err := rootfs.Connect(); err != nil {
		rootfs.Disconnect()
		return nil, status.Errorf(codes.Internal, "Connect RootFS %s failed: %v", rootfsID, err)
	}

	if err := rootfs.WriteConfig(); err != nil {
		rootfs.Disconnect()
		return nil, status.Errorf(codes.Internal, "Write RootFS %s config failed: %v", rootfsID, err)
	}

	log.DebugLog(ctx, "NodePublishVolume RootFS success: %v", rootfs)

	return &csi.NodePublishVolumeResponse{}, nil
}
func (ns *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {

	if err := ns.validateNodeUnpublishVolumeRequest(req); err != nil {
		return nil, err
	}

	// 由于 UnpublishVolume 只会传 PV 名称，因此这里使用 PV 的名称作为 rootfs 的 ID
	rootfsID := req.VolumeId

	if acquired := ns.rootfsLock.TryAcquire(rootfsID); !acquired {
		return nil, status.Errorf(codes.Aborted, "an operation with the given Volume ID %s already exists", rootfsID)
	}
	defer ns.rootfsLock.Release(rootfsID)

	rootfs, err := LoadRootFS(rootfsID, ns.basePath)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Load RootFS %s failed: %v", rootfsID, err)
	}

	if err := rootfs.Disconnect(); err != nil {
		return nil, status.Errorf(codes.Internal, "Write RootFS %s config failed: %v", rootfsID, err)
	}

	if err := rootfs.WriteConfig(); err != nil {
		return nil, status.Errorf(codes.Internal, "Write RootFS %s config failed: %v", rootfsID, err)
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *NodeServer) validateNodePublishVolumeRequest(request *csi.NodePublishVolumeRequest) error {

	if request.GetVolumeCapability() == nil {
		return status.Errorf(codes.InvalidArgument, "volume capability cannot be empty")
	}

	if request.GetVolumeId() == "" {
		return status.Errorf(codes.InvalidArgument, "volume ID cannot be empty")
	}

	if request.GetTargetPath() == "" {
		return status.Errorf(codes.InvalidArgument, "target path cannot be empty")
	}

	//if request.GetStagingTargetPath() == "" {
	//	return status.Errorf(codes.FailedPrecondition, "staging target path cannot be empty")
	//}
	return ns.validateFromVolContext(request.VolumeContext)
}

func (ns *NodeServer) validateNodeUnpublishVolumeRequest(request *csi.NodeUnpublishVolumeRequest) error {

	if request.GetVolumeId() == "" {
		return status.Errorf(codes.InvalidArgument, "volume ID cannot be empty")
	}

	if request.GetTargetPath() == "" {
		return status.Errorf(codes.InvalidArgument, "target path cannot be empty")
	}

	return nil
}

func (ns *NodeServer) validateFromVolContext(volContext map[string]string) error {

	//rootfsImage := volContext[RootFSImageKey]
	//if rootfsImage == "" {
	//	return status.Errorf(codes.InvalidArgument, "rootfs image is empty")
	//}

	rootfsType := volContext[RootFSTypeKey]
	if rootfsType != RootfsTypeQCOW2 {
		return status.Errorf(codes.InvalidArgument, "error rootfs type %s", rootfsType)
	}

	return nil
}
