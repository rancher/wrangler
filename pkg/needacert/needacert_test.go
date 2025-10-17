package needacert

import (
	"bytes"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/rancher/wrangler/v3/pkg/generic/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	adminregv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
)

func TestCreateSecret(t *testing.T) {
	h := &handler{}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "ns",
		},
	}
	dnsNames := []string{"svc.ns", "svc.ns.svc"}
	secret, err := h.createSecret(service, "ns", "mysecret", dnsNames)
	assert.NoError(t, err)
	assert.Equal(t, "mysecret", secret.Name)
	assert.Equal(t, "ns", secret.Namespace)
	assert.Equal(t, corev1.SecretTypeTLS, secret.Type)
	assert.NotEmpty(t, secret.Data[corev1.TLSCertKey])
	assert.NotEmpty(t, secret.Data[corev1.TLSPrivateKeyKey])
}

func TestUpdateSecret_ExpiredCert_ManyParallel(t *testing.T) {
	const runs = 50
	for i := 0; i < runs; i++ {
		t.Run(fmt.Sprintf("run-%d", i), func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
			mockServices.EXPECT().EnqueueAfter("ns", "svc", gomock.Any()).AnyTimes()
			h := &handler{services: mockServices}
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc",
					Namespace: "ns",
				},
			}
			dnsNames := []string{"svc.ns", "svc.ns.svc"}

			certPEM, keyPEM, err := cert.GenerateSelfSignedCertKeyWithOptions(cert.SelfSignedCertKeyOptions{
				Host:         "ns-mysecret",
				AlternateDNS: dnsNames,
				MaxAge:       1 * time.Second,
			})
			assert.NoError(t, err)

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ns",
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					corev1.TLSCertKey:       certPEM,
					corev1.TLSPrivateKeyKey: keyPEM,
				},
			}

			time.Sleep(2 * time.Second)

			parsedCerts, parseErr := cert.ParseCertsPEM(certPEM)
			assert.NoError(t, parseErr)
			updated, err := h.updateSecret(service, secret, dnsNames, parsedCerts[0])
			assert.NoError(t, err)
			assert.NotNil(t, updated)
		})
	}
}

func TestGenerateSecret_NoAnnotation(t *testing.T) {
	h := &handler{}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "svc",
			Namespace:   "ns",
			Annotations: map[string]string{},
		},
	}
	secret, err := h.generateSecret(service)
	assert.NoError(t, err)
	assert.Nil(t, secret)
}

func TestHandler_OnMutationWebhookChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockServiceCache := fake.NewMockCacheInterface[*corev1.Service](ctrl)
	mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
	mockSecretsCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
	mockSecrets := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockMutatingWebHooks := fake.NewMockNonNamespacedControllerInterface[*adminregv1.MutatingWebhookConfiguration, *adminregv1.MutatingWebhookConfigurationList](ctrl)

	mockService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "ns",
			Annotations: map[string]string{
				SecretAnnotation: "mysecret",
			},
		},
	}
	certPEM, keyPEM, _ := cert.GenerateSelfSignedCertKey("ns-mysecret", nil, []string{"svc.ns", "svc.ns.svc"})
	mockSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysecret",
			Namespace: "ns",
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       certPEM,
			corev1.TLSPrivateKeyKey: keyPEM,
		},
	}

	mockServices.EXPECT().
		EnqueueAfter("ns", "svc", gomock.Any()).
		AnyTimes()

	mockServiceCache.EXPECT().
		Get("ns", "svc").
		Return(mockService, nil).
		Times(2)

	mockSecretsCache.EXPECT().
		Get("ns", "mysecret").
		Return(mockSecret, nil).
		Times(2)

	mockSecrets.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
			return secret, nil
		}).Times(2)

	mockMutatingWebHooks.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(webhook *adminregv1.MutatingWebhookConfiguration) (*adminregv1.MutatingWebhookConfiguration, error) {
			return webhook, nil
		}).Times(1)

	h := &handler{
		services:         mockServices,
		serviceCache:     mockServiceCache,
		secretsCache:     mockSecretsCache,
		secrets:          mockSecrets,
		mutatingWebHooks: mockMutatingWebHooks,
	}

	webhook := &adminregv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "webhook",
		},
		Webhooks: []adminregv1.MutatingWebhook{
			{
				Name: "wh",
				ClientConfig: adminregv1.WebhookClientConfig{
					Service: &adminregv1.ServiceReference{
						Namespace: "ns",
						Name:      "svc",
					},
					CABundle: []byte{},
				},
			},
			{
				Name: "wh2",
				ClientConfig: adminregv1.WebhookClientConfig{
					Service: &adminregv1.ServiceReference{
						Namespace: "ns",
						Name:      "svc",
					},
					CABundle: []byte{},
				},
			},
		},
	}

	updated, err := h.OnMutationWebhookChange("key", webhook)
	assert.NoError(t, err)
	assert.NotNil(t, updated)
	assert.NotEmpty(t, updated.Webhooks[0].ClientConfig.CABundle)
	assert.NotEmpty(t, updated.Webhooks[1].ClientConfig.CABundle)
	assert.True(t, bytes.HasPrefix(updated.Webhooks[0].ClientConfig.CABundle, []byte("-----BEGIN CERTIFICATE-----")))
	assert.True(t, bytes.HasPrefix(updated.Webhooks[1].ClientConfig.CABundle, []byte("-----BEGIN CERTIFICATE-----")))
}

