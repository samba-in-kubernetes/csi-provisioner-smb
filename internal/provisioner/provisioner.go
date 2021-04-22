package provisioner

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/kubernetes/pkg/volume/util/volumepathhandler"

	smbapi "github.com/samba-in-kubernetes/samba-operator/api/v1alpha1"
)

type smbProvisioner struct {
	name              string
	nodeID            string
	version           string
	endpoint          string

	ids *identityServer
	cs  *controllerServer
}

var (
	errNoDriverName = errors.New("no driver name provided")
	errNoNodeID = errors.New("no node id provided")
	errNoDriverEndpoint = errors.New("no driver endpoint provided")
)

func NewSmbProvisionerDriver(driverName, nodeID, endpoint string, version string) (*smbProvisioner, error) {
	if driverName == "" {
		return nil, errNoDriverName
	}

	if nodeID == "" {
		return nil, errNoNodeID
	}

	if endpoint == "" {
		return nil, errNoDriverEndpoint
	}
	if version != "" {
		vendorVersion = version
	}

	glog.Infof("Driver: %v ", driverName)
	glog.Infof("Version: %s", vendorVersion)

	return &smbProvisioner{
		name:              driverName,
		version:           vendorVersion,
		nodeID:            nodeID,
		endpoint:          endpoint,
	}, nil
}

func (sp *smbProvisioner) Run() {
	// Create GRPC servers
	sp.ids = NewIdentityServer(sp.name, sp.version)
	sp.cs = NewControllerServer(sp.nodeID)

	s := NewNonBlockingGRPCServer()
	s.Start(sp.endpoint, sp.ids, sp.cs)
	s.Wait()
}

func getVolumeByID(volumeID string) (hostPathVolume, error) {
	if hostPathVol, ok := hostPathVolumes[volumeID]; ok {
		return hostPathVol, nil
	}
	return hostPathVolume{}, fmt.Errorf("volume id %s does not exist in the volumes list", volumeID)
}

func getVolumeByName(volName string) (hostPathVolume, error) {
	for _, hostPathVol := range hostPathVolumes {
		if hostPathVol.VolName == volName {
			return hostPathVol, nil
		}
	}
	return hostPathVolume{}, fmt.Errorf("volume name %s does not exist in the volumes list", volName)
}

// getVolumePath returns the canonical path for hostpath volume
func getVolumePath(volID string) string {
	return filepath.Join(dataRoot, volID)
}

// createVolume create the directory for the hostpath volume.
// It returns the volume path or err if one occurs.
func createHostpathVolume(volID, name string, cap int64, volAccessType accessType, ephemeral bool) (*hostPathVolume, error) {
	path := getVolumePath(volID)

	switch volAccessType {
	case mountAccess:
		err := os.MkdirAll(path, 0777)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported access type %v", volAccessType)
	}

	hostpathVol := hostPathVolume{
		VolID:         volID,
		VolName:       name,
		VolSize:       cap,
		VolPath:       path,
		VolAccessType: volAccessType,
		Ephemeral:     ephemeral,
	}
	hostPathVolumes[volID] = hostpathVol
	return &hostpathVol, nil
}

// updateVolume updates the existing hostpath volume.
func updateHostpathVolume(volID string, volume hostPathVolume) error {
	glog.V(4).Infof("updating hostpath volume: %s", volID)

	if _, err := getVolumeByID(volID); err != nil {
		return err
	}

	hostPathVolumes[volID] = volume
	return nil
}

// deleteVolume deletes the directory for the hostpath volume.
func deleteHostpathVolume(volID string) error {
	glog.V(4).Infof("deleting hostpath volume: %s", volID)

	vol, err := getVolumeByID(volID)
	if err != nil {
		// Return OK if the volume is not found.
		return nil
	}

	if vol.VolAccessType == blockAccess {
		volPathHandler := volumepathhandler.VolumePathHandler{}
		// Get the associated loop device.
		device, err := volPathHandler.GetLoopDevice(getVolumePath(volID))
		if err != nil {
			return fmt.Errorf("failed to get the loop device: %v", err)
		}

		if device != "" {
			// Remove any associated loop device.
			glog.V(4).Infof("deleting loop device %s", device)
			if err := volPathHandler.RemoveLoopDevice(device); err != nil {
				return fmt.Errorf("failed to remove loop device %v: %v", device, err)
			}
		}
	}

	path := getVolumePath(volID)
	if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	delete(hostPathVolumes, volID)
	return nil
}
