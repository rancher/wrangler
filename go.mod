module github.com/rancher/wrangler

go 1.13

require (
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/moby/locker v1.0.1
	github.com/onsi/gomega v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rancher/lasso v0.0.0-20210224225615-d687d78ea02a
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/tools v0.0.0-20190920225731-5eefd052ad72
	k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver v0.18.0
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v0.18.8
	k8s.io/code-generator v0.18.0
	k8s.io/gengo v0.0.0-20200114144118-36b2048a9120
	k8s.io/kube-aggregator v0.18.0
	sigs.k8s.io/cli-utils v0.16.0
)
