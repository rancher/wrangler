module github.com/rancher/wrangler

go 1.12

replace (
	github.com/matryer/moq => github.com/rancher/moq v0.0.0-20190404221404-ee5226d43009
	k8s.io/api => k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go => k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190311093542-50b561225d70
	k8s.io/kubernetes => k8s.io/kubernetes v1.14.1
)

require (
	github.com/ghodss/yaml v1.0.0
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/matryer/moq v0.0.0-20190312154309-6cfb0558e1bd
	github.com/pkg/errors v0.8.1
	github.com/rancher/mapper v0.0.0-20190329182506-3504e44bb041
	github.com/sirupsen/logrus v1.4.1
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a // indirect
	golang.org/x/sync v0.0.0-20190227155943-e225da77a7e6
	golang.org/x/tools v0.0.0-20190411180116-681f9ce8ac52
	k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver v0.0.0-20190325193600-475668423e9f
	k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go v2.0.0-alpha.0.0.20190307161346-7621a5ebb88b+incompatible
	k8s.io/code-generator v0.0.0-20181117043124-c2090bec4d9b
	k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a
	k8s.io/klog v0.2.0
	k8s.io/kube-openapi v0.0.0-20190401085232-94e1e7b7574c // indirect
	k8s.io/kubernetes v1.13.5
)
