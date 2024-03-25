package driver

import (
	"encoding/json"
	"github.com/QQGoblin/extrootfs/pkg/utils/log"
	"github.com/QQGoblin/extrootfs/pkg/utils/qemu"
	"github.com/pkg/errors"
	"os"
	"path"
	"path/filepath"
)

type QEMURootFS struct {
	*BaseRootFS
	ImagePath  string        `json:"image_path"`
	RootFSPath string        `json:"rootfs_path"`
	BaseInfo   *qemu.ImgInfo `json:"base_info"`
	NBD        *qemu.NBD     `json:"nbd_info"`
}

var _ RootFS = &QEMURootFS{}

const (
	qemuConfig   = "qemu-config.json"
	qemuImageKey = "extrootfs.io/qemu/image"
)

func NewQEMURootFS(rootfsID, basePath, outputBase string, config map[string]string) (RootFS, error) {

	base, err := NewBaseRootFS(rootfsID, basePath, outputBase, config)
	if err != nil {
		return nil, errors.Wrap(err, "create base")
	}

	image := config[qemuImageKey]
	rootfs := &QEMURootFS{
		BaseRootFS: base,
		ImagePath:  path.Join(basePath, RootfsTypeQemu, DefaultImagesDir, image),
		RootFSPath: path.Join(basePath, rootfsID, DefaultRootFSFile),
	}

	return rootfs, nil
}

func (q *QEMURootFS) Allocate() error {

	var err error
	if q.BaseInfo, err = qemu.ImageInfo(q.ImagePath); err != nil {
		return err
	}

	if _, err = os.Stat(q.RootFSPath); os.IsNotExist(err) {
		_, createErr := qemu.CreateImageFromBase(q.RootFSPath, q.ImagePath)
		return createErr
	}

	return err
}

func (q *QEMURootFS) Connect() error {

	q.NBD = &qemu.NBD{}
	// TODO: lock!!
	qemu.NBDConnectLock.Lock()
	defer qemu.NBDConnectLock.Unlock()

	if err := q.NBD.Connect(q.RootFSPath, q.BaseInfo.Format); err != nil {
		return err
	}
	q.Device = q.NBD.DevicePath
	return nil

}

func (q *QEMURootFS) WriteConfig() error {
	b, err := json.Marshal(q)
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(q.DataPath, qemuConfig), b, 0600); err != nil {
		return err
	}

	return q.BaseRootFS.WriteOutput()

}

func (q *QEMURootFS) Disconnect() error {

	q.Device = ""
	if q.NBD == nil {
		return nil
	}

	if err := q.NBD.Disconnect(); err != nil {
		log.WarningLogMsg("Disconnect NBD failed: %v", err)
	}
	q.NBD = nil
	return nil

}

func (q *QEMURootFS) Cleanup() error {
	//TODO implement me
	panic("implement me")
}

func LoadQEMURootFS(dataPath string) (*QEMURootFS, error) {

	rootfs := &QEMURootFS{}

	b, err := os.ReadFile(path.Join(dataPath, qemuConfig))
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(b, rootfs); err != nil {
		return nil, err
	}

	return rootfs, nil
}
