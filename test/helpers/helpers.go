package helpers

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func init() {
	utilruntime.Must(ipamv1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(clusterv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(ipamv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(admissionv1.AddToScheme(scheme.Scheme))
}

// TestEnvironment encapsulates a Kubernetes local test environment.
type TestEnvironment struct {
	manager.Manager
	client.Client
	Config *rest.Config
	env    *envtest.Environment
	cancel context.CancelFunc
}

func NewTestEnvironment(crdDirectoryPaths []string) (*TestEnvironment, error) {
	env := &envtest.Environment{
		CRDDirectoryPaths:        crdDirectoryPaths,
		ErrorIfCRDPathMissing:    true,
		ControlPlaneStopTimeout:  60 * time.Second,
		AttachControlPlaneOutput: true,
		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.30.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}
	// Config is defined in this file globally.
	cfg, err := env.Start()
	if err != nil {
		return nil, err
	}

	options := ctrl.Options{
		Scheme: scheme.Scheme,
	}
	mgr, err := ctrl.NewManager(cfg, options)
	if err != nil {
		return nil, err
	}

	cl := mgr.GetClient()
	komega.SetClient(cl)

	return &TestEnvironment{
		env:     env,
		Client:  cl,
		Config:  cfg,
		Manager: mgr,
	}, nil
}

func (t *TestEnvironment) StartManager(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	return t.Manager.Start(ctx)
}

func (t *TestEnvironment) StopManager() error {
	t.cancel()
	return t.env.Stop()
}
