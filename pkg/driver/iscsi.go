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
)

const (
	iscsiTargetKey   = "extrootfs.io/iscsi/target"
	iscsiPortalKey   = "extrootfs.io/iscsi/portal"
	iscsiUserKey     = "extrootfs.io/iscsi/user"
	iscsiPasswordKey = "extrootfs.io/iscsi/password"
	iscsiConfig      = "iscsi-config.json"
)

type ISCSIRootFS struct {
	ID             string      `json:"id"`
	Image          string      `json:"image"`
	Device         string      `json:"device"`
	FileSystemType string      `json:"file_system_type"`
	DataPath       string      `json:"data_path"`
	Target         string      `json:"target"`
	Portals        []string    `json:"portals"`
	Username       string      `json:"username"`
	Password       string      `json:"password"`
	ISCSIDisk      *iscsi.Disk `json:"iscsi_disk"`
}

var _ RootFS = &ISCSIRootFS{}

func NewISCSIRootFS(rootfsID, basePath string, config map[string]string) (RootFS, error) {

	rootfs := &ISCSIRootFS{
		ID:       rootfsID,
		DataPath: path.Join(basePath, rootfsID),
		Target:   config[iscsiTargetKey],
		Portals:  []string{config[iscsiPortalKey]},
		Username: config[iscsiUserKey],
		Password: config[iscsiPasswordKey],
	}

	if err := os.MkdirAll(rootfs.DataPath, 0755); err != nil {
		return nil, errors.Wrap(err, "new rootfs")
	}

	if err := os.WriteFile(path.Join(rootfs.DataPath, DefualtTypeFile), []byte(RootfsTypeISCSI), 0600); err != nil {
		return nil, errors.Wrap(err, "new rootfs")
	}

	return rootfs, nil
}

func (irs *ISCSIRootFS) Allocate() error {
	return nil
}

func (irs *ISCSIRootFS) Connect() error {

	iscsiDisk := iscsi.New(irs.Username, irs.Password, irs.Target, irs.Portals)

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
	return os.WriteFile(filepath.Join(irs.DataPath, iscsiConfig), b, 0600)
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