func TestHandler_OnValidatingWebhookChange_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const runs = 10
	for i := 0; i < runs; i++ {
		t.Run(fmt.Sprintf("run-%d", i), func(t *testing.T) {
			t.Parallel()

			mockServiceCache := fake.NewMockCacheInterface[*corev1.Service](ctrl)
			mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
			mockSecretsCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
			mockSecrets := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
			mockValidatingWebHooks := fake.NewMockNonNamespacedControllerInterface[*adminregv1.ValidatingWebhookConfiguration, *adminregv1.ValidatingWebhookConfigurationList](ctrl)

			mockService := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc",
					Namespace: "ns",
					Annotations: map[string]string{
						SecretAnnotation: "mysecret",
					},
				},
			}
			certPEM, keyPEM, _ := cert.GenerateSelfSignedCertKey("ns-mysecret", nil, []string{"svc.ns", "svc.ns.svc"})
			mockSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ns",
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					corev1.TLSCertKey:       certPEM,
					corev1.TLSPrivateKeyKey: keyPEM,
				},
			}

			mockServices.EXPECT().
				EnqueueAfter("ns", "svc", gomock.Any()).
				AnyTimes()

			mockServiceCache.EXPECT().
				Get("ns", "svc").
				Return(mockService, nil).
				Times(2)

			mockSecretsCache.EXPECT().
				Get("ns", "mysecret").
				Return(mockSecret, nil).
				Times(2)

			mockSecrets.EXPECT().
				Update(gomock.Any()).
				DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
					return secret, nil
				}).
				Times(2)

			mockValidatingWebHooks.EXPECT().
				Update(gomock.Any()).
				DoAndReturn(func(webhook *adminregv1.ValidatingWebhookConfiguration) (*adminregv1.ValidatingWebhookConfiguration, error) {
					return webhook, nil
				}).Times(1)

			h := &handler{
				services:           mockServices,
				serviceCache:       mockServiceCache,
				secretsCache:       mockSecretsCache,
				secrets:            mockSecrets,
				validatingWebHooks: mockValidatingWebHooks,
			}

			webhook := &adminregv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "webhook",
				},
				Webhooks: []adminregv1.ValidatingWebhook{
					{
						Name: "wh",
						ClientConfig: adminregv1.WebhookClientConfig{
							Service: &adminregv1.ServiceReference{
								Namespace: "ns",
								Name:      "svc",
							},
							CABundle: []byte{},
						},
					},
					{
						Name: "wh2",
						ClientConfig: adminregv1.WebhookClientConfig{
							Service: &adminregv1.ServiceReference{
								Namespace: "ns",
								Name:      "svc",
							},
							CABundle: []byte{},
						},
					},
				},
			}

			updated, err := h.OnValidatingWebhookChange("key", webhook)
			assert.NoError(t, err)
			assert.NotNil(t, updated)
			assert.NotEmpty(t, updated.Webhooks[0].ClientConfig.CABundle)
			assert.NotEmpty(t, updated.Webhooks[1].ClientConfig.CABundle)
			assert.True(t, bytes.HasPrefix(updated.Webhooks[0].ClientConfig.CABundle, []byte("-----BEGIN CERTIFICATE-----")))
			assert.True(t, bytes.HasPrefix(updated.Webhooks[1].ClientConfig.CABundle, []byte("-----BEGIN CERTIFICATE-----")))
		})
	}
}

