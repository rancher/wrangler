package needacert

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"github.com/moby/locker"
	admissionregcontrollers "github.com/rancher/wrangler/v3/pkg/generated/controllers/admissionregistration.k8s.io/v1"
	apiextcontrollers "github.com/rancher/wrangler/v3/pkg/generated/controllers/apiextensions.k8s.io/v1"
	corecontrollers "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/gvk"
	"github.com/rancher/wrangler/v3/pkg/slice"
	"github.com/sirupsen/logrus"
	adminregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/cert"
)

var (
	SecretAnnotation = "need-a-cert.cattle.io/secret-name"
	DNSAnnotation    = "need-a-cert.cattle.io/dns-name"
)

const (
	byServiceIndex = "byService"
)

func Register(ctx context.Context,
	secrets corecontrollers.SecretController,
	service corecontrollers.ServiceController,
	mutatingController admissionregcontrollers.MutatingWebhookConfigurationController,
	validatingController admissionregcontrollers.ValidatingWebhookConfigurationController,
	crdController apiextcontrollers.CustomResourceDefinitionController) {
	h := handler{
		secretsCache:       secrets.Cache(),
		secrets:            secrets,
		services:           service,
		serviceCache:       service.Cache(),
		mutatingWebHooks:   mutatingController,
		validatingWebHooks: validatingController,
		crds:               crdController,
	}

	mutatingController.Cache().AddIndexer(byServiceIndex, mutatingWebhookServices)
	validatingController.Cache().AddIndexer(byServiceIndex, validatingWebhookServices)
	crdController.Cache().AddIndexer(byServiceIndex, crdWebhookServices)

	mutatingController.OnChange(ctx, "need-a-cert", h.OnMutationWebhookChange)
	validatingController.OnChange(ctx, "need-a-cert", h.OnValidatingWebhookChange)
	crdController.OnChange(ctx, "need-a-cert", h.OnCRDChange)
	service.OnChange(ctx, "need-a-cert", h.OnService)
	secrets.OnChange(ctx, "need-a-cert", h.OnSecretChange)
}

