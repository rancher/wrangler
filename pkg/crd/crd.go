// Package crd helps with the installation of CustomResourceDefinitions.
package crd

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	clientv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	buffSize     = 4096
	waitDuration = 1 * time.Minute
	waitInterval = 500 * time.Millisecond
)

// BatchCreateCRDs will create or update the list of crds if they do not exist.
func BatchCreateCRDs(ctx context.Context, crdClient clientv1.CustomResourceDefinitionInterface, crds []*apiextv1.CustomResourceDefinition) error {
	existingCRDs, err := getExistingCRD(ctx, crdClient)
	if err != nil {
		// we do not need to return this error because we can still attempt to create the CRDs if the list fails
		logrus.Warnf("unable to check for existing CRDs: %s", err.Error())
	}

	// ensure each CRD exist
	for _, crd := range crds {
		err = ensureCRD(ctx, crd, crdClient, existingCRDs)
		if err != nil {
			return err
		}
	}

	existingCRDs, err = getExistingCRD(ctx, crdClient)
	if err != nil {
		return fmt.Errorf("unable to check for existing CRDs: %w", err)
	}
	// ensure each CRD is ready
	for _, crd := range crds {
		if existingCRD, ok := existingCRDs[crd.Name]; ok && crdIsReady(existingCRD) {
			continue
		}
		// wait for the CRD to be ready
		err := waitCRD(ctx, crd.Name, crdClient)
		if err != nil {
			return fmt.Errorf("failed waiting on CRD '%s': %w", crd.Name, err)
		}
	}

	return nil
}

// getExistingCRD returns a map of all CRD resource on the cluster.
func getExistingCRD(ctx context.Context, crdClient clientv1.CustomResourceDefinitionInterface) (map[string]*apiextv1.CustomResourceDefinition, error) {
	storedCRDs, err := crdClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list crds: %w", err)
	}

	// convert existingCRDs to a map for faster lookup
	existingCRDs := map[string]*apiextv1.CustomResourceDefinition{}
	for i := range storedCRDs.Items {
		crd := &storedCRDs.Items[i]
		existingCRDs[crd.Name] = crd
	}
	return existingCRDs, nil
}

// ensureCRD checks if there is an existing CRD that matches if not it will attempt to create or update the CRD.
func ensureCRD(ctx context.Context, crd *apiextv1.CustomResourceDefinition, crdClient clientv1.CustomResourceDefinitionInterface, existingCRDs map[string]*apiextv1.CustomResourceDefinition) error {
	existingCRD, exist := existingCRDs[crd.Name]
	if !exist {
		logrus.Infof("Creating embedded CRD %s", crd.Name)
		_, err := crdClient.Create(ctx, crd, metav1.CreateOptions{})
		if err == nil {
			return nil
		}
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create crd '%s': %w", crd.Name, err)
		}
		// item was not in intial list but does exist so we will attempt to get the latest resourceVersion
		existingCRD, err = crdClient.Get(ctx, crd.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get exiting '%s' CRD: %w", crd.Name, err)
		}
		// fallthrough and attempt and update
	}
	// only keep the resource version for the desired object
	crd.ResourceVersion = existingCRD.ResourceVersion
	// if the CRD exist attempt to update it
	logrus.Infof("Updating embedded CRD %s", crd.Name)
	_, err := crdClient.Update(ctx, crd, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update crd '%s': %w", crd.Name, err)
	}
	return nil
}

// waitCRD repeatably checks CRD status until it is established.
func waitCRD(ctx context.Context, crdName string, crdClient clientv1.CustomResourceDefinitionInterface) error {
	logrus.Infof("Waiting for CRD %s to become available", crdName)
	defer logrus.Infof("Done waiting for CRD %s to become available", crdName)

	return wait.PollImmediate(waitInterval, waitDuration, func() (bool, error) {
		crd, err := crdClient.Get(ctx, crdName, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get CRD '%s' for status checking: %w", crdName, err)
		}
		if crdIsReady(crd) {
			return true, nil
		}

		return false, nil
	})
}

func crdIsReady(crd *apiextv1.CustomResourceDefinition) bool {
	for i := range crd.Status.Conditions {
		cond := &crd.Status.Conditions[i]
		switch cond.Type {
		case apiextv1.Established:
			if cond.Status == apiextv1.ConditionTrue {
				return true
			}
		case apiextv1.NamesAccepted:
			if cond.Status == apiextv1.ConditionFalse {
				logrus.Infof("Name conflict for CRD %s: %s", crd.Name, cond.Reason)
			}
		}
	}
	return false
}