func TestHandler_OnMutationWebhookChange_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const runs = 10
	for i := 0; i < runs; i++ {
		t.Run(fmt.Sprintf("run-%d", i), func(t *testing.T) {
			t.Parallel()

			mockServiceCache := fake.NewMockCacheInterface[*corev1.Service](ctrl)
			mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
			mockSecretsCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
			mockSecrets := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
			mockMutatingWebHooks := fake.NewMockNonNamespacedControllerInterface[*adminregv1.MutatingWebhookConfiguration, *adminregv1.MutatingWebhookConfigurationList](ctrl)

			mockService := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc",
					Namespace: "ns",
					Annotations: map[string]string{
						SecretAnnotation: "mysecret",
					},
				},
			}
			certPEM, keyPEM, _ := cert.GenerateSelfSignedCertKey("ns-mysecret", nil, []string{"svc.ns", "svc.ns.svc"})
			mockSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ns",
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					corev1.TLSCertKey:       certPEM,
					corev1.TLSPrivateKeyKey: keyPEM,
				},
			}

			mockServices.EXPECT().
				EnqueueAfter("ns", "svc", gomock.Any()).
				AnyTimes()

			mockServiceCache.EXPECT().
				Get("ns", "svc").
				Return(mockService, nil)

			mockSecretsCache.EXPECT().
				Get("ns", "mysecret").
				Return(mockSecret, nil)

			mockSecrets.EXPECT().
				Update(gomock.Any()).
				DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
					return secret, nil
				})

			mockMutatingWebHooks.EXPECT().
				Update(gomock.Any()).
				DoAndReturn(func(webhook *adminregv1.MutatingWebhookConfiguration) (*adminregv1.MutatingWebhookConfiguration, error) {
					return webhook, nil
				})

			h := &handler{
				services:         mockServices,
				serviceCache:     mockServiceCache,
				secretsCache:     mockSecretsCache,
				secrets:          mockSecrets,
				mutatingWebHooks: mockMutatingWebHooks,
			}

			webhook := &adminregv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "webhook",
				},
				Webhooks: []adminregv1.MutatingWebhook{
					{
						Name: "wh",
						ClientConfig: adminregv1.WebhookClientConfig{
							Service: &adminregv1.ServiceReference{
								Namespace: "ns",
								Name:      "svc",
							},
							CABundle: []byte{},
						},
					},
				},
			}

			updated, err := h.OnMutationWebhookChange("key", webhook)
			assert.NoError(t, err)
			assert.NotNil(t, updated)
			assert.NotEmpty(t, updated.Webhooks[0].ClientConfig.CABundle)
			assert.True(t, bytes.HasPrefix(updated.Webhooks[0].ClientConfig.CABundle, []byte("-----BEGIN CERTIFICATE-----")))
		})
	}
}

func TestHandler_OnService_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const runs = 10
	for i := 0; i < runs; i++ {
		t.Run(fmt.Sprintf("run-%d", i), func(t *testing.T) {
			t.Parallel()

			mockServiceCache := fake.NewMockCacheInterface[*corev1.Service](ctrl)
			mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
			mockSecretsCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
			mockSecrets := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
			mockMutatingWebHooks := fake.NewMockNonNamespacedControllerInterface[*adminregv1.MutatingWebhookConfiguration, *adminregv1.MutatingWebhookConfigurationList](ctrl)
			mockValidatingWebHooks := fake.NewMockNonNamespacedControllerInterface[*adminregv1.ValidatingWebhookConfiguration, *adminregv1.ValidatingWebhookConfigurationList](ctrl)
			mockCRDs := fake.NewMockNonNamespacedControllerInterface[*apiextv1.CustomResourceDefinition, *apiextv1.CustomResourceDefinitionList](ctrl)

			mockMutatingCache := fake.NewMockNonNamespacedCacheInterface[*adminregv1.MutatingWebhookConfiguration](ctrl)
			mockValidatingCache := fake.NewMockNonNamespacedCacheInterface[*adminregv1.ValidatingWebhookConfiguration](ctrl)
			mockCRDsCache := fake.NewMockNonNamespacedCacheInterface[*apiextv1.CustomResourceDefinition](ctrl)

			mockMutatingWebHooks.EXPECT().Cache().Return(mockMutatingCache).AnyTimes()
			mockValidatingWebHooks.EXPECT().Cache().Return(mockValidatingCache).AnyTimes()
			mockCRDs.EXPECT().Cache().Return(mockCRDsCache).AnyTimes()

			mockMutatingCache.EXPECT().GetByIndex(gomock.Any(), gomock.Any()).Return([]*adminregv1.MutatingWebhookConfiguration{}, nil).AnyTimes()
			mockValidatingCache.EXPECT().GetByIndex(gomock.Any(), gomock.Any()).Return([]*adminregv1.ValidatingWebhookConfiguration{}, nil).AnyTimes()
			mockCRDsCache.EXPECT().GetByIndex(gomock.Any(), gomock.Any()).Return([]*apiextv1.CustomResourceDefinition{}, nil).AnyTimes()

			mockMutatingWebHooks.EXPECT().Enqueue(gomock.Any()).AnyTimes()
			mockValidatingWebHooks.EXPECT().Enqueue(gomock.Any()).AnyTimes()
			mockCRDs.EXPECT().Enqueue(gomock.Any()).AnyTimes()

			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc",
					Namespace: "ns",
					Annotations: map[string]string{
						SecretAnnotation: "mysecret",
					},
				},
			}

			certPEM, keyPEM, _ := cert.GenerateSelfSignedCertKey("ns-mysecret", nil, []string{"svc.ns", "svc.ns.svc"})
			mockSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ns",
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					corev1.TLSCertKey:       certPEM,
					corev1.TLSPrivateKeyKey: keyPEM,
				},
			}

			mockServices.EXPECT().
				EnqueueAfter("ns", "svc", gomock.Any()).
				AnyTimes()
			mockSecretsCache.EXPECT().
				Get("ns", "mysecret").
				Return(mockSecret, nil).AnyTimes()
			mockSecrets.EXPECT().
				Update(gomock.Any()).
				DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
					return secret, nil
				}).AnyTimes()

			h := &handler{
				services:           mockServices,
				serviceCache:       mockServiceCache,
				secretsCache:       mockSecretsCache,
				secrets:            mockSecrets,
				mutatingWebHooks:   mockMutatingWebHooks,
				validatingWebHooks: mockValidatingWebHooks,
				crds:               mockCRDs,
			}

			_, err := h.OnService("ns/svc", service)
			assert.NoError(t, err)
		})
	}
}

