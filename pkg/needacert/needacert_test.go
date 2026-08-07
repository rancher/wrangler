package needacert

import (
	"bytes"
	"encoding/pem"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/rancher/wrangler/v4/pkg/generic/fake"
	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/assert"
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
			h := &handler{}
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

			existingCert, err := cert.ParseCertsPEM(certPEM)
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

			updated, err := h.updateSecret(service, secret, dnsNames, existingCert[0])
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

			mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
			mockServiceCache := fake.NewMockCacheInterface[*corev1.Service](ctrl)
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

			mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
			mockServiceCache := fake.NewMockCacheInterface[*corev1.Service](ctrl)
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

			mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
			mockServiceCache := fake.NewMockCacheInterface[*corev1.Service](ctrl)
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

	mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
	mockServiceCache := fake.NewMockCacheInterface[*corev1.Service](ctrl)
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

func TestCAOnly(t *testing.T) {
	fullChain, _, err := cert.GenerateSelfSignedCertKey("ns-mysecret", nil, []string{"svc.ns", "svc.ns.svc"})
	assert.NoError(t, err)

	fullChainCerts, err := cert.ParseCertsPEM(fullChain)
	assert.NoError(t, err)
	assert.Len(t, fullChainCerts, 2, "GenerateSelfSignedCertKey is expected to return leaf+CA")

	caOnlyPEM, err := caOnly(fullChain)
	assert.NoError(t, err)
	assert.Less(t, len(caOnlyPEM), len(fullChain), "CA-only bundle should be smaller than the full chain")

	caOnlyCerts, err := cert.ParseCertsPEM(caOnlyPEM)
	assert.NoError(t, err)
	assert.Len(t, caOnlyCerts, 1)
	assert.True(t, caOnlyCerts[0].IsCA)
	assert.Equal(t, fullChainCerts[len(fullChainCerts)-1].Raw, caOnlyCerts[0].Raw)
}

func TestCAOnly_SingleCert(t *testing.T) {
	fullChain, _, err := cert.GenerateSelfSignedCertKey("ns-mysecret", nil, []string{"svc.ns", "svc.ns.svc"})
	assert.NoError(t, err)

	// Strip the chain down to just the leaf cert, so caOnly sees a single-cert
	// PEM blob (as if the CA had already been dropped, or a bundle was never
	// chained in the first place).
	leafBlock, _ := pem.Decode(fullChain)
	assert.NotNil(t, leafBlock)
	leafOnlyPEM := pem.EncodeToMemory(leafBlock)

	result, err := caOnly(leafOnlyPEM)
	assert.NoError(t, err)
	assert.Equal(t, leafOnlyPEM, result, "single-cert chain should be returned unmodified, not re-encoded")
}

func TestCAOnly_InvalidPEM(t *testing.T) {
	_, err := caOnly([]byte("not a cert"))
	assert.Error(t, err)
}

func TestHandler_CABundleFor(t *testing.T) {
	fullChain, _, err := cert.GenerateSelfSignedCertKey("ns-mysecret", nil, []string{"svc.ns", "svc.ns.svc"})
	assert.NoError(t, err)
	secret := &corev1.Secret{
		Data: map[string][]byte{
			corev1.TLSCertKey: fullChain,
		},
	}

	t.Run("nil service returns full chain", func(t *testing.T) {
		bundle, err := caBundleFor(nil, secret)
		assert.NoError(t, err)
		assert.Equal(t, fullChain, bundle)
	})

	t.Run("no annotation returns full chain", func(t *testing.T) {
		svc := &corev1.Service{}
		bundle, err := caBundleFor(svc, secret)
		assert.NoError(t, err)
		assert.Equal(t, fullChain, bundle)
	})

	t.Run("unrecognized annotation value returns full chain", func(t *testing.T) {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{CABundleModeAnnotation: "bogus"},
			},
		}
		bundle, err := caBundleFor(svc, secret)
		assert.NoError(t, err)
		assert.Equal(t, fullChain, bundle)
	})

	t.Run("ca-only annotation returns just the CA cert", func(t *testing.T) {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{CABundleModeAnnotation: CABundleModeCAOnly},
			},
		}
		bundle, err := caBundleFor(svc, secret)
		assert.NoError(t, err)
		assert.NotEqual(t, fullChain, bundle)
		assert.Less(t, len(bundle), len(fullChain))

		certs, err := cert.ParseCertsPEM(bundle)
		assert.NoError(t, err)
		assert.Len(t, certs, 1)
		assert.True(t, certs[0].IsCA)
	})

	t.Run("full-chain annotation explicitly keeps full chain", func(t *testing.T) {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{CABundleModeAnnotation: CABundleModeFullChain},
			},
		}
		bundle, err := caBundleFor(svc, secret)
		assert.NoError(t, err)
		assert.Equal(t, fullChain, bundle)
	})
}

func TestHandler_OnMutationWebhookChange_CAOnlyAnnotation(t *testing.T) {
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
				SecretAnnotation:       "mysecret",
				CABundleModeAnnotation: CABundleModeCAOnly,
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
	bundle := updated.Webhooks[0].ClientConfig.CABundle
	assert.NotEmpty(t, bundle)
	assert.Less(t, len(bundle), len(certPEM))

	certs, err := cert.ParseCertsPEM(bundle)
	assert.NoError(t, err)
	assert.Len(t, certs, 1)
	assert.True(t, certs[0].IsCA)
}

