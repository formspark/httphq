# Dependencies

How advisories are gated on both sides, and what moves together when a
toolchain or a dependency changes.

## Advisories

`check:audit` is `pnpm audit --prod --audit-level high`, over the Node
periphery. The Go side has govulncheck, which gates on reachability rather
than presence, and is the stricter of the two: it is why `golang.org/x/net` and
`golang.org/x/crypto` carry advisories at module level while verify stays
green. The npm scan has no such filter, so it is scoped to what ships and to
what is worth stopping for. An advisory that has no fix, or one that cannot be
reached from this code, goes in `audit.ignore` (a list of GHSA ids) in
`pnpm-workspace.yaml`, with the reason in a comment beside it. Nothing is ignored
today.

The pipeline only answers whether a change introduces an advisory. `audit.yml`
asks the other question once a week, against whatever is on master, because a
repository nobody pushes to runs nothing. The nightly `test.yml` already
covers the Go side the same way.

## Pins move together

`nodejs` and `pnpm` in `.tool-versions`, `packageManager` and `engines` in
package.json, the `go` line in go.mod (the Dockerfile's `GOTOOLCHAIN=auto`
fetches it), and the Go tool versions in the package.json scripts. CI reads
`.tool-versions` and go.mod, so a pin changed in one place and not the others
builds one thing locally and another in CI.

## New releases wait a day

`minimumReleaseAge` in `pnpm-workspace.yaml` holds every package back until it
has been published for a day, and its comment says why. A release urgent enough
to need sooner goes in `minimumReleaseAgeExclude` with its reason.
