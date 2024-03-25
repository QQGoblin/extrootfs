package driver

import (
	"encoding/json"
	"github.com/pkg/errors"
	"os"
	"path"
)

const (
	RootFSTypeKey = "extrootfs.io/type"

	DefaultRootFSFile = "rootfs"
	DefaultImagesDir  = "images"
	DefaultTypeFile   = "rootfs_type"
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

func NewRootFS(rootfsID, rootfsType, basePath, outputbase string, config map[string]string) (RootFS, error) {

	switch rootfsType {
	case RootfsTypeQemu:
		return NewQEMURootFS(rootfsID, basePath, outputbase, config)
	case RootfsTypeISCSI:
		return NewISCSIRootFS(rootfsID, basePath, outputbase, config)
	}

	return nil, errors.New("extrootfs type not support")
}

func LoadRootFS(rootfsID, basePath string) (RootFS, error) {

	dataPath := path.Join(basePath, rootfsID)

	b, err := os.ReadFile(path.Join(dataPath, DefaultTypeFile))
	if err != nil {
		return nil, errors.Wrap(err, "load rootfs")
	}

	switch string(b) {
	case RootfsTypeQemu:
		return LoadQEMURootFS(dataPath)
	case RootfsTypeISCSI:
		return LoadISCSIRootFS(dataPath)
	}

	return nil, errors.New("unknow rootfs type")

}

type BaseRootFS struct {
	ID             string `json:"id"`
	PVCName        string `json:"pvc_name"`
	Output         string `json:"output"`
	DataPath       string `json:"data_path"`
	RootFSType     string `json:"rootfs_type"`
	Device         string `json:"device"`
	FileSystemType string `json:"file_system_type"`
}

func NewBaseRootFS(rootfsID, basePath, outputBase string, config map[string]string) (*BaseRootFS, error) {

	rootfs := &BaseRootFS{
		ID:         rootfsID,
		PVCName:    config["csi.storage.k8s.io/pvc/name"],
		Output:     path.Join(outputBase, config["csi.storage.k8s.io/pvc/name"]),
		DataPath:   path.Join(basePath, rootfsID),
		RootFSType: config[RootFSTypeKey],
	}

	if err := os.MkdirAll(rootfs.DataPath, 0755); err != nil {
		return nil, errors.Wrap(err, "new rootfs")
	}

	if err := os.WriteFile(path.Join(rootfs.DataPath, DefaultTypeFile), []byte(rootfs.RootFSType), 0600); err != nil {
		return nil, errors.Wrap(err, "new rootfs")
	}

	return rootfs, nil
}

type RootFSOutput struct {
	Device         string `json:"device"`
	FilesystemType string `json:"fs_type"`
}

func (rs *BaseRootFS) WriteOutput() error {

	if err := os.MkdirAll(path.Base(rs.Output), 0755); err != nil {
		return err
	}

	o := RootFSOutput{
		Device:         rs.Device,
		FilesystemType: rs.FileSystemType,
	}

	b, err := json.Marshal(o)
	if err != nil {
		return err
	}

	return os.WriteFile(rs.Output, b, 0644)
}