func TestHandler_OnCRDChange_Parallel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const runs = 10
	for i := 0; i < runs; i++ {
		t.Run(fmt.Sprintf("run-%d", i), func(t *testing.T) {
			t.Parallel()

			mockServiceCache := fake.NewMockCacheInterface[*corev1.Service](ctrl)
			mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
			mockSecretsCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
			mockSecrets := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
			mockCRDs := fake.NewMockNonNamespacedControllerInterface[*apiextv1.CustomResourceDefinition, *apiextv1.CustomResourceDefinitionList](ctrl)

			mockCRDsCache := fake.NewMockNonNamespacedCacheInterface[*apiextv1.CustomResourceDefinition](ctrl)
			mockCRDs.EXPECT().Cache().Return(mockCRDsCache).AnyTimes()
			mockCRDs.EXPECT().Enqueue(gomock.Any()).AnyTimes()

			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc",
					Namespace: "ns",
					Annotations: map[string]string{
						SecretAnnotation: "mysecret",
					},
				},
			}

			certPEM, keyPEM, _ := cert.GenerateSelfSignedCertKey("ns-mysecret", nil, []string{"svc.ns", "svc.ns.svc"})
			mockSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ns",
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					corev1.TLSCertKey:       certPEM,
					corev1.TLSPrivateKeyKey: keyPEM,
				},
			}

			mockServices.EXPECT().
				EnqueueAfter("ns", "svc", gomock.Any()).
				AnyTimes()
			mockServiceCache.EXPECT().
				Get("ns", "svc").
				Return(service, nil).AnyTimes()
			mockSecretsCache.EXPECT().
				Get("ns", "mysecret").
				Return(mockSecret, nil).AnyTimes()
			mockSecrets.EXPECT().
				Update(gomock.Any()).
				DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
					return secret, nil
				}).AnyTimes()
			mockSecrets.EXPECT().
				Create(gomock.Any()).
				DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
					return secret, nil
				}).AnyTimes()
			mockCRDs.EXPECT().
				Update(gomock.Any()).
				DoAndReturn(func(crd *apiextv1.CustomResourceDefinition) (*apiextv1.CustomResourceDefinition, error) {
					return crd, nil
				}).AnyTimes()

			h := &handler{
				services:     mockServices,
				serviceCache: mockServiceCache,
				secretsCache: mockSecretsCache,
				secrets:      mockSecrets,
				crds:         mockCRDs,
			}

			crd := &apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "crd",
				},
				Spec: apiextv1.CustomResourceDefinitionSpec{
					Conversion: &apiextv1.CustomResourceConversion{
						Strategy: apiextv1.WebhookConverter,
						Webhook: &apiextv1.WebhookConversion{
							ClientConfig: &apiextv1.WebhookClientConfig{
								Service: &apiextv1.ServiceReference{
									Namespace: "ns",
									Name:      "svc",
								},
								CABundle: []byte{},
							},
						},
					},
				},
			}

			updated, err := h.OnCRDChange("key", crd)
			assert.NoError(t, err)
			assert.NotNil(t, updated)
			assert.NotEmpty(t, updated.Spec.Conversion.Webhook.ClientConfig.CABundle)
			assert.True(t, bytes.HasPrefix(updated.Spec.Conversion.Webhook.ClientConfig.CABundle, []byte("-----BEGIN CERTIFICATE-----")))
		})
	}
}

