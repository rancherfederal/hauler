package store

import (
	"context"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"

	"github.com/rancherfederal/hauler/cmd/hauler/cli/download"
	"github.com/rancherfederal/hauler/pkg/log"
	"github.com/rancherfederal/hauler/pkg/store"
)

type ExtractOpts struct {
	DestinationDir string
}

func (o *ExtractOpts) AddArgs(cmd *cobra.Command) {
	f := cmd.Flags()

	f.StringVar(&o.DestinationDir, "dir", "", "Directory to save contents to (defaults to current directory)")
}

func ExtractCmd(ctx context.Context, o *ExtractOpts, s *store.Store, reference string) error {
	l := log.FromContext(ctx)
	l.Debugf("running command `hauler store extract`")

	s.Open()
	defer s.Close()

	ref, err := name.ParseReference(reference)
	if err != nil {
		return err
	}

	eref := s.RelocateReference(ref)

	gopts := &download.Opts{
		DestinationDir: o.DestinationDir,
	}

	return download.Cmd(ctx, gopts, eref.Name())
}