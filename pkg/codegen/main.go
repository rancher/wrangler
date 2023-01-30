package main

import (
	controllergen "github.com/rancher/wrangler/pkg/controller-gen"
	"github.com/rancher/wrangler/pkg/controller-gen/args"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
					v1.PersistentVolume{},
					v1.PersistentVolumeClaim{},
					v1.Pod{},
				},
			},
			discoveryv1.GroupName: {
				Types: []interface{}{
					discoveryv1.EndpointSlice{},
				},
				OutputControllerPackageName: "discovery",
			},
			extensionsv1beta1.GroupName: {
				Types: []interface{}{
					extensionsv1beta1.Ingress{},
				},
			},
			rbacv1.GroupName: {
				Types: []interface{}{
					rbacv1.Role{},
					rbacv1.RoleBinding{},
					rbacv1.ClusterRole{},
					rbacv1.ClusterRoleBinding{},
				},
				OutputControllerPackageName: "rbac",
			},
			appsv1.GroupName: {
				Types: []interface{}{
					appsv1.Deployment{},
					appsv1.DaemonSet{},
					appsv1.StatefulSet{},
				},
			},
			storagev1.GroupName: {
				OutputControllerPackageName: "storage",
				Types: []interface{}{
					storagev1.StorageClass{},
				},
			},
			apiextv1.GroupName: {
				Types: []interface{}{
					apiextv1.CustomResourceDefinition{},
				},
			},
			apiv1.GroupName: {
				Types: []interface{}{
					apiv1.APIService{},
				},
			},
			batchv1.GroupName: {
				Types: []interface{}{
					batchv1.Job{},
				},
			},
			networkingv1.GroupName: {
				Types: []interface{}{
					networkingv1.NetworkPolicy{},
				},
			},
			admissionregistrationv1.GroupName: {
				Types: []interface{}{
					admissionregistrationv1.ValidatingWebhookConfiguration{},
					admissionregistrationv1.MutatingWebhookConfiguration{},
				},
			},
			coordinationv1.GroupName: {
				Types: []interface{}{
					coordinationv1.Lease{},
				},
			},
		},
	})
}