func TestHandler_GenerateSecret_Race(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
	mockSecretsCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
	mockSecrets := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "ns",
			Annotations: map[string]string{
				SecretAnnotation: "mysecret",
			},
		},
	}

	certPEM, keyPEM, _ := cert.GenerateSelfSignedCertKey("ns-mysecret", nil, []string{"svc.ns", "svc.ns.svc"})
	mockSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysecret",
			Namespace: "ns",
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       certPEM,
			corev1.TLSPrivateKeyKey: keyPEM,
		},
	}

	mockServices.EXPECT().
		EnqueueAfter("ns", "svc", gomock.Any()).
		AnyTimes()
	mockSecretsCache.EXPECT().
		Get("ns", "mysecret").
		Return(mockSecret, nil).AnyTimes()
	mockSecrets.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
			return secret, nil
		}).AnyTimes()

	h := &handler{
		services:     mockServices,
		secretsCache: mockSecretsCache,
		secrets:      mockSecrets,
	}

	const concurrency = 10
	done := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		go func() {
			_, err := h.generateSecret(service)
			assert.NoError(t, err)
			done <- struct{}{}
		}()
	}
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

func TestHandler_GenerateSecret_Race_MultiService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
	mockSecretsCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
	mockSecrets := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)

	const concurrency = 10
	done := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		serviceName := fmt.Sprintf("svc-%d", i)
		secretName := fmt.Sprintf("secret-%d", i)
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: "ns",
				Annotations: map[string]string{
					SecretAnnotation: secretName,
				},
			},
		}
		certPEM, keyPEM, _ := cert.GenerateSelfSignedCertKey("ns-"+secretName, nil, []string{serviceName + ".ns", serviceName + ".ns.svc"})
		mockSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: "ns",
			},
			Type: corev1.SecretTypeTLS,
			Data: map[string][]byte{
				corev1.TLSCertKey:       certPEM,
				corev1.TLSPrivateKeyKey: keyPEM,
			},
		}

		mockServices.EXPECT().
			EnqueueAfter("ns", gomock.Any(), gomock.Any()).
			AnyTimes()
		mockSecretsCache.EXPECT().
			Get("ns", secretName).
			Return(mockSecret, nil).AnyTimes()
		mockSecrets.EXPECT().
			Update(gomock.Any()).
			DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
				return secret, nil
			}).AnyTimes()

		go func(svc *corev1.Service) {
			h := &handler{
				services:     mockServices,
				secretsCache: mockSecretsCache,
				secrets:      mockSecrets,
			}
			_, err := h.generateSecret(svc)
			assert.NoError(t, err)
			done <- struct{}{}
		}(service)
	}

	for i := 0; i < concurrency; i++ {
		<-done
	}
}

