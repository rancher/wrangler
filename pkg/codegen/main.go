package main

import (
	controllergen "github.com/rancher/wrangler/pkg/controller-gen"
	"github.com/rancher/wrangler/pkg/controller-gen/args"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

func main() {
	controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher/wrangler/pkg/generated",
		Boilerplate:   "scripts/boilerplate.go.txt",
		Groups: map[string]args.Group{
			v1.GroupName: {
				Types: []interface{}{
					v1.Event{},
					v1.Node{},
					v1.Namespace{},
					v1.Secret{},
					v1.Service{},
					v1.ServiceAccount{},
					v1.Endpoints{},
					v1.ConfigMap{},
					v1.PersistentVolumeClaim{},
					v1.Pod{},
				},
				InformersPackage: "k8s.io/client-go/informers",
				ClientSetPackage: "k8s.io/client-go/kubernetes",
				ListersPackage:   "k8s.io/client-go/listers",
			},
			extensionsv1beta1.GroupName: {
				Types: []interface{}{
					extensionsv1beta1.Ingress{},
				},
				InformersPackage: "k8s.io/client-go/informers",
				ClientSetPackage: "k8s.io/client-go/kubernetes",
				ListersPackage:   "k8s.io/client-go/listers",
			},
			rbacv1.GroupName: {
				Types: []interface{}{
					rbacv1.Role{},
					rbacv1.RoleBinding{},
					rbacv1.ClusterRole{},
					rbacv1.ClusterRoleBinding{},
				},
				OutputControllerPackageName: "rbac",
				InformersPackage:            "k8s.io/client-go/informers",
				ClientSetPackage:            "k8s.io/client-go/kubernetes",
				ListersPackage:              "k8s.io/client-go/listers",
			},
			appsv1.GroupName: {
				Types: []interface{}{
					appsv1.Deployment{},
					appsv1.DaemonSet{},
					appsv1.StatefulSet{},
				},
				InformersPackage: "k8s.io/client-go/informers",
				ClientSetPackage: "k8s.io/client-go/kubernetes",
				ListersPackage:   "k8s.io/client-go/listers",
			},
			storagev1.GroupName: {
				OutputControllerPackageName: "storage",
				Types: []interface{}{
					storagev1.StorageClass{},
				},
				InformersPackage: "k8s.io/client-go/informers",
				ClientSetPackage: "k8s.io/client-go/kubernetes",
				ListersPackage:   "k8s.io/client-go/listers",
			},
			v1beta1.GroupName: {
				Types: []interface{}{
					v1beta1.CustomResourceDefinition{},
				},
				ClientSetPackage: "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset",
				InformersPackage: "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions",
				ListersPackage:   "k8s.io/apiextensions-apiserver/pkg/client/listers",
			},
			apiv1.GroupName: {
				Types: []interface{}{
					apiv1.APIService{},
				},
				ClientSetPackage: "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset",
				InformersPackage: "k8s.io/kube-aggregator/pkg/client/informers/externalversions",
				ListersPackage:   "k8s.io/kube-aggregator/pkg/client/listers",
			},
			batchv1.GroupName: {
				Types: []interface{}{
					batchv1.Job{},
				},
				InformersPackage: "k8s.io/client-go/informers",
				ClientSetPackage: "k8s.io/client-go/kubernetes",
				ListersPackage:   "k8s.io/client-go/listers",
			},
		},
	})
}