// TestHandler_OnMutationWebhookChange_PerServiceAnnotationScoping proves that
// CABundleModeAnnotation lets one Service opt into CA-only CABundles without
// affecting a sibling Service on the same handler that hasn't opted in - the
// scenario needed so only a specific consumer (e.g. a CAPI provider) is
// affected, not every webhook sharing the same needacert controller.
func TestHandler_OnMutationWebhookChange_PerServiceAnnotationScoping(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockServiceCache := fake.NewMockCacheInterface[*corev1.Service](ctrl)
	mockServices := fake.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
	mockSecretsCache := fake.NewMockCacheInterface[*corev1.Secret](ctrl)
	mockSecrets := fake.NewMockControllerInterface[*corev1.Secret, *corev1.SecretList](ctrl)
	mockMutatingWebHooks := fake.NewMockNonNamespacedControllerInterface[*adminregv1.MutatingWebhookConfiguration, *adminregv1.MutatingWebhookConfigurationList](ctrl)

	optedInService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "turtles-svc",
			Namespace: "ns",
			Annotations: map[string]string{
				SecretAnnotation:       "turtles-secret",
				CABundleModeAnnotation: "ca-only",
			},
		},
	}
	defaultService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-svc",
			Namespace: "ns",
			Annotations: map[string]string{
				SecretAnnotation: "other-secret",
			},
		},
	}

	turtlesCertPEM, turtlesKeyPEM, _ := cert.GenerateSelfSignedCertKey("ns-turtles-secret", nil, []string{"turtles-svc.ns"})
	turtlesSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "turtles-secret", Namespace: "ns"},
		Type:       corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       turtlesCertPEM,
			corev1.TLSPrivateKeyKey: turtlesKeyPEM,
		},
	}
	otherCertPEM, otherKeyPEM, _ := cert.GenerateSelfSignedCertKey("ns-other-secret", nil, []string{"other-svc.ns"})
	otherSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "other-secret", Namespace: "ns"},
		Type:       corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       otherCertPEM,
			corev1.TLSPrivateKeyKey: otherKeyPEM,
		},
	}

	mockServices.EXPECT().EnqueueAfter(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockServiceCache.EXPECT().Get("ns", "turtles-svc").Return(optedInService, nil)
	mockServiceCache.EXPECT().Get("ns", "other-svc").Return(defaultService, nil)
	mockSecretsCache.EXPECT().Get("ns", "turtles-secret").Return(turtlesSecret, nil)
	mockSecretsCache.EXPECT().Get("ns", "other-secret").Return(otherSecret, nil)
	mockSecrets.EXPECT().Update(gomock.Any()).DoAndReturn(func(secret *corev1.Secret) (*corev1.Secret, error) {
		return secret, nil
	}).Times(2)
	mockMutatingWebHooks.EXPECT().Update(gomock.Any()).DoAndReturn(func(webhook *adminregv1.MutatingWebhookConfiguration) (*adminregv1.MutatingWebhookConfiguration, error) {
		return webhook, nil
	})

	// Neither service opts in via anything but its own annotation - only
	// optedInService should get a CA-only bundle.
	h := &handler{
		services:         mockServices,
		serviceCache:     mockServiceCache,
		secretsCache:     mockSecretsCache,
		secrets:          mockSecrets,
		mutatingWebHooks: mockMutatingWebHooks,
	}

	webhook := &adminregv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: "webhook"},
		Webhooks: []adminregv1.MutatingWebhook{
			{
				Name: "turtles",
				ClientConfig: adminregv1.WebhookClientConfig{
					Service:  &adminregv1.ServiceReference{Namespace: "ns", Name: "turtles-svc"},
					CABundle: []byte{},
				},
			},
			{
				Name: "other",
				ClientConfig: adminregv1.WebhookClientConfig{
					Service:  &adminregv1.ServiceReference{Namespace: "ns", Name: "other-svc"},
					CABundle: []byte{},
				},
			},
		},
	}

	updated, err := h.OnMutationWebhookChange("key", webhook)
	assert.NoError(t, err)
	assert.NotNil(t, updated)

	turtlesBundle := updated.Webhooks[0].ClientConfig.CABundle
	otherBundle := updated.Webhooks[1].ClientConfig.CABundle

	turtlesCerts, err := cert.ParseCertsPEM(turtlesBundle)
	assert.NoError(t, err)
	assert.Len(t, turtlesCerts, 1, "opted-in service should get a CA-only bundle")
	assert.True(t, turtlesCerts[0].IsCA)

	otherCerts, err := cert.ParseCertsPEM(otherBundle)
	assert.NoError(t, err)
	assert.Len(t, otherCerts, 2, "service without the annotation should keep the full leaf+CA chain")
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
			name:      "cert expires in 90 days → schedule at 30 days",
			maxAge:    90 * 24 * time.Hour,
			wantDelay: 30 * 24 * time.Hour,
		},
		{
			name:      "cert expires in 30 days → clamp to 1 minute",
			maxAge:    30 * 24 * time.Hour,
			wantDelay: time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls = nil

			certPEM, keyPEM, err := cert.GenerateSelfSignedCertKeyWithOptions(cert.SelfSignedCertKeyOptions{
				Host:   "ns-mysecret",
				MaxAge: tt.maxAge,
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

			obj := &corev1.Service{}
			err = h.scheduleNextCertCheck(obj, secret)

			if (err != nil) != tt.wantErr {
				t.Fatalf("scheduleNextCertCheck() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(calls) != 1 {
				t.Fatalf("expected 1 EnqueueAfter call, got %d", len(calls))
			}

			got := calls[0].delay
			tolerance := 1*time.Hour + 10*time.Second
			if math.Abs(got.Seconds()-tt.wantDelay.Seconds()) > tolerance.Seconds() {
				t.Errorf("expected delay ~%v, got %v", tt.wantDelay, got)
			}
		})
	}
}