// OnSecretChange handles Secret changes and enqueues the related Service.
func (h *handler) OnSecretChange(key string, secret *corev1.Secret) (*corev1.Secret, error) {
	if secret == nil {
		return secret, nil
	}

	services, err := h.services.List(secret.Namespace, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, service := range services.Items {
		if service.Annotations[SecretAnnotation] == secret.Name {
			h.services.Enqueue(service.Namespace, service.Name)
		}
	}

	return secret, nil
}

type handler struct {
	locker             locker.Locker
	secretsCache       corecontrollers.SecretCache
	secrets            corecontrollers.SecretClient
	serviceCache       corecontrollers.ServiceCache
	services           corecontrollers.ServiceController
	mutatingWebHooks   admissionregcontrollers.MutatingWebhookConfigurationController
	validatingWebHooks admissionregcontrollers.ValidatingWebhookConfigurationController
	crds               apiextcontrollers.CustomResourceDefinitionController
}

func validatingWebhookServices(obj *adminregv1.ValidatingWebhookConfiguration) (result []string, _ error) {
	for _, webhook := range obj.Webhooks {
		if webhook.ClientConfig.Service != nil {
			result = append(result, webhook.ClientConfig.Service.Namespace+"/"+webhook.ClientConfig.Service.Name)
		}
	}
	return
}

func crdWebhookServices(obj *apiextv1.CustomResourceDefinition) (result []string, _ error) {
	if obj.Spec.Conversion != nil &&
		obj.Spec.Conversion.Webhook != nil &&
		obj.Spec.Conversion.Webhook.ClientConfig != nil &&
		obj.Spec.Conversion.Webhook.ClientConfig.Service != nil {
		return []string{
			fmt.Sprintf("%s/%s",
				obj.Spec.Conversion.Webhook.ClientConfig.Service.Namespace,
				obj.Spec.Conversion.Webhook.ClientConfig.Service.Name),
		}, nil
	}
	return nil, nil
}

func mutatingWebhookServices(obj *adminregv1.MutatingWebhookConfiguration) (result []string, _ error) {
	for _, webhook := range obj.Webhooks {
		if webhook.ClientConfig.Service != nil {
			result = append(result, webhook.ClientConfig.Service.Namespace+"/"+webhook.ClientConfig.Service.Name)
		}
	}
	return
}

func (h *handler) OnMutationWebhookChange(key string, webhook *adminregv1.MutatingWebhookConfiguration) (*adminregv1.MutatingWebhookConfiguration, error) {
	if webhook == nil {
		return nil, nil
	}
	needUpdate := false
	for i, webhookConfig := range webhook.Webhooks {
		if webhookConfig.ClientConfig.Service == nil || webhookConfig.ClientConfig.Service.Name == "" {
			continue
		}

		service, err := h.serviceCache.Get(webhookConfig.ClientConfig.Service.Namespace, webhookConfig.ClientConfig.Service.Name)
		if apierror.IsNotFound(err) {
			// OnService will be called when the service is created, which will eventually update the webhook, so no
			// need to enqueue anything if we don't find the service
			return webhook, nil
		} else if err != nil {
			return nil, err
		}

		secret, err := h.generateSecret(service)
		if err != nil {
			return nil, err
		} else if secret == nil {
			continue
		}

		if !bytes.Equal(webhookConfig.ClientConfig.CABundle, secret.Data[corev1.TLSCertKey]) {
			webhook = webhook.DeepCopy()
			webhook.Webhooks[i].ClientConfig.CABundle = secret.Data[corev1.TLSCertKey]
			needUpdate = true
		}
	}

	if needUpdate {
		logrus.Debugf("Updating MutatingWebhookConfiguration %s/%s", webhook.GetNamespace(), webhook.GetName())
		webhook, err := h.mutatingWebHooks.Update(webhook)
		if err != nil {
			return webhook, err
		}
	}

	return webhook, nil
}

func (h *handler) OnValidatingWebhookChange(key string, webhook *adminregv1.ValidatingWebhookConfiguration) (*adminregv1.ValidatingWebhookConfiguration, error) {
	if webhook == nil {
		return nil, nil
	}
	needUpdate := false
	for i, webhookConfig := range webhook.Webhooks {
		if webhookConfig.ClientConfig.Service == nil || webhookConfig.ClientConfig.Service.Name == "" {
			continue
		}

		service, err := h.serviceCache.Get(webhookConfig.ClientConfig.Service.Namespace, webhookConfig.ClientConfig.Service.Name)
		if apierror.IsNotFound(err) {
			// OnService will be called when the service is created, which will eventually update the webhook, so no
			// need to enqueue anything if we don't find the service
			return webhook, nil
		} else if err != nil {
			return nil, err
		}

		secret, err := h.generateSecret(service)
		if err != nil {
			return nil, err
		} else if secret == nil {
			continue
		}

		if !bytes.Equal(webhookConfig.ClientConfig.CABundle, secret.Data[corev1.TLSCertKey]) {
			webhook = webhook.DeepCopy()
			webhook.Webhooks[i].ClientConfig.CABundle = secret.Data[corev1.TLSCertKey]
			needUpdate = true
		}
	}

	if needUpdate {
		logrus.Debugf("Updating ValidatingWebhookConfiguration %s/%s", webhook.GetNamespace(), webhook.GetName())
		webhook, err := h.validatingWebHooks.Update(webhook)
		if err != nil {
			return webhook, err
		}
	}

	return webhook, nil
}

func (h *handler) OnService(key string, service *corev1.Service) (*corev1.Service, error) {
	if service == nil {
		return service, nil
	}

	_, err := h.generateSecret(service)
	if err != nil {
		return nil, err
	}

	indexKey := service.Namespace + "/" + service.Name
	mutating, err := h.mutatingWebHooks.Cache().GetByIndex(byServiceIndex, indexKey)
	if err != nil {
		return nil, err
	}
	for _, mutating := range mutating {
		h.mutatingWebHooks.Enqueue(mutating.Name)
	}

	validating, err := h.validatingWebHooks.Cache().GetByIndex(byServiceIndex, indexKey)
	if err != nil {
		return nil, err
	}
	for _, validating := range validating {
		h.validatingWebHooks.Enqueue(validating.Name)
	}

	crd, err := h.crds.Cache().GetByIndex(byServiceIndex, indexKey)
	if err != nil {
		return nil, err
	}
	for _, crd := range crd {
		h.crds.Enqueue(crd.Name)
	}

	return service, err
}

func (h *handler) OnCRDChange(key string, crd *apiextv1.CustomResourceDefinition) (*apiextv1.CustomResourceDefinition, error) {
	if crd == nil || crd.Spec.Conversion == nil || crd.Spec.Conversion.Webhook == nil ||
		crd.Spec.Conversion.Webhook.ClientConfig == nil ||
		crd.Spec.Conversion.Webhook.ClientConfig.Service == nil ||
		crd.Spec.Conversion.Webhook.ClientConfig.Service.Name == "" {
		return crd, nil
	}

	service, err := h.serviceCache.Get(crd.Spec.Conversion.Webhook.ClientConfig.Service.Namespace,
		crd.Spec.Conversion.Webhook.ClientConfig.Service.Name)

	if apierror.IsNotFound(err) {
		// OnService will be called when the service is created, which will eventually update the CRD, so no
		// need to enqueue anything if we don't find the service
		return crd, nil
	} else if err != nil {
		return nil, err
	}

	secret, err := h.generateSecret(service)
	if err != nil || secret == nil {
		return crd, nil
	}

	if !bytes.Equal(crd.Spec.Conversion.Webhook.ClientConfig.CABundle, secret.Data[corev1.TLSCertKey]) {
		crd := crd.DeepCopy()
		crd.Spec.Conversion.Webhook.ClientConfig.CABundle = secret.Data[corev1.TLSCertKey]
		return h.crds.Update(crd)
	}

	return crd, nil
}

func (h *handler) generateSecret(service *corev1.Service) (*corev1.Secret, error) {
	secretName := service.Annotations[SecretAnnotation]
	if secretName == "" {
		return nil, nil
	}

	lockKey := service.Namespace + "/" + service.Name
	h.locker.Lock(lockKey)
	defer h.locker.Unlock(lockKey)

	dnsNameSet := sets.NewString(service.Name+"."+service.Namespace,
		service.Name+"."+service.Namespace+".svc",
		service.Name+"."+service.Namespace+".svc.cluster.local")
	for k, v := range service.Annotations {
		if !strings.HasPrefix(k, DNSAnnotation) {
			continue
		}
		dnsNameSet.Insert(v)
	}

	// this is sorted
	dnsNames := dnsNameSet.List()
	secret, err := h.secretsCache.Get(service.Namespace, secretName)
	if apierror.IsNotFound(err) {
		newSecret, err := h.createSecret(service, service.Namespace, secretName, dnsNames)
		if err != nil {
			return nil, err
		}
		created, err := h.secrets.Create(newSecret)
		if apierror.IsAlreadyExists(err) {
			existing, getErr := h.secrets.Get(service.Namespace, secretName, metav1.GetOptions{})
			if getErr != nil {
				return nil, getErr
			}
			if err := h.scheduleNextCertCheck(service, existing, nil); err != nil {
				return nil, fmt.Errorf("schedule next cert check failed for existing secret: %w", err)
			}
			return existing, nil
		} else if err != nil {
			return created, err
		}

		if err := h.scheduleNextCertCheck(service, created, nil); err != nil {
			return nil, fmt.Errorf("schedule next cert check failed for new secret: %w", err)
		}
		return created, nil
	} else if err != nil {
		return nil, err
	}

	if updated, err := h.updateSecret(service, secret, dnsNames); err != nil {
		return nil, err
	} else if updated != nil {
		return h.secrets.Update(updated)
	}

	return secret, nil
}

func (h *handler) updateSecret(owner runtime.Object, secret *corev1.Secret, dnsNames []string) (*corev1.Secret, error) {
	cert, err := parseCert(secret)
	if err != nil {
		return nil, err
	}

	metaObj, err := meta.Accessor(owner)
	if err != nil {
		return nil, err
	}

	if err := h.scheduleNextCertCheck(metaObj, secret, cert); err != nil {
		return nil, fmt.Errorf("failed to schedule next cert check: %w", err)
	}

	logrus.Debugf("checking cert %s for %s/%s", cert.Subject.CommonName, secret.Namespace, secret.Name)
	if time.Now().Add(24*60*time.Hour).After(cert.NotAfter) ||
		len(cert.DNSNames) == 0 ||
		!slice.StringsEqual(cert.DNSNames[1:], dnsNames) {
		logrus.Debugf("regenerating cert %s for %s/%s", cert.Subject.CommonName, secret.Namespace, secret.Name)
		newSecret, err := h.createSecret(owner, secret.Namespace, secret.Name, dnsNames)
		if err != nil {
			return nil, err
		}
		secret = secret.DeepCopy()
		secret.Data = newSecret.Data
		return secret, nil
	} else {
		logrus.Debugf("cert %s for %s/%s is valid until %s and covers %v", cert.Subject.CommonName, secret.Namespace, secret.Name, cert.NotAfter, cert.DNSNames)
	}

	return nil, nil
}

func (h *handler) scheduleNextCertCheck(obj metav1.Object, secret *corev1.Secret, cert *x509.Certificate) error {
	if cert == nil {
		var err error
		cert, err = parseCert(secret)
		if err != nil {
			return fmt.Errorf("cannot parse certificate: %w", err)
		}
	}

	renewBefore := 24 * time.Hour
	nextCheck := time.Until(cert.NotAfter.Add(-renewBefore))
	if nextCheck < time.Minute {
		nextCheck = time.Minute
	}

	logrus.Debugf("Next cert check for %s/%s scheduled in %v (expires %v)",
		obj.GetNamespace(), obj.GetName(), nextCheck.Round(time.Second), cert.NotAfter)

	h.services.EnqueueAfter(obj.GetNamespace(), obj.GetName(), nextCheck)
	return nil
}

func (h *handler) createSecret(owner runtime.Object, ns, name string, dnsNames []string) (*corev1.Secret, error) {
	cert, key, err := cert.GenerateSelfSignedCertKey(ns+"-"+name, nil, dnsNames)
	if err != nil {
		return nil, err
	}

	meta, err := meta.Accessor(owner)
	if err != nil {
		return nil, err
	}

	gvk, err := gvk.Get(owner)
	if err != nil {
		return nil, err
	}

	ref := metav1.OwnerReference{
		Name: meta.GetName(),
		UID:  meta.GetUID(),
	}
	ref.APIVersion, ref.Kind = gvk.ToAPIVersionAndKind()

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       ns,
			OwnerReferences: []metav1.OwnerReference{ref},
		},
		Data: map[string][]byte{
			corev1.TLSCertKey:       cert,
			corev1.TLSPrivateKeyKey: key,
		},
		Type: corev1.SecretTypeTLS,
	}, nil
}

func parseCert(secret *corev1.Secret) (*x509.Certificate, error) {
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret or secret.Data is nil")
	}

	tlsPair, err := tls.X509KeyPair(secret.Data[corev1.TLSCertKey], secret.Data[corev1.TLSPrivateKeyKey])
	if err != nil || len(tlsPair.Certificate) == 0 {
		return nil, fmt.Errorf("failed to load TLS keypair: %w", err)
	}

	cert, err := x509.ParseCertificate(tlsPair.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse X509 certificate: %w", err)
	}

	return cert, nil
}
