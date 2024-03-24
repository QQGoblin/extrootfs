package qemu

import (
	"encoding/json"
	"github.com/pkg/errors"
	"os"
	"os/exec"
)

type ImgInfo struct {
	// Filename analyzed
	Filename string `json:"filename"`

	// Type of image (e.g. qcow2, raw, etc.)
	Format string `json:"format"`

	// Physical size of the disk (e.g. 550637568)
	ActualSize int64 `json:"actual-size"`

	// Virtual size of the disk (e.g. 2361393152)
	VirtualSize int64 `json:"virtual-size"`
}

func info(name string) (*ImgInfo, error) {

	if _, err := os.Stat(name); err != nil {
		return nil, err
	}

	cmd := exec.Command(
		"qemu-img",
		"info",
		"--output=json",
		name,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "image.Info")
	}

	info := &ImgInfo{}
	if err = json.Unmarshal(out, info); err != nil {
		return nil, errors.Wrap(err, "image.Info")
	}

	return info, nil

}

func ImageInfo(name string) (*ImgInfo, error) {
	return info(name)
}

func CreateImageFromBase(name, base string) (*ImgInfo, error) {

	baseInfo, err := info(base)
	if err != nil {
		return nil, errors.Wrap(err, "image.Create")
	}

	//qemu-img create -f qcow2 rootfs.qcow2 -b $PWD/centos-7.4.1708.qcow2 -F qcow2
	cmd := exec.Command("qemu-img", "create",
		"-f", baseInfo.Format, name,
		"-b", base, "-F", baseInfo.Format,
	)

	if err := cmd.Run(); err != nil {
		return nil, errors.Wrap(err, "image.Create")
	}

	return info(base)
}
