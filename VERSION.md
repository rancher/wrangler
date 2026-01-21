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

The current supported release lines are:

| Wrangler Branch | Wrangler Major version | Supported Kubernetes Versions |
|--------------------------|------------------------------------|-------------------------------|
| main | v3 | v1.26 - v1.35 |
| release/v2 | v2 | v1.23 - v1.26 |
