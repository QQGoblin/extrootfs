package driver

import (
	"encoding/json"
	"github.com/QQGoblin/extrootfs/pkg/utils/iscsi"
	"github.com/QQGoblin/extrootfs/pkg/utils/log"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

const (
	iscsiTargetKey   = "extrootfs.io/iscsi/target"
	iscsiPortalKey   = "extrootfs.io/iscsi/portal"
	iscsiLunKey      = "extrootfs.io/iscsi/lun"
	iscsiUserKey     = "extrootfs.io/iscsi/user"
	iscsiPasswordKey = "extrootfs.io/iscsi/password"
	iscsiConfig      = "iscsi-config.json"
)

type ISCSIRootFS struct {
	BaseRootFS
	Target    string      `json:"target"`
	Portals   []string    `json:"portals"`
	Lun       int         `json:"lun"`
	Username  string      `json:"username"`
	Password  string      `json:"password"`
	ISCSIDisk *iscsi.Disk `json:"iscsi_disk"`
}

var _ RootFS = &ISCSIRootFS{}

func NewISCSIRootFS(rootfsID, basePath, outputBase string, config map[string]string) (RootFS, error) {

	base, err := NewBaseRootFS(rootfsID, basePath, outputBase, config)
	if err != nil {
		return nil, errors.Wrap(err, "create base")
	}

	lun, err := strconv.Atoi(config[iscsiLunKey])
	if err != nil {
		return nil, errors.Wrap(err, "error lun")
	}
	rootfs := &ISCSIRootFS{
		BaseRootFS: *base,
		Target:     config[iscsiTargetKey],
		Portals:    []string{config[iscsiPortalKey]},
		Lun:        lun,
		Username:   config[iscsiUserKey],
		Password:   config[iscsiPasswordKey],
	}

	return rootfs, nil
}

func (irs *ISCSIRootFS) Allocate() error {
	return nil
}

func (irs *ISCSIRootFS) Connect() error {

	iscsiDisk := iscsi.New(irs.Username, irs.Password, irs.Target, irs.Portals, int32(irs.Lun))

	if err := iscsiDisk.ReopenDisk(); err != nil {
		return errors.Wrap(err, "reopen disk")
	}
	// 修改 iscsi 设备系统参数
	if err := iscsiDisk.SetKernalConfig(); err != nil {
		return errors.Wrap(err, "config device")
	}

	// 设置 preempt key，进行抢占式挂载
	if err := iscsi.PreemptLUN(iscsiDisk.DevicePath); err != nil {
		// TODO: disconnect POS target
		_ = iscsiDisk.DetachDisk()
		return status.Errorf(codes.Internal, "Preempt LUN err:%v", err)
	}

	irs.ISCSIDisk = iscsiDisk
	irs.Device = iscsiDisk.DevicePath

	return nil
}

func (irs *ISCSIRootFS) Disconnect() error {
	irs.Device = ""
	if irs.ISCSIDisk == nil {
		return nil
	}

	if err := irs.ISCSIDisk.DetachDisk(); err != nil {
		log.WarningLogMsg("Disconnect NBD failed: %v", err)
	}
	irs.ISCSIDisk = nil
	return nil
}

func (irs *ISCSIRootFS) Cleanup() error {
	//TODO implement me
	panic("implement me")
}

func (irs *ISCSIRootFS) WriteConfig() error {
	b, err := json.Marshal(irs)
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(irs.DataPath, iscsiConfig), b, 0600); err != nil {
		return err
	}

	return irs.BaseRootFS.WriteOutput()

}

func LoadISCSIRootFS(dataPath string) (*ISCSIRootFS, error) {

	rootfs := &ISCSIRootFS{}

	b, err := os.ReadFile(path.Join(dataPath, iscsiConfig))
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(b, rootfs); err != nil {
		return nil, err
	}

	return rootfs, nil
}
