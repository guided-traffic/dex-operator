/*
Copyright 2025 Guided Traffic GmbH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package builder

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dexv1 "github.com/guided-traffic/dex-operator/api/v1"
)

// kindStaticClient is the kind reported for static clients.
const kindStaticClient = "DexStaticClient"

// localConnectorID is the connector ID under which Dex serves its password
// database while enablePasswordDB is set.
const localConnectorID = "local"

// ErrNoConnector is wrapped by the error [Build] returns when connectors were
// collected but none of them renders and no DexLocalConnector enables the
// password database.  Dex does not start without a connector, so the render
// is aborted and the last written config stays.  The [Output] returned with
// it carries Rejected and DroppedTrustedPeers, but no config.
var ErrNoConnector = errors.New("no connector renders")

// unitResult is what building one child produced.
type unitResult struct {
	connector ConnectorEntry
	client    StaticClient
	mounts    []MountedSecret
}

// renderUnit is one connector or static client on its way into the config:
// the ID it claims, how to build its entry and what became of it.
type renderUnit struct {
	kind      string
	namespace string
	name      string
	created   metav1.Time

	// id is the claimed ID; empty when the child claims none.
	id string
	// resolved reports whether id comes from the spec or the Secret, as
	// opposed to the client's status.clientID fallback.
	resolved bool
	// claimErr is why the ID could not be resolved (static clients only).
	claimErr error
	build    func(ctx context.Context, env *childEnv) (unitResult, error)

	// wins reports that the contest rule leaves the ID to this child.
	wins bool
	// reserved reports that the ID is reserved by Dex itself.
	reserved bool
	rendered bool
	buildErr error
	result   unitResult
}

// claim is one child's claim on an ID, as far as the contest rule sees it.
type claim struct {
	namespace string
	id        string
}

// contest applies the contest rule to the claims of one ID space and reports,
// per claim, whether the rule leaves the ID to it:
//
//  1. one claimant: it wins;
//  2. several claimants, exactly one of them in home (the installation's
//     namespace): that one wins;
//  3. otherwise nobody wins.
//
// An empty ID is no claim and never wins.  The outcome depends on the claims
// alone, not on their order.
func contest(claims []claim, home string) []bool {
	byID := make(map[string][]int, len(claims))
	for i, c := range claims {
		if c.id != "" {
			byID[c.id] = append(byID[c.id], i)
		}
	}

	wins := make([]bool, len(claims))
	for _, idx := range byID {
		if w := contestWinner(claims, idx, home); w >= 0 {
			wins[w] = true
		}
	}
	return wins
}

// contestWinner returns the index of the claim that wins among the claims
// at idx, or -1 when nobody does.
func contestWinner(claims []claim, idx []int, home string) int {
	if len(idx) == 1 {
		return idx[0]
	}
	winner := -1
	for _, i := range idx {
		if claims[i].namespace != home {
			continue
		}
		if winner >= 0 {
			return -1
		}
		winner = i
	}
	return winner
}

// decide runs the contest over units, one ID space, and marks the winners.
// Reserved units take no part.
func decide(units []*renderUnit, home string) {
	claims := make([]claim, len(units))
	for i, u := range units {
		if !u.reserved {
			claims[i] = claim{namespace: u.namespace, id: u.id}
		}
	}
	for i, wins := range contest(claims, home) {
		units[i].wins = wins
	}
}

// byEnvPriority orders units for env var key assignment: the installation's
// namespace first, then the oldest, then by kind, namespace and name.  The
// render itself keeps its own order; this order only decides who keeps a
// plain key when two keys collide.
func byEnvPriority(home string) func(a, b *renderUnit) int {
	return func(a, b *renderUnit) int {
		if aHome, bHome := a.namespace == home, b.namespace == home; aHome != bHome {
			if aHome {
				return -1
			}
			return 1
		}
		if c := a.created.Compare(b.created.Time); c != 0 {
			return c
		}
		return cmp.Or(
			cmp.Compare(a.kind, b.kind),
			cmp.Compare(a.namespace, b.namespace),
			cmp.Compare(a.name, b.name),
		)
	}
}

// buildUnits builds every unit in env priority order.  A winner builds into
// the render's env and renders when its build succeeds; every other unit
// builds dry, only to learn whether its own build fails.
func buildUnits(ctx context.Context, units []*renderUnit, home string, envs map[string][]byte) {
	ordered := slices.Clone(units)
	slices.SortStableFunc(ordered, byEnvPriority(home))
	for _, u := range ordered {
		u.buildOne(ctx, envs)
	}
}

func (u *renderUnit) buildOne(ctx context.Context, envs map[string][]byte) {
	if u.claimErr != nil {
		u.buildErr = u.claimErr
		return
	}
	if !u.wins {
		_, u.buildErr = u.build(ctx, newChildEnv(nil, u.kind, u.namespace, u.name))
		return
	}
	env := newChildEnv(envs, u.kind, u.namespace, u.name)
	res, err := u.build(ctx, env)
	if err != nil {
		u.buildErr = err
		return
	}
	env.commit()
	u.result = res
	u.rendered = true
}

// rejection reports why u is not part of the rendered config, or nil when
// it is.  A child's own build error takes precedence over a lost contest,
// because it is what the child's owner can act on.  The message never names
// another namespace.
func (u *renderUnit) rejection(inst *dexv1.DexInstallation) *dexv1.RejectedChild {
	if u.rendered {
		return nil
	}
	rc := &dexv1.RejectedChild{Kind: u.kind, Namespace: u.namespace, Name: u.name, ID: u.id}
	if u.buildErr != nil {
		rc.Reason = dexv1.RejectionReasonBuildFailed
		rc.Message = u.buildErr.Error()
		return rc
	}
	rc.Reason = dexv1.RejectionReasonDuplicateID
	rc.Message = u.duplicateMessage(inst)
	return rc
}

func (u *renderUnit) duplicateMessage(inst *dexv1.DexInstallation) string {
	switch {
	case u.reserved:
		return fmt.Sprintf("connector ID %q is reserved for the password database of DexInstallation %s/%s",
			u.id, inst.Namespace, inst.Name)
	case u.kind == kindStaticClient:
		return fmt.Sprintf("client ID %q is claimed by more than one DexStaticClient of DexInstallation %s/%s",
			u.id, inst.Namespace, inst.Name)
	default:
		return fmt.Sprintf("connector ID %q is claimed by more than one connector of DexInstallation %s/%s",
			u.id, inst.Namespace, inst.Name)
	}
}

// rejectedChildren returns the rejection of every unit that does not render,
// sorted by kind, namespace and name.
func rejectedChildren(inst *dexv1.DexInstallation, unitLists ...[]*renderUnit) []dexv1.RejectedChild {
	var out []dexv1.RejectedChild
	for _, units := range unitLists {
		for _, u := range units {
			if rc := u.rejection(inst); rc != nil {
				out = append(out, *rc)
			}
		}
	}
	slices.SortFunc(out, func(a, b dexv1.RejectedChild) int {
		return cmp.Or(
			cmp.Compare(a.Kind, b.Kind),
			cmp.Compare(a.Namespace, b.Namespace),
			cmp.Compare(a.Name, b.Name),
		)
	})
	return out
}

// noConnectorError returns the connector guard's error: connectors were
// collected, none renders and no DexLocalConnector renders.  It returns nil
// whenever the render may go ahead.
func noConnectorError(connectors []*renderUnit, local bool, rejected []dexv1.RejectedChild) error {
	if local || len(connectors) == 0 {
		return nil
	}
	for _, u := range connectors {
		if u.rendered {
			return nil
		}
	}

	parts := make([]string, 0, len(connectors))
	for _, rc := range rejected {
		if rc.Kind == kindStaticClient {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s/%s: %s: %s", rc.Kind, rc.Namespace, rc.Name, rc.Reason, rc.Message))
	}
	return fmt.Errorf("%w: none of the %d collected connectors renders and Dex does not start without one, "+
		"the last written config stays: %s", ErrNoConnector, len(connectors), strings.Join(parts, "; "))
}

// countRendered returns how many of units render.
func countRendered(units []*renderUnit) int {
	n := 0
	for _, u := range units {
		if u.rendered {
			n++
		}
	}
	return n
}
