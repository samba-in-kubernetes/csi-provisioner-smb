package util

import (
	"context"

	smbapi "github.com/samba-in-kubernetes/samba-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

const (
	smbShareType = "smbshares"
)

type SmbOperatorClient struct {
	client *rest.RESTClient
	ns     string
}

// NewSmbOperatorClient returns an in-cluster Kubernetes client that can be
// used to create, list, update and delete objects in the cluster.
func NewSmbOperatorClient(namespace string) *SmbOperatorClient {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	smbapi.AddToScheme(scheme.Scheme)

	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &smbapi.GroupVersion
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	c, err := rest.UnversionedRESTClientFor(&crdConfig)
	if err != nil {
		panic(err)
	}

	return &SmbOperatorClient{
		client: c,
		ns:     namespace,
	}
}

// GetSmbShare returns the SmbShare with the given name.
func (soc *SmbOperatorClient) GetSmbShare(name string) (*smbapi.SmbShare, error) {
	share := &smbapi.SmbShare{}

	err := soc.client.
		Get().
		Namespace(soc.ns).
		Resource(smbShareType).
		VersionedParams(&metav1.GetOptions{}, scheme.ParameterCodec).
		Do(context.TODO()).
		Into(share)
	if err != nil {
		return nil, err
	}

	return share, nil
}

// CreateSmbShare creates a new SmbShare. This function does not wait for
// successful completion (starting of the SmbShare Pod), use GetSmbShare to
// verify the status of the SmbShare.
func (soc *SmbOperatorClient) CreateSmbShare(share *smbapi.SmbShare) (*smbapi.SmbShare, error) {
	newShare := &smbapi.SmbShare{}

	err := soc.client.Post().
		Resource(smbShareType).
		Namespace(soc.ns).
		Name(share.Spec.ShareName).
		Body(share).
		Do(context.TODO()).
		Into(newShare)
	if err != nil {
		return nil, err
	}

	return newShare, nil
}

// DeleteSmbShare removes the SmbShare with the given name.
func (soc *SmbOperatorClient) DeleteSmbShare(name string) error {
	_, err := soc.client.Delete().
		Resource(smbShareType).
		Name(name).
		Namespace(soc.ns).
		DoRaw(context.TODO())
	return err
}
