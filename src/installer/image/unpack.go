package image

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/Dynatrace/dynatrace-operator/src/dockerkeychain"
	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	containerv1 "github.com/google/go-containerregistry/pkg/v1"
	typesv1 "github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/pkg/errors"
)

const (

	//TODO: REMOVE USE CONSTANTS FROM github.com/google/go-containerregistry/pkg/v1/types
	// MediaTypeImageLayerGzip is the media type used for gzipped layers
	// referenced by the manifest.
	mediaTypeImageLayerGzip = "application/vnd.oci.image.layer.v1.tar+gzip"

	//TODO: REMOVE USE CONSTANTS FROM github.com/google/go-containerregistry/pkg/v1/types
	// MediaTypeImageLayerZstd is the media type used for zstd compressed
	// layers referenced by the manifest.
	mediaTypeImageLayerZstd = "application/vnd.oci.image.layer.v1.tar+zstd"

)

type imagePullInfo struct {
	imageCacheDir string
	targetDir     string
}

func (installer Installer) extractAgentBinariesFromImage(pullInfo imagePullInfo, registryAuthPath string, imageName string) error { //nolint
	img, err := installer.pullImageInfo(registryAuthPath, imageName)
	if err != nil {
		log.Info("pullImageInfo", "error", err)
		return err
	}

	image := *img

	err = installer.pullOCIimage(image, imageName, pullInfo.imageCacheDir, pullInfo.targetDir)
	if err != nil {
		log.Info("pullOCIimage", "err", err)
		return err
	}

	return nil
}

func (installer Installer) pullImageInfo(registryAuthPath string, imageName string) (*containerv1.Image, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, errors.WithMessagef(err, "parsing reference %q:", imageName)
	}

	keyChain := dockerkeychain.NewDockerKeychain(registryAuthPath, installer.fs)

	image, err := remote.Image(ref, remote.WithContext(context.TODO()), remote.WithAuthFromKeychain(keyChain), remote.WithTransport(installer.transport))
	if err != nil {
		return nil, errors.WithMessagef(err, "getting image %q", imageName)
	}
	return &image, nil
}

func (installer Installer) pullOCIimage(image containerv1.Image, imageName string, imageCacheDir string, targetDir string) error {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return errors.WithMessagef(err, "parsing reference %q", imageName)
	}

	log.Info("pullOciImage", "ref_identifier", ref.Identifier(), "ref.Name", ref.Name(), "ref.String", ref.String())

	err = installer.fs.MkdirAll(imageCacheDir, common.MkDirFileMode)
	if err != nil {
		log.Info("failed to create cache dir", "dir", imageCacheDir, "err", err)
		return errors.WithStack(err)
	}

	if err := crane.SaveOCI(image, path.Join(imageCacheDir, ref.Identifier())); err != nil {
		log.Info("saving v1.Image img as an OCI Image Layout at path", imageCacheDir, err)
		return errors.WithMessagef(err, "saving v1.Image img as an OCI Image Layout at path %s", imageCacheDir)
	}
	layers, _ := image.Layers()

	err = installer.unpackOciImage(layers, filepath.Join(imageCacheDir, ref.Identifier()), targetDir)
	if err != nil {
		log.Info("failed to unpackOciImage", "error", err)
		return errors.WithStack(err)
	}
	return nil
}

func (installer Installer) unpackOciImage(layers []containerv1.Layer, imageCacheDir string, targetDir string) error {
	for _, layer := range layers {
		mediaType, _ := layer.MediaType()
		switch mediaType{
		case typesv1.DockerLayer:
			digest, _ := layer.Digest()
			sourcePath := filepath.Join(imageCacheDir, "blobs", digest.Algorithm, digest.Hex)
			log.Info("unpackOciImage", "sourcePath", sourcePath)
			if err := installer.extractor.ExtractGzip(sourcePath, targetDir); err != nil {
				return err
			}
		//TODO: REMOVE USE CONSTANTS FROM github.com/google/go-containerregistry/pkg/v1/types
		case mediaTypeImageLayerGzip:
			return errors.New("MediaTypeImageLayerGzip is not implemented")
		//TODO: REMOVE USE CONSTANTS FROM github.com/google/go-containerregistry/pkg/v1/types
		case mediaTypeImageLayerZstd:
			return errors.New("MediaTypeImageLayerZstd is not implemented")
		default:
			return fmt.Errorf("unknown media type: %s", mediaType)
		}
	}
	log.Info("unpackOciImage", "targetDir", targetDir)
	return nil
}
