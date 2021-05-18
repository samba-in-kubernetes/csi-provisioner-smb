package provisioner

import (
	"errors"

	"github.com/golang/glog"
	smbapi "github.com/samba-in-kubernetes/samba-operator/api/v1alpha1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/samba-in-kubernetes/csi-provisioner-smb/internal/util"
)

type smbProvisioner struct {
	name     string
	version  string
	endpoint string

	ids *identityServer
	cs  *controllerServer
}

var (
	errNoDriverName     = errors.New("no driver name provided")
	errNoNodeID         = errors.New("no node id provided")
	errNoDriverEndpoint = errors.New("no driver endpoint provided")

	// FIXME: check where the version comes from
	vendorVersion = "v1alpha1"
)

func NewSmbProvisionerDriver(driverName, endpoint string, version string) (*smbProvisioner, error) {
	if driverName == "" {
		return nil, errNoDriverName
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
		name:     driverName,
		version:  vendorVersion,
		endpoint: endpoint,
	}, nil
}

func (sp *smbProvisioner) Run() {
	// Create GRPC servers
	sp.ids = NewIdentityServer(sp.name, sp.version)
	sp.cs = NewControllerServer()

	s := NewNonBlockingGRPCServer()
	s.Start(sp.endpoint, sp.ids, sp.cs)
	s.Wait()
}

func createPVC(capacity int64, storageClass *string) *v1.PersistentVolumeClaimSpec {
	mode := v1.PersistentVolumeFilesystem

	return &v1.PersistentVolumeClaimSpec{
		AccessModes: []v1.PersistentVolumeAccessMode{
			v1.ReadWriteOnce,
		},
		Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): *resource.NewQuantity(capacity, resource.BinarySI),
			},
		},
		StorageClassName: storageClass,
		VolumeMode: &mode,
	}
}

func createSmbShare(name string, pvcSpec *v1.PersistentVolumeClaimSpec) (*smbapi.SmbShare, error) {
	share := &smbapi.SmbShare{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: smbapi.SmbShareSpec{
			ShareName: name,
			Storage: smbapi.SmbShareStorageSpec{
				Pvc: &smbapi.SmbSharePvcSpec{
					Spec: pvcSpec,
				},
			},
			// TODO: add securitycontext
			// TODO: mark readOnly for ROX. Needed when VolumeContentSource is done.
		},
	}

	// TODO: get the namespace from the StorageClass?
	client := util.NewSmbOperatorClient("samba-operator-system")
	return client.CreateSmbShare(share)
}

func deleteSmbShare(volID string) error {
	// TODO: get the namespace from the StorageClass?
	client := util.NewSmbOperatorClient("samba-operator-system")
	return client.DeleteSmbShare(volID)
}
