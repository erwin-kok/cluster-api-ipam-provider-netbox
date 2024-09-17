package controller

import (
	"context"

	ipamv1alpha1 "github.com/erwin-kok/cluster-api-ipam-provider-netbox/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/pkg/netbox"
)

const (
	UrlKey      = "url"
	ApiTokenKey = "apiToken"
)

func getSecretForPool(ctx context.Context, cl client.Reader, pool *ipamv1alpha1.NetboxIPPool) (*corev1.Secret, error) {
	credRef := pool.Spec.CredentialsRef
	if credRef == nil {
		return nil, errors.New("pool does not has a CredentialsRef")
	}

	namespace := credRef.Namespace
	if len(namespace) == 0 {
		namespace = pool.GetNamespace()
	}

	secret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Namespace: namespace,
		Name:      credRef.Name,
	}
	err := cl.Get(ctx, secretKey, secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func getNetboxClient(secret *corev1.Secret, netboxServiceFactory func(url, apiToken string) (netbox.Client, error)) (netbox.Client, error) {
	url := getData(secret, UrlKey)
	if url != "" {
		return nil, errors.New("can not connect to Netbox, secret must contain url")
	}
	apiToken := getData(secret, ApiTokenKey)
	if url != "" {
		return nil, errors.New("can not connect to Netbox, secret must contain apiToken")
	}
	if netboxServiceFactory == nil {
		return nil, errors.New("must provide a Netbox service factory")
	}
	return netboxServiceFactory(url, apiToken)
}

func getData(secret *corev1.Secret, key string) string {
	if secret.Data == nil {
		return ""
	}
	if val, ok := secret.Data[key]; ok {
		return string(val)
	}
	return ""
}
