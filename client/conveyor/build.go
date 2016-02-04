package conveyor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// Build is a helper around the BuildCreate, BuildInfo, LogsStream and
// ArtifactInfo methods to ultimately return an Artifact and stream any
// build logs.
func (s *Service) Build(w io.Writer, o BuildCreateOpts) (*Artifact, error) {
	if o.Sha == nil {
		return nil, errors.New("cannot build without sha")
	}

	repoSha := fmt.Sprintf("%s@%s", o.Repository, *o.Sha)

	a, err := s.ArtifactInfo(repoSha)
	if err == nil {
		return a, nil
	}

	if !notFound(err) {
		return nil, err
	}

	b, err := s.BuildInfo(repoSha)
	if err != nil {
		if !notFound(err) {
			return nil, err
		}

		// No build, create one
		b, err = s.BuildCreate(o)
		if err != nil {
			return nil, err
		}

		io.WriteString(w, fmt.Sprintf("Build: %s\n", b.ID))
	}

	buildID := b.ID

	if err := s.LogsStream(w, buildID); err != nil {
		fmt.Fprintf(os.Stderr, "error streaming logs: %v\n", err)
	}

	for {
		<-time.After(5 * time.Second)

		b, err = s.BuildInfo(buildID)
		if err != nil {
			return nil, err
		}

		// If the build failed, return an error.
		if b.State == "failed" {
			return nil, fmt.Errorf("build %s failed", buildID)
		}

		// If the build completed, we should have an artifact.
		if b.CompletedAt != nil {
			break
		}
	}

	return s.ArtifactInfo(repoSha)
}
