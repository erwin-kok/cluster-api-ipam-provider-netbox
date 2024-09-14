package controller

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/erwin-kok/cluster-api-ipam-provider-netbox/internal/netbox"
)

const (
	UrlKey      = "url"
	ApiTokenKey = "apiToken"
)

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
