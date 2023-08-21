package version

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
	"github.com/Dynatrace/dynatrace-operator/src/registry"
	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/afero"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ImageVersionFunc can fetch image information from img
type ImageVersionFunc func(
	ctx context.Context,
	apiReader client.Reader,
	registryClient registry.ImageGetter,
	dynakube *dynatracev1beta1.DynaKube,
	imageName string,
	registryAuthPath string,
) (
	registry.ImageVersion,
	error,
)

var _ ImageVersionFunc = GetImageVersion

// GetImageVersion fetches image information for imageName
func GetImageVersion( //nolint:revive // argument-limit
	ctx context.Context,
	apiReader client.Reader,
	registryClient registry.ImageGetter,
	dynakube *dynatracev1beta1.DynaKube,
	imageName string,
	registryAuthPath string,
) (
	registry.ImageVersion,
	error,
) {
	keychain, transport, err := prepareKeychainAndTransport(ctx, apiReader, dynakube, registryAuthPath)
	if err != nil {
		return registry.ImageVersion{}, err
	}

	return registryClient.GetImageVersion(ctx, keychain, transport, imageName)
}

func PullImageInfo( //nolint:revive // argument-limit
	ctx context.Context,
	apiReader client.Reader,
	registryClient registry.ImageGetter,
	dynakube *dynatracev1beta1.DynaKube,
	imageName string,
	registryAuthPath string,
) (*v1.Image, error) {
	keychain, transport, err := prepareKeychainAndTransport(ctx, apiReader, dynakube, registryAuthPath)
	if err != nil {
		return nil, err
	}

	imageInfo, err := registryClient.PullImageInfo(keychain, transport, imageName)
	if err != nil {
		return nil, err
	}

	return imageInfo, nil
}

func prepareKeychainAndTransport(ctx context.Context, apiReader client.Reader, dynakube *dynatracev1beta1.DynaKube, registryAuthPath string) (authn.Keychain, *http.Transport, error) {
	var err error
	var proxy string

	keychain := dockerkeychain.NewDockerKeychain(registryAuthPath, afero.NewOsFs())
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if dynakube.HasProxy() {
		proxy, err = dynakube.Proxy(ctx, apiReader)
		if err != nil {
			return nil, nil, err
		}
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			log.Info("invalid proxy spec", "proxy", proxy)
			return nil, nil, err
		}

		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			return proxyUrl, nil
		}
	}

	if dynakube.Spec.TrustedCAs != "" {
		transport, err = addCertificates(transport, dynakube, apiReader)
		if err != nil {
			return nil, nil, fmt.Errorf("addCertificates(): %w", err)
		}
	}
	return keychain, transport, nil
}

func addCertificates(transport *http.Transport, dynakube *dynatracev1beta1.DynaKube, apiReader client.Reader) (*http.Transport, error) {
	trustedCAs, err := dynakube.TrustedCAs(context.TODO(), apiReader)
	if err != nil {
		return transport, err
	}

	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(trustedCAs); !ok {
		log.Info("failed to append custom certs!")
	}
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{} //nolint:gosec
	}
	transport.TLSClientConfig.RootCAs = rootCAs

	return transport, nil
}
