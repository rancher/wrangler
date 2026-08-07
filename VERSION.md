Each wrangler major version supports a range of Kubernetes minor versions. The range supported by each major release line is include below. Wrangler follows the following rules for changes between major/minor/patch:

<ins>Major Version Increases</ins>:
- Support for a kubernetes version is explicitly removed (note that this means that wrangler uses a feature that does not work on this version).
- A breaking change is made, which is not necessary to resolve a defect.

<ins>Minor Version Increases</ins>:
- Support for a kubernetes version is added.
- A breaking change in an exported function is made to resolve a defect.

<ins>Patch Version Increases</ins>
- A bug was fixed.
- A feature was added, in a backwards-compatible way.
- A breaking change in an exported function is made to resolve a CVE.

<ins>Dealing with Kubernetes 1.35</ins>
Clients working with versions of Kubernetes before 1.35 might not work with the `main`
branch. Use a tag off the `release/v3.6` branch instead.

<ins>Wrangler v4</ins>
`main` is Wrangler major version `v4`, imported as `github.com/rancher/wrangler/v4`.

The major increase is required because [lasso](pkg/lasso/README.md) was merged into
this repository: the packages that used to be imported from
`github.com/rancher/lasso/pkg/...` are now `github.com/rancher/wrangler/v4/pkg/lasso/...`.
Because Go type identity includes the declaring package's import path, the relocated
types are distinct from the ones in the old module even though the definitions are
unchanged, so any consumer that names a lasso type in code that also talks to wrangler
has to move to the new paths.

Migrating:
- Update the wrangler import path from `/v3` to `/v4`.
- Rewrite `github.com/rancher/lasso/pkg/X` to `github.com/rancher/wrangler/v4/pkg/lasso/X`.
- Regenerate any code produced by wrangler's controller-gen, which now emits the new paths.
- Confirm the migration is complete: `go mod graph | grep github.com/rancher/lasso`
  should come back empty.

The current supported release lines are:

| Wrangler Branch | Wrangler Major version | Supported Kubernetes Versions |
|--------------------------|------------------------------------|------------------------------------------------|
| main | v4 | v1.26 - v1.36 |
| release/v3.6 | v3 | v1.23 - v1.35 |
| release/v3.3 | v3 | v1.23 - v1.34 |
| release/v2 | v2 | v1.23 - v1.26 |
