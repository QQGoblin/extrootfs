package driver

import (
	"encoding/json"
	"github.com/QQGoblin/extrootfs/pkg/util/log"
	"github.com/QQGoblin/extrootfs/pkg/util/qemu"
	"os"
	"path"
	"path/filepath"
)

var _ RootFS = &QEMURootFS{}

const (
	qemuConfig = "qemu-config.json"
)

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
	return os.WriteFile(filepath.Join(q.DataPath, qemuConfig), b, 0600)
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