func TestHandler_Race_Stress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockServiceCache := fake.NewMockCacheInterface[*corev1.Service](ctrl)
	mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
	mockSecretsCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
	mockSecrets := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockMutatingWebHooks := fake.NewMockNonNamespacedControllerInterface[*adminregv1.MutatingWebhookConfiguration, *adminregv1.MutatingWebhookConfigurationList](ctrl)
	mockValidatingWebHooks := fake.NewMockNonNamespacedControllerInterface[*adminregv1.ValidatingWebhookConfiguration, *adminregv1.ValidatingWebhookConfigurationList](ctrl)
	mockCRDs := fake.NewMockNonNamespacedControllerInterface[*apiextv1.CustomResourceDefinition, *apiextv1.CustomResourceDefinitionList](ctrl)

	mockMutatingCache := fake.NewMockNonNamespacedCacheInterface[*adminregv1.MutatingWebhookConfiguration](ctrl)
	mockValidatingCache := fake.NewMockNonNamespacedCacheInterface[*adminregv1.ValidatingWebhookConfiguration](ctrl)
	mockCRDsCache := fake.NewMockNonNamespacedCacheInterface[*apiextv1.CustomResourceDefinition](ctrl)

	mockMutatingWebHooks.EXPECT().Cache().Return(mockMutatingCache).AnyTimes()
	mockValidatingWebHooks.EXPECT().Cache().Return(mockValidatingCache).AnyTimes()
	mockCRDs.EXPECT().Cache().Return(mockCRDsCache).AnyTimes()

	mockMutatingCache.EXPECT().GetByIndex(gomock.Any(), gomock.Any()).Return([]*adminregv1.MutatingWebhookConfiguration{}, nil).AnyTimes()
	mockValidatingCache.EXPECT().GetByIndex(gomock.Any(), gomock.Any()).Return([]*adminregv1.ValidatingWebhookConfiguration{}, nil).AnyTimes()
	mockCRDsCache.EXPECT().GetByIndex(gomock.Any(), gomock.Any()).Return([]*apiextv1.CustomResourceDefinition{}, nil).AnyTimes()

	mockMutatingWebHooks.EXPECT().Enqueue(gomock.Any()).AnyTimes()
	mockValidatingWebHooks.EXPECT().Enqueue(gomock.Any()).AnyTimes()
	mockCRDs.EXPECT().Enqueue(gomock.Any()).AnyTimes()

	mockSecrets.EXPECT().Update(gomock.Any()).DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
		return secret, nil
	}).AnyTimes()
	mockSecrets.EXPECT().Create(gomock.Any()).DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
		return secret, nil
	}).AnyTimes()
	mockCRDs.EXPECT().Update(gomock.Any()).DoAndReturn(func(crd *apiextv1.CustomResourceDefinition) (*apiextv1.CustomResourceDefinition, error) {
		return crd, nil
	}).AnyTimes()

	h := &handler{
		services:           mockServices,
		serviceCache:       mockServiceCache,
		secretsCache:       mockSecretsCache,
		secrets:            mockSecrets,
		mutatingWebHooks:   mockMutatingWebHooks,
		validatingWebHooks: mockValidatingWebHooks,
		crds:               mockCRDs,
	}

	const concurrency = 10
	done := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		go func(i int) {
			serviceName := fmt.Sprintf("svc-%d", i%5)
			secretName := fmt.Sprintf("secret-%d", i%5)
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: "ns",
					Annotations: map[string]string{
						SecretAnnotation: secretName,
					},
				},
			}
			mockServices.EXPECT().
				EnqueueAfter("ns", gomock.Any(), gomock.Any()).
				AnyTimes()
			mockServiceCache.EXPECT().
				Get("ns", serviceName).
				Return(service, nil).AnyTimes()
			certPEM, keyPEM, _ := cert.GenerateSelfSignedCertKey("ns-"+secretName, nil, []string{serviceName + ".ns", serviceName + ".ns.svc"})
			mockSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "ns",
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					corev1.TLSCertKey:       certPEM,
					corev1.TLSPrivateKeyKey: keyPEM,
				},
			}
			mockSecretsCache.EXPECT().
				Get("ns", secretName).
				Return(mockSecret, nil).AnyTimes()

			switch i % 3 {
			case 0:
				_, err := h.generateSecret(service)
				assert.NoError(t, err)
			case 1:
				_, err := h.OnService("ns/"+serviceName, service)
				assert.NoError(t, err)
			case 2:
				crd := &apiextv1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: "crd-" + serviceName,
					},
					Spec: apiextv1.CustomResourceDefinitionSpec{
						Conversion: &apiextv1.CustomResourceConversion{
							Strategy: apiextv1.WebhookConverter,
							Webhook: &apiextv1.WebhookConversion{
								ClientConfig: &apiextv1.WebhookClientConfig{
									Service: &apiextv1.ServiceReference{
										Namespace: "ns",
										Name:      serviceName,
									},
									CABundle: []byte{},
								},
							},
						},
					},
				}
				_, err := h.OnCRDChange("key", crd)
				assert.NoError(t, err)
			}
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

func TestHandler_ParseCert_CorruptedData(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "badsecret",
			Namespace: "ns",
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       []byte("-----BEGIN CERTIFICATE-----\nMIIB fake cert\n-----END CERTIFICATE-----"),
			corev1.TLSPrivateKeyKey: []byte("not-a-key"),
		},
	}

	parsed, err := parseCert(secret)
	assert.Error(t, err, "expected error when parsing corrupted TLS secret")
	assert.Nil(t, parsed, "no updated secret should be returned on corrupted data")
}

