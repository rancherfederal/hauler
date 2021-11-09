package store

import (
	"context"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/rancherfederal/hauler/pkg/artifact"
	"github.com/rancherfederal/hauler/pkg/layout"
	"github.com/rancherfederal/hauler/pkg/log"
)

// Add will add an artifact.OCI to the store
//  The method to achieve this is to save artifact.OCI to a temporary directory in an OCI layout compatible form.  Once
//  saved, the entirety of the layout is copied to the store (which is just a registry).  This allows us to not only use
//  strict types to define generic content, but provides a processing pipeline suitable for extensibility.  In the
//  future we'll allow users to define their own content that must adhere either by artifact.OCI or simply an OCI layout.
func (s *Store) Add(ctx context.Context, oci artifact.OCI, locationRef name.Reference) (ocispec.Descriptor, error) {
	lgr := log.FromContext(ctx)

	if err := s.precheck(); err != nil {
		return ocispec.Descriptor{}, err
	}

	relocated, err := RelocateReference(locationRef, s.RegistryURL())
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	lgr.Debugf("adding %s to store", relocated.Name())

	tmpdir, err := os.MkdirTemp("", "hauler")
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	defer os.RemoveAll(tmpdir)

	l, err := layout.FromPath(tmpdir)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	mdesc, err := l.WriteOci(oci, relocated.Name())
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	if err := s.AddFromLayout(ctx, tmpdir); err != nil {
		return ocispec.Descriptor{}, err
	}

	return mdesc, err
}

// AddFromLayout will read an oci-layout and add all manifests referenced in index.json to the store
func (s *Store) AddFromLayout(ctx context.Context, layoutPath string) error {
	if err := s.precheck(); err != nil {
		return err
	}

	ociStore, err := layout.NewOCIStore(layoutPath)
	if err != nil {
		return err
	}

	return layout.Copy(ctx, ociStore, s.RegistryURL())
}
