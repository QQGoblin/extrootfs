package driver

import (
	"encoding/json"
	"github.com/QQGoblin/extrootfs/pkg/util/qemu"
	"os"
	"path"
	"path/filepath"
)

const (
	RootFSTypeKey     = "extrootfs.io/type"
	RootFSImageKey    = "extrootfs.io/image"
	DefaultRootFSFile = "rootfs"
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
}

var _ RootFS = &QEMURootFS{}

type QEMURootFS struct {
	ID             string        `json:"id"`
	Image          string        `json:"image"`
	Device         string        `json:"device"`
	FileSystemType string        `json:"file_system_type"`
	DataPath       string        `json:"data_path"`
	ImagePath      string        `json:"image_path"`
	RootFSPath     string        `json:"rootfs_path"`
	BaseInfo       *qemu.ImgInfo `json:"base_info"`
}

func NewRootFS(rootfsID, rootfsType, image, dataPath string) RootFS {
	return &QEMURootFS{
		ID:         rootfsID,
		Image:      image,
		DataPath:   path.Join(dataPath, rootfsType, image),
		ImagePath:  path.Join(dataPath, rootfsType, "image", image),
		RootFSPath: path.Join(dataPath, rootfsType, image, DefaultRootFSFile),
	}
}

func (q *QEMURootFS) Allocate() error {

	var err error
	if q.BaseInfo, err = qemu.ImageInfo(q.Image); err != nil {
		return err
	}

	if err := os.MkdirAll(q.DataPath, 0755); err != nil {
		return err
	}

	if _, err = os.Stat(q.RootFSPath); os.IsNotExist(err) {
		if _, err := qemu.CreateImageFromBase(q.RootFSPath, q.ImagePath); err != nil {
			return err
		}
	}

	return err
}

func (q *QEMURootFS) Connect() error {

	nbd := qemu.NBD{}

	// TODO: lock!!
	qemu.NBDConnectLock.Lock()
	defer qemu.NBDConnectLock.Unlock()

	if err := nbd.Connect(q.RootFSPath, q.BaseInfo.Format); err != nil {
		return err
	}
	q.Device = nbd.DevicePath

	b, err := json.Marshal(nbd.DevicePath)
	if err != nil {
		nbd.Disconnect()
		return err
	}
	if err = os.WriteFile(filepath.Join(q.RootFSPath, "qemu-config.json"), b, 0600); err != nil {
		nbd.Disconnect()
	}

	return nil

}

func (q *QEMURootFS) Disconnect() error {
	//TODO implement me
	panic("implement me")
}

func (q *QEMURootFS) Cleanup() error {
	//TODO implement me
	panic("implement me")
}
