package driver

import (
	"github.com/QQGoblin/extrootfs/pkg/util/qemu"
	"github.com/pkg/errors"
	"os"
	"path"
)

const (
	RootFSTypeKey     = "extrootfs.io/type"
	RootFSImageKey    = "extrootfs.io/image"
	DefaultRootFSFile = "rootfs"
	DefaultImagesDir  = "images"
	DefualtTypeFile   = "rootfs_type"
)

type RootFSType string

const (
	RootfsTypeQCOW2 = "qcow2"
)

type RootFS interface {
	Allocate() error
	Connect() error
	Disconnect() error
	Cleanup() error
	WriteConfig() error
}

type QEMURootFS struct {
	ID             string        `json:"id"`
	Image          string        `json:"image"`
	Device         string        `json:"device"`
	FileSystemType string        `json:"file_system_type"`
	DataPath       string        `json:"data_path"`
	ImagePath      string        `json:"image_path"`
	RootFSPath     string        `json:"rootfs_path"`
	BaseInfo       *qemu.ImgInfo `json:"base_info"`
	NBD            *qemu.NBD     `json:"nbd_info"`
}

func NewRootFS(rootfsID, rootfsType, image, basePath string) (RootFS, error) {

	rootfs := &QEMURootFS{
		ID:         rootfsID,
		Image:      image,
		DataPath:   path.Join(basePath, rootfsID),
		ImagePath:  path.Join(basePath, rootfsType, DefaultImagesDir, image),
		RootFSPath: path.Join(basePath, rootfsID, DefaultRootFSFile),
	}

	if err := os.MkdirAll(rootfs.DataPath, 0755); err != nil {
		return nil, errors.Wrap(err, "new rootfs")
	}

	if err := os.WriteFile(path.Join(rootfs.DataPath, DefualtTypeFile), []byte(rootfsType), 0600); err != nil {
		return nil, errors.Wrap(err, "new rootfs")
	}

	return rootfs, nil
}

func LoadRootFS(rootfsID, basePath string) (RootFS, error) {

	dataPath := path.Join(basePath, rootfsID)

	b, err := os.ReadFile(path.Join(dataPath, DefualtTypeFile))
	if err != nil {
		return nil, errors.Wrap(err, "load rootfs")
	}

	switch string(b) {
	case RootfsTypeQCOW2:
		return LoadQEMURootFS(dataPath)
	}

	return nil, errors.New("unknow rootfs type")

}
