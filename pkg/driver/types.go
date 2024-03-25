package driver

import (
	"github.com/pkg/errors"
	"os"
	"path"
)

const (
	RootFSTypeKey      = "extrootfs.io/type"
	RootFSQemuImageKey = "extrootfs.io/qemu/image"
	DefaultRootFSFile  = "rootfs"
	DefaultImagesDir   = "images"
	DefualtTypeFile    = "rootfs_type"
)

type RootFSType string

const (
	RootfsTypeQemu  = "qemu"
	RootfsTypeISCSI = "iscsi"
)

type RootFS interface {
	Allocate() error
	Connect() error
	Disconnect() error
	Cleanup() error
	WriteConfig() error
}

func NewRootFS(rootfsID, rootfsType, basePath string, config map[string]string) (RootFS, error) {

	switch rootfsType {
	case RootfsTypeQemu:
		return NewQEMURootFS(rootfsID, basePath, config)
	case RootfsTypeISCSI:
		return NewISCSIRootFS(rootfsID, basePath, config)
	}

	return nil, errors.New("extrootfs type not support")
}

func LoadRootFS(rootfsID, basePath string) (RootFS, error) {

	dataPath := path.Join(basePath, rootfsID)

	b, err := os.ReadFile(path.Join(dataPath, DefualtTypeFile))
	if err != nil {
		return nil, errors.Wrap(err, "load rootfs")
	}

	switch string(b) {
	case RootfsTypeQemu:
		return LoadQEMURootFS(dataPath)
	}

	return nil, errors.New("unknow rootfs type")

}
