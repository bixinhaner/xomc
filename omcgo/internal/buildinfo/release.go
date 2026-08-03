package buildinfo

import (
	"crypto/sha256"
	"strings"

	"github.com/google/uuid"
)

var ReleaseVersion = "dev"
var GitCommit = "unknown"

func ReleaseCampaignID() (uuid.UUID, bool) {
	version := strings.TrimSpace(ReleaseVersion)
	revision := strings.TrimSpace(GitCommit)
	if version == "" || version == "dev" ||
		revision == "" || revision == "unknown" || revision == "n/a" {
		return uuid.Nil, false
	}
	// ReleaseVersion contains the packaging timestamp. The immutable commit is
	// the campaign identity: repackaging/redeploying identical code must resume
	// the same durable campaign instead of scheduling another full-device sync.
	sum := sha256.Sum256([]byte(revision))
	var id uuid.UUID
	copy(id[:], sum[:16])
	id[6] = (id[6] & 0x0f) | 0x80
	id[8] = (id[8] & 0x3f) | 0x80
	return id, true
}
