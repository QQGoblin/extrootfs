package driver

import (
	"github.com/pkg/errors"
	"os"
	"path"
)

const ()

type ISCSIRootFS struct {
	ID             string   `json:"id"`
	Image          string   `json:"image"`
	Device         string   `json:"device"`
	FileSystemType string   `json:"file_system_type"`
	DataPath       string   `json:"data_path"`
	Target         string   `json:"target"`
	Portals        []string `json:"portals"`
	Username       string   `json:"username"`
	Password       string   `json:"password"`
}

var _ RootFS = &ISCSIRootFS{}

func NewISCSIRootFS(rootfsID, basePath string, config map[string]string) (RootFS, error) {

	rootfs := &ISCSIRootFS{
		ID:       rootfsID,
		DataPath: path.Join(basePath, rootfsID),
	}

	if err := os.MkdirAll(rootfs.DataPath, 0755); err != nil {
		return nil, errors.Wrap(err, "new rootfs")
	}

	if err := os.WriteFile(path.Join(rootfs.DataPath, DefualtTypeFile), []byte(RootfsTypeISCSI), 0600); err != nil {
		return nil, errors.Wrap(err, "new rootfs")
	}

	return rootfs, nil
}

func (I ISCSIRootFS) Allocate() error {
	//TODO implement me
	panic("implement me")
}

func (I ISCSIRootFS) Connect() error {
	//TODO implement me
	panic("implement me")
}

func (I ISCSIRootFS) Disconnect() error {
	//TODO implement me
	panic("implement me")
}

func (I ISCSIRootFS) Cleanup() error {
	//TODO implement me
	panic("implement me")
}

func (I ISCSIRootFS) WriteConfig() error {
	//TODO implement me
	panic("implement me")
}