func TestHandler_GenerateSecret_Race_SharedSecret(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
	mockSecretsCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
	mockSecrets := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)

	// Both services point to the same secret name
	service1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc1",
			Namespace: "ns",
			Annotations: map[string]string{
				SecretAnnotation: "shared-secret",
			},
		},
	}
	service2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc2",
			Namespace: "ns",
			Annotations: map[string]string{
				SecretAnnotation: "shared-secret",
			},
		},
	}

	// Intentionally returning notfound from the cache each time so that
	// multiple goroutines will attempt to create the same secret concurrently.
	mockServices.EXPECT().
		EnqueueAfter("ns", gomock.Any(), gomock.Any()).
		AnyTimes()
	mockSecretsCache.EXPECT().
		Get("ns", "shared-secret").
		Return(nil, apierror.NewNotFound(corev1.Resource("secrets"), "shared-secret")).
		AnyTimes()
	mockSecrets.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
			return secret, nil
		}).AnyTimes()
	mockSecrets.EXPECT().
		Update(gomock.Any()).
		DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
			return secret, nil
		}).AnyTimes()

	h := &handler{
		services:     mockServices,
		secretsCache: mockSecretsCache,
		secrets:      mockSecrets,
	}

	const concurrency = 10
	done := make(chan struct{})
	for i := 0; i < concurrency; i++ {
		go func(i int) {
			var svc *corev1.Service
			if i%2 == 0 {
				svc = service1
			} else {
				svc = service2
			}
			_, err := h.generateSecret(svc)
			assert.NoError(t, err)
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < concurrency; i++ {
		<-done
	}
}

func TestHandler_GenerateSecret_StaleCacheAlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
	mockSecretsCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
	mockSecrets := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "ns",
			Annotations: map[string]string{
				SecretAnnotation: "mysecret",
			},
		},
	}

	// Simulate cache always lags and reports NotFound
	mockServices.EXPECT().
		EnqueueAfter("ns", "svc", gomock.Any()).
		AnyTimes()
	mockSecretsCache.EXPECT().
		Get("ns", "mysecret").
		Return(nil, apierror.NewNotFound(corev1.Resource("secrets"), "mysecret")).
		AnyTimes()

	mockSecrets.EXPECT().
		Create(gomock.Any()).
		DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
			return nil, apierror.NewAlreadyExists(corev1.Resource("secrets"), "mysecret")
		}).
		AnyTimes()

	certPEM, keyPEM, err := cert.GenerateSelfSignedCertKey("mysecret", nil, []string{"svc.ns"})
	assert.NoError(t, err)

	expectedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysecret",
			Namespace: "ns",
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       certPEM,
			corev1.TLSPrivateKeyKey: keyPEM,
		},
	}
	mockSecrets.EXPECT().
		Get("ns", "mysecret", gomock.Any()).
		Return(expectedSecret, nil).
		AnyTimes()
	mockSecrets.EXPECT().
		Update(gomock.Any()).
		Return(expectedSecret, nil).
		AnyTimes()

	h := &handler{
		services:     mockServices,
		secretsCache: mockSecretsCache,
		secrets:      mockSecrets,
	}

	secret, err := h.generateSecret(service)

	assert.NoError(t, err)
	assert.NotNil(t, secret)
}

func TestHandler_OnSecretChange_Then_OnService_UpdatesCABundle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	//Mocks
	mockServiceController := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
	mockServiceCache := fake.NewMockCacheInterface[*corev1.Service](ctrl)
	mockSecretsCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
	mockSecrets := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockMutatingWebHooks := fake.NewMockNonNamespacedControllerInterface[*adminregv1.MutatingWebhookConfiguration, *adminregv1.MutatingWebhookConfigurationList](ctrl)
	mockValidatingWebHooks := fake.NewMockNonNamespacedControllerInterface[*adminregv1.ValidatingWebhookConfiguration, *adminregv1.ValidatingWebhookConfigurationList](ctrl)
	mockCRDs := fake.NewMockNonNamespacedControllerInterface[*apiextv1.CustomResourceDefinition, *apiextv1.CustomResourceDefinitionList](ctrl)
	mockCRDCache := fake.NewMockNonNamespacedCacheInterface[*apiextv1.CustomResourceDefinition](ctrl)
	mockMutatingWebHooksCache := fake.NewMockNonNamespacedCacheInterface[*adminregv1.MutatingWebhookConfiguration](ctrl)
	mockValidatingWebHooksCache := fake.NewMockNonNamespacedCacheInterface[*adminregv1.ValidatingWebhookConfiguration](ctrl)

	// Generate self-signed cert
	certPEM, keyPEM, err := cert.GenerateSelfSignedCertKey("svc-mysecret", nil, []string{"svc.ns", "svc.ns.svc", "svc.ns.svc.cluster.local"})
	require.NoError(t, err)

	// Objects already created
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysecret",
			Namespace: "ns",
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       certPEM,
			corev1.TLSPrivateKeyKey: keyPEM,
		},
	}

	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc",
			Namespace: "ns",
			Annotations: map[string]string{
				SecretAnnotation: "mysecret",
			},
		},
	}

	serviceList := &corev1.ServiceList{Items: []corev1.Service{service}}

	webhook := &adminregv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "webhook"},
		Webhooks: []adminregv1.MutatingWebhook{
			{
				Name: "wh",
				ClientConfig: adminregv1.WebhookClientConfig{
					Service: &adminregv1.ServiceReference{
						Namespace: "ns",
						Name:      "svc",
					},
					CABundle: []byte(""), // empty → should trigger Update
				},
			},
		},
	}

	webhookList := []*adminregv1.MutatingWebhookConfiguration{webhook}

	// Expected mock calls
	mockServiceController.EXPECT().List("ns", gomock.Any()).Return(serviceList, nil).Times(1)
	mockServiceController.EXPECT().Enqueue("ns", "svc").Times(1)

	mockSecretsCache.EXPECT().Get("ns", "mysecret").Return(secret, nil).Times(2)

	mockMutatingWebHooks.EXPECT().Cache().Return(mockMutatingWebHooksCache).AnyTimes()
	mockMutatingWebHooksCache.EXPECT().GetByIndex(byServiceIndex, "ns/svc").Return(webhookList, nil).Times(1)
	mockMutatingWebHooks.EXPECT().Enqueue("webhook").Times(1)
	mockMutatingWebHooks.EXPECT().Update(gomock.Any()).DoAndReturn(func(updated *adminregv1.MutatingWebhookConfiguration) (*adminregv1.MutatingWebhookConfiguration, error) {
		for _, wh := range updated.Webhooks {
			assert.NotEmpty(t, wh.ClientConfig.CABundle, "CABundle should be updated")
		}
		return updated, nil
	}).Times(1)

	mockValidatingWebHooks.EXPECT().Cache().Return(mockValidatingWebHooksCache).AnyTimes()
	mockValidatingWebHooksCache.EXPECT().GetByIndex(byServiceIndex, "ns/svc").Return([]*adminregv1.ValidatingWebhookConfiguration{}, nil).AnyTimes()

	mockCRDs.EXPECT().Cache().Return(mockCRDCache).Times(1)
	mockCRDCache.EXPECT().GetByIndex(byServiceIndex, "ns/svc").Return([]*apiextv1.CustomResourceDefinition{}, nil).Times(1)

	mockServiceController.EXPECT().
		EnqueueAfter("ns", gomock.Any(), gomock.Any()).
		AnyTimes()
	mockServiceCache.EXPECT().
		Get("ns", "svc").
		Return(&service, nil).
		Times(1)

	h := &handler{
		services:           mockServiceController,
		serviceCache:       mockServiceCache,
		secrets:            mockSecrets,
		secretsCache:       mockSecretsCache,
		mutatingWebHooks:   mockMutatingWebHooks,
		validatingWebHooks: mockValidatingWebHooks,
		crds:               mockCRDs,
	}

	// Run OnSecretChange ---
	gotSecret, err := h.OnSecretChange("ns/mysecret", secret)
	assert.NoError(t, err)
	assert.Equal(t, secret, gotSecret)

	// Run OnService triggered by OnSecretChange
	gotService, err := h.OnService("ns/svc", &service)
	assert.NoError(t, err)
	assert.Equal(t, &service, gotService)

	// Run OnMutationWebhookChange triggered by OnService
	updatedWebhook, err := h.OnMutationWebhookChange("webhook", webhook)
	assert.NoError(t, err)
	for _, wh := range updatedWebhook.Webhooks {
		assert.NotEmpty(t, wh.ClientConfig.CABundle, "CABundle should be updated")
	}
}

