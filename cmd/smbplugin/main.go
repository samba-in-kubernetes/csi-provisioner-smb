package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	sp "github.com/nixpanic/csi-provisioner-smb/internal/provisioner"
)

func init() {
	flag.Set("logtostderr", "true")
}

var (
	endpoint          = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	driverName        = flag.String("drivername", "smb.csi.k8s.io", "name of the driver")
	showVersion       = flag.Bool("version", false, "Show version.")
	// Set by the build process
	version = ""
)

func main() {
	flag.Parse()

	if *showVersion {
		baseName := path.Base(os.Args[0])
		fmt.Println(baseName, version)
		return
	}

	handle()
	os.Exit(0)
}

func handle() {
	driver, err := sp.NewSmbProvisionerDriver(*driverName, *nodeID, *endpoint, version)
	if err != nil {
		fmt.Printf("Failed to initialize driver: %s", err.Error())
		os.Exit(1)
	}
	driver.Run()
}
