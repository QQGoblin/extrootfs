package driver

import (
	"context"
	csicommon "github.com/QQGoblin/extrootfs/pkg/csi-common"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ControllerServer struct {
	*csicommon.DefaultControllerServer
	driverName string
}

func (cs *ControllerServer) CreateVolume(ctx context.Context, request *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {

	if err := cs.validateCreateVolumeRequest(request); err != nil {
		return nil, err
	}
	if request.VolumeContentSource != nil {
		return nil, status.Error(codes.InvalidArgument, "not support for create volume from snapshot or clone")
	}
	parameters := request.GetParameters()

	volume := &csi.Volume{
		VolumeId:      request.Name,
		CapacityBytes: request.GetCapacityRange().GetRequiredBytes(),
		VolumeContext: parameters,
		ContentSource: request.GetVolumeContentSource(),
	}

	return &csi.CreateVolumeResponse{Volume: volume}, nil
}

func (cs *ControllerServer) DeleteVolume(ctx context.Context, request *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	if err := cs.validateDeleteVolumeRequest(request); err != nil {
		return nil, err
	}
	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *ControllerServer) validateCreateVolumeRequest(request *csi.CreateVolumeRequest) error {
	if request.Name == "" {
		return status.Error(codes.InvalidArgument, "volume name cannot be empty")
	}
	if request.VolumeCapabilities == nil {
		return status.Error(codes.InvalidArgument, "volume capabilities cannot be empty")
	}

	return nil
}

func (cs *ControllerServer) validateDeleteVolumeRequest(request *csi.DeleteVolumeRequest) error {
	if request.GetVolumeId() == "" {
		return status.Error(codes.InvalidArgument, "empty volume ID in request")
	}
	return nil
}