func TestHandler_scheduleNextCertCheck(t *testing.T) {
	type enqueueCall struct {
		ns, name string
		delay    time.Duration
	}
	var calls []enqueueCall

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
	mockServices.EXPECT().
		EnqueueAfter(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(ns, name string, delay time.Duration) {
			calls = append(calls, enqueueCall{ns, name, delay})
		}).
		Times(2)

	h := &handler{services: mockServices}

	tests := []struct {
		name      string
		maxAge    time.Duration
		wantDelay time.Duration
		wantErr   bool
	}{
		{
			name:      "expiring in 30 days",
			maxAge:    30 * 24 * time.Hour,
			wantDelay: 29*24*time.Hour - 1*time.Hour,
			wantErr:   false,
		},
		{
			name:      "expiring in 12 hours (less than renewBefore)",
			maxAge:    12 * time.Hour,
			wantDelay: time.Minute,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls = nil

			certPEM, keyPEM, err := cert.GenerateSelfSignedCertKeyWithOptions(cert.SelfSignedCertKeyOptions{
				Host:   "ns-mysecret",
				MaxAge: tt.maxAge,
			})
			require.NoError(t, err)

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysecret",
					Namespace: "ns",
				},
				Type: corev1.SecretTypeTLS,
				Data: map[string][]byte{
					corev1.TLSCertKey:       certPEM,
					corev1.TLSPrivateKeyKey: keyPEM,
				},
			}

			obj := &corev1.Service{}
			err = h.scheduleNextCertCheck(obj, secret)

			if (err != nil) != tt.wantErr {
				t.Fatalf("scheduleNextCertCheck() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(calls) != 1 {
				t.Fatalf("expected 1 EnqueueAfter call, got %d", len(calls))
			}

			got := calls[0].delay
			tolerance := time.Second * 10
			if math.Abs(got.Seconds()-tt.wantDelay.Seconds()) > tolerance.Seconds() {
				t.Errorf("expected delay ~%v, got %v", tt.wantDelay, got)
			}
		})
	}
}
