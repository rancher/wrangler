package crd

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"

	// Ensure the gvks are loaded so that apply works correctly
	_ "github.com/rancher/wrangler/pkg/generated/controllers/apiextensions.k8s.io/v1beta1"
)

func (f *Factory) BatchCreateCRDsV1Beta1(ctx context.Context, crds ...CRD) *Factory {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		if _, err := f.CreateCRDsV1Beta1(ctx, crds...); err != nil && f.err == nil {
			f.err = err
		}
	}()
	return f
}

func (f *Factory) CreateCRDsV1Beta1(ctx context.Context, crds ...CRD) (map[schema.GroupVersionKind]*apiextv1beta1.CustomResourceDefinition, error) {
	if len(crds) == 0 {
		return nil, nil
	}

	if ok, err := f.ensureAccessV1Beta1(ctx); err != nil {
		return nil, err
	} else if !ok {
		logrus.Infof("No access to list CRDs, assuming CRDs are pre-created.")
		return nil, err
	}

	crdStatus := map[schema.GroupVersionKind]*apiextv1beta1.CustomResourceDefinition{}

	ready, err := f.getReadyCRDsV1Beta1(ctx)
	if err != nil {
		return nil, err
	}

	for _, crdDef := range crds {
		crd, err := f.createCRDV1Beta1(ctx, crdDef)
		if err != nil {
			return nil, err
		}
		crdStatus[crdDef.GVK] = crd
	}

	ready, err = f.getReadyCRDsV1Beta1(ctx)
	if err != nil {
		return nil, err
	}

	for gvk, crd := range crdStatus {
		if readyCrd, ok := ready[crd.Name]; ok {
			crdStatus[gvk] = readyCrd
		} else {
			if err := f.waitCRDV1Beta1(ctx, crd.Name, gvk, crdStatus); err != nil {
				return nil, err
			}
		}
	}

	return crdStatus, nil
}

func (f *Factory) waitCRDV1Beta1(ctx context.Context, crdName string, gvk schema.GroupVersionKind, crdStatus map[schema.GroupVersionKind]*apiextv1beta1.CustomResourceDefinition) error {
	logrus.Infof("Waiting for CRD %s to become available", crdName)
	defer logrus.Infof("Done waiting for CRD %s to become available", crdName)

	first := true
	return wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		if !first {
			logrus.Infof("Waiting for CRD %s to become available", crdName)
		}
		first = false

		crd, err := f.CRDClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(ctx, crdName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextv1beta1.Established:
				if cond.Status == apiextv1beta1.ConditionTrue {
					crdStatus[gvk] = crd
					return true, err
				}
			case apiextv1beta1.NamesAccepted:
				if cond.Status == apiextv1beta1.ConditionFalse {
					logrus.Infof("Name conflict on %s: %v\n", crdName, cond.Reason)
				}
			}
		}

		return false, ctx.Err()
	})
}

func (f *Factory) createCRDV1Beta1(ctx context.Context, crdDef CRD) (*apiextv1beta1.CustomResourceDefinition, error) {
	crd, err := crdDef.ToCustomResourceDefinitionV1Beta1()
	if err != nil {
		return nil, err
	}

	meta, err := meta.Accessor(crd)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Applying CRD %s", meta.GetName())
	if err := f.apply.WithOwner(crd).ApplyObjects(crd); err != nil {
		return nil, err
	}

	return f.CRDClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(ctx, meta.GetName(), metav1.GetOptions{})
}

func (f *Factory) ensureAccessV1Beta1(ctx context.Context) (bool, error) {
	_, err := f.CRDClient.ApiextensionsV1beta1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if apierrors.IsForbidden(err) {
		return false, nil
	}
	return true, err
}

func (f *Factory) getReadyCRDsV1Beta1(ctx context.Context) (map[string]*apiextv1beta1.CustomResourceDefinition, error) {
	list, err := f.CRDClient.ApiextensionsV1beta1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := map[string]*apiextv1beta1.CustomResourceDefinition{}

	for i, crd := range list.Items {
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextv1beta1.Established:
				if cond.Status == apiextv1beta1.ConditionTrue {
					result[crd.Name] = &list.Items[i]
				}
			}
		}
	}

	return result, nil
}
