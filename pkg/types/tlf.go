package types

import (
	"context"
	"fmt"

	"github.com/dotmesh-io/dotmesh/pkg/auth"
	"github.com/dotmesh-io/dotmesh/pkg/user"
)

// special admin user with global privs
const ADMIN_USER_UUID = "00000000-0000-0000-0000-000000000000"

// TODO: to implement forks, we can just construct a TopLevelFilesystem
// where MasterBranch refers to an id which isn't _actually_ a top level
// filesystem, because it (at the ZFS layer) does have an origin. This
// _should_ all "just work", apart from the fact that the name
// TopLevelFilesystem becomes even more confusing. Rename it to 'Dot'?

type TopLevelFilesystem struct {
	MasterBranch  DotmeshVolume
	OtherBranches []DotmeshVolume
	Owner         user.SafeUser
	Collaborators []user.SafeUser
}

func (t TopLevelFilesystem) AuthorizeOwner(ctx context.Context) (bool, error) {
	return t.authorize(ctx, false)
}

func (t TopLevelFilesystem) Authorize(ctx context.Context) (bool, error) {
	return t.authorize(ctx, true)
}

// TODO: this is where we'll add support for read-only collaborators
// when we do https://github.com/dotmesh-io/dotmesh/issues/575

func (t TopLevelFilesystem) authorize(ctx context.Context, includeCollab bool) (bool, error) {
	user := auth.GetUserFromCtx(ctx)
	if user == nil {
		return false, fmt.Errorf("No user found in context.")
	}
	// admin user is always authorized (e.g. docker daemon). users and auth are
	// only really meaningful over the network for data synchronization, when a
	// dotmesh cluster is being used like a hub.
	if user.Id == ADMIN_USER_UUID {
		return true, nil
	}
	if user.Id == t.Owner.Id {
		return true, nil
	}
	if includeCollab {
		for _, other := range t.Collaborators {
			if user.Id == other.Id {
				return true, nil
			}
		}
	}
	return false, nil
}
