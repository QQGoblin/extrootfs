package qemu

import (
	"fmt"
	"github.com/QQGoblin/extrootfs/pkg/utils/log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// NBD collects details about the used NBD device.
type NBD struct {
	BlockPath  string `json:"block_path"`
	DevicePath string `json:"device_path"`
	Name       string `json:"name"`
	PID        string `json:"pid"`
	PIDFile    string `json:"pid_file"`
}

var (
	NBDConnectLock sync.Mutex
)

// Connect a given image.
func (n *NBD) Connect(image string, format string) error {
	if err := n.allocate(); err != nil {
		return err
	}

	var cmd *exec.Cmd = exec.Command(
		"qemu-nbd",
		fmt.Sprintf("--format=%s", format),
		"--connect", n.DevicePath,
		image,
	)

	log.DefaultLog("Connect NBD: %s", cmd.String())

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "nbd.Connect")
	}

	if err := n.waitForPID(); err != nil {
		return errors.Wrap(err, "nbd.Connect")
	}

	if err := exec.Command("blockdev", "--rereadpt", n.DevicePath).Run(); err != nil {
		return errors.Wrap(err, "nbd.Connect")
	}

	//log.DebugLogMsg("udevadm settle")
	//if err := exec.Command("udevadm", "settle").Run(); err != nil {
	//	return errors.Wrap(err, "nbd.Connect")
	//}

	return nil
}

// Disconnect the NBD device from qemu-nbd to free it.
func (n *NBD) Disconnect() error {

	if n.DevicePath == "" {
		return nil
	}

	log.DebugLogMsg("Disconnect NBD from %s", n.Name)

	if err := exec.Command("qemu-nbd", "--disconnect", n.DevicePath).Run(); err != nil {
		return errors.Wrap(err, "nbd.Disconnect")
	}

	if err := n.waitForPIDCleanup(); err != nil {
		return errors.Wrap(err, "nbd.Disconnect")
	}

	//if err := exec.Command("udevadm", "settle").Run(); err != nil {
	//	return errors.Wrap(err, "nbd.Disconnect")
	//}

	return nil
}

// Allocate checks for nbd kernel module and finds an empty NBD device to use.
func (n *NBD) allocate() error {

	//if !n.isNBDLoaded() {
	//	if err := n.loadNBD(); err != nil {
	//		return errors.Wrap(err, "nbd.allocate")
	//	}
	//}

	files, err := filepath.Glob("/sys/block/nbd*")
	if err != nil {
		return errors.Wrap(err, "nbd.allocate")
	}

	for _, file := range files {
		if _, err := os.Stat(path.Join(file, "pid")); os.IsNotExist(err) {
			n.Name = filepath.Base(file)
			n.DevicePath = path.Join("/dev", filepath.Base(file))
			n.BlockPath = file
			n.PIDFile = filepath.Join(file, "pid")
			return nil
		}
	}

	return errors.New("Unable to allocate an NBD device")
}

// isNBDLoaded verifies if the nbd kernel module is loaded.
func (n *NBD) isNBDLoaded() bool {
	out, err := exec.Command("lsmod").Output()
	if err != nil {
		log.ErrorLogMsg("lsmod failed: %v", err)
	}

	if !strings.Contains(string(out), "nbd") {
		return false
	}

	return true
}

// loadNBD loads the NBD kernel module if required.
func (n *NBD) loadNBD() error {

	if err := exec.Command("modprobe", "nbd").Run(); err != nil {
		return errors.Wrap(err, "nbd.loadNBD")
	}

	if err := exec.Command("udevadm", "settle").Run(); err != nil {
		return errors.Wrap(err, "nbd.loadNBD")
	}

	return nil
}

// waitForPID waits on pidfile for nbd device.
func (n *NBD) waitForPID() error {
	for i := 0; i < 30; i++ {
		data, err := os.ReadFile(n.PIDFile)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		n.PID = strings.TrimSpace(string(data))
		return nil
	}

	return errors.New("timed out waiting for nbd file")
}

// waitForPIDCleanup need to wait for the PID file to disappear.
func (n *NBD) waitForPIDCleanup() error {
	for i := 0; i < 30; i++ {
		_, err := os.ReadFile(n.PIDFile)
		if err != nil {
			return nil
		}

		time.Sleep(time.Second)
		continue
	}

	return errors.New("timed out waiting for nbd file to cleanup")
}
