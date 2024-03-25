package main

import (
	"flag"
	"github.com/QQGoblin/extrootfs/pkg/driver"
	"k8s.io/klog/v2"
)

var (
	nodeid              string
	drivername          string
	endpoint            string
	basePath            string
	outputBase          string
	skipCreateAndDelete bool
)

func init() {

	flag.StringVar(&nodeid, "nodeid", "", "node id.")
	flag.StringVar(&drivername, "drivername", driver.DefaultDriverName, "external rootfs driver name.")
	flag.StringVar(&endpoint, "endpoint", "unix://run/extrootfs.sock", "default endpoint.")
	flag.StringVar(&basePath, "base", "/opt/extrootfs", "default endpoint.")
	flag.StringVar(&outputBase, "output", "/opt/extrootfs/output", "output for message.")
	flag.BoolVar(&skipCreateAndDelete, "skip-create-and-delete", false, "skip create and delete rootfs")
	klog.InitFlags(nil)

	if err := flag.Set("logtostderr", "true"); err != nil {
		klog.Exitf("failed to set logtostderr flag: %v", err)
	}

	flag.Parse()
}

func main() {

	driver := driver.NewDriver(drivername, nodeid, endpoint, basePath, outputBase, !skipCreateAndDelete)
	driver.Run()
}
