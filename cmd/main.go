package main

import (
	"flag"
	"github.com/QQGoblin/extrootfs/pkg/driver"
	"k8s.io/klog/v2"
)

var (
	nodeid     string
	drivername string
	endpoint   string
	basePath   string
)

func init() {

	flag.StringVar(&nodeid, "nodeid", "", "node id.")
	flag.StringVar(&drivername, "drivername", driver.DefaultDriverName, "external rootfs driver name.")
	flag.StringVar(&endpoint, "endpoint", "unix://run/extrootfs.sock", "default endpoint.")
	flag.StringVar(&basePath, "base", "/opt/extrootfs", "default endpoint.")
	klog.InitFlags(nil)

	if err := flag.Set("logtostderr", "true"); err != nil {
		klog.Exitf("failed to set logtostderr flag: %v", err)
	}

	flag.Parse()
}

func main() {

	driver := driver.NewDriver(drivername, nodeid, endpoint, basePath)
	driver.Run()
}
