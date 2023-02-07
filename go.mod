module github.com/rancher/wrangler

go 1.16

require (
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/moby/locker v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/rancher/lasso v0.0.0-20210616224652-fc3ebd901c08
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20211202192323-5770296d904e
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	golang.org/x/tools v0.1.0
	k8s.io/api v0.21.14
	k8s.io/apiextensions-apiserver v0.21.14
	k8s.io/apimachinery v0.21.14
	k8s.io/client-go v0.21.14
	k8s.io/code-generator v0.21.14
	k8s.io/gengo v0.0.0-20201214224949-b6c5ce23f027
	k8s.io/kube-aggregator v0.21.14
	sigs.k8s.io/cli-utils v0.21.1
)
