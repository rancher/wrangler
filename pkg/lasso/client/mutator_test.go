// Mocks for this test are generated with the following command.
//go:generate mockgen --build_flags=--mod=mod -package client -destination ./mocks_test.go github.com/rancher/lasso/pkg/client SharedClientFactory

package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
)

var (
	validGVK = schema.GroupVersionKind{
		Group:   "core",
		Version: "v1",
		Kind:    "Secret",
	}
	validGVR = schema.GroupVersionResource{
		Group:    validGVK.Group,
		Version:  validGVK.Version,
		Resource: "secrets",
	}
)

type userAgentRoundTripper struct {
	expectedUserAgent string
	t                 *testing.T
}

func (u userAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	assert.Equal(u.t, u.expectedUserAgent, req.Header.Get("User-Agent"))
	return &http.Response{}, nil
}

type fakeRoundTripper struct {
	lastRequest *http.Request
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	f.lastRequest = req
	return &http.Response{}, nil
}

func TestNewSharedClientFactoryWithAgent(t *testing.T) {
	mockSharedClientFactory := NewMockSharedClientFactory(gomock.NewController(t))

	testUserAgent := "testagent"
	testConfig := rest.Config{
		UserAgent: "defaultuseragent",
		ContentConfig: rest.ContentConfig{
			NegotiatedSerializer: serializer.WithoutConversionCodecFactory{},
		},
		Transport: userAgentRoundTripper{t: t, expectedUserAgent: testUserAgent},
	}

	testRestClient, err := rest.UnversionedRESTClientFor(&testConfig)
	assert.Nil(t, err)
	testClient := &Client{
		RESTClient: testRestClient,
		Config:     testConfig,
	}
	mockSharedClientFactory.EXPECT().ForKind(gomock.Any()).DoAndReturn(func(gvk schema.GroupVersionKind) (*Client, error) {
		return testClient, nil
	}).AnyTimes()
	mockSharedClientFactory.EXPECT().ForResource(gomock.Any(), gomock.Any()).DoAndReturn(func(gvr schema.GroupVersionResource, namespaced bool) (*Client, error) {
		return testClient, nil
	}).AnyTimes()
	mockSharedClientFactory.EXPECT().ForResourceKind(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(gvr schema.GroupVersionResource, kind string, namespaced bool) *Client {
		return testClient
	}).AnyTimes()
	testSharedClientFactoryWithAgent := NewSharedClientFactoryWithAgent("testagent", mockSharedClientFactory)

	// for kind
	clientWithAgent, err := testSharedClientFactoryWithAgent.ForKind(schema.GroupVersionKind{})
	assert.Nil(t, err)

	restClient := clientWithAgent.RESTClient.(*rest.RESTClient)
	restClient.Client.Transport.RoundTrip(&http.Request{})

	// for resource
	clientWithAgent, err = testSharedClientFactoryWithAgent.ForResource(schema.GroupVersionResource{}, false)
	assert.Nil(t, err)

	restClient = clientWithAgent.RESTClient.(*rest.RESTClient)
	restClient.Client.Transport.RoundTrip(&http.Request{})

	// for resource kind
	clientWithAgent = testSharedClientFactoryWithAgent.ForResourceKind(schema.GroupVersionResource{}, "", false)
	assert.Nil(t, err)

	restClient = clientWithAgent.RESTClient.(*rest.RESTClient)
	restClient.Client.Transport.RoundTrip(&http.Request{})

	// test with errors
	mockSharedClientErrFactory := NewMockSharedClientFactory(gomock.NewController(t))
	mockSharedClientErrFactory.EXPECT().ForKind(gomock.Any()).DoAndReturn(func(gvk schema.GroupVersionKind) (*Client, error) {
		return nil, fmt.Errorf("some error")
	}).AnyTimes()
	mockSharedClientErrFactory.EXPECT().ForResource(gomock.Any(), gomock.Any()).DoAndReturn(func(gvr schema.GroupVersionResource, namespaced bool) (*Client, error) {
		return nil, fmt.Errorf("some error")
	}).AnyTimes()
	mockSharedClientErrFactory.EXPECT().ForResourceKind(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(gvr schema.GroupVersionResource, kind string, namespaced bool) *Client {
		return nil
	}).AnyTimes()
	testSharedClientFactoryWithAgent = NewSharedClientFactoryWithAgent("", mockSharedClientErrFactory)

	// for kind
	clientWithAgent, err = testSharedClientFactoryWithAgent.ForKind(schema.GroupVersionKind{})
	assert.NotNil(t, err)
	assert.Nil(t, clientWithAgent)

	// for resource
	clientWithAgent, err = testSharedClientFactoryWithAgent.ForResource(schema.GroupVersionResource{}, false)
	assert.NotNil(t, err)
	assert.Nil(t, clientWithAgent)

	// for resource kind
	clientWithAgent = testSharedClientFactoryWithAgent.ForResourceKind(schema.GroupVersionResource{}, "", false)
	assert.Nil(t, clientWithAgent)
}

func TestNewSharedClientFactoryWithImpersonation(t *testing.T) {

	expectedError := fmt.Errorf("expected test error")
	testRT := fakeRoundTripper{}

	// Create Test client that will be mutated.
	testConfig := rest.Config{
		ContentConfig: rest.ContentConfig{
			NegotiatedSerializer: serializer.WithoutConversionCodecFactory{},
		},
		Transport: &testRT,
	}
	testRestClient, err := rest.UnversionedRESTClientFor(&testConfig)
	require.NoError(t, err)
	testClient := &Client{
		RESTClient: testRestClient,
		Config:     testConfig,
	}

	mockSharedClientFactory := NewMockSharedClientFactory(gomock.NewController(t))

	// return testClient for all valid request
	mockSharedClientFactory.EXPECT().ForKind(validGVK).Return(testClient, nil).AnyTimes()
	mockSharedClientFactory.EXPECT().ForResource(validGVR, gomock.Any()).Return(testClient, nil).AnyTimes()
	mockSharedClientFactory.EXPECT().ForResourceKind(validGVR, validGVK.Kind, gomock.Any()).Return(testClient).AnyTimes()

	// return error for all unknown gvr request
	mockSharedClientFactory.EXPECT().ForKind(gomock.Any()).Return(nil, expectedError).AnyTimes()
	mockSharedClientFactory.EXPECT().ForResource(gomock.Any(), gomock.Any()).Return(nil, expectedError).AnyTimes()
	mockSharedClientFactory.EXPECT().ForResourceKind(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	tests := []struct {
		name        string
		impersonate rest.ImpersonationConfig
		gvr         schema.GroupVersionResource
		gvk         schema.GroupVersionKind
	}{
		{
			name: "no impersonation set",
			gvr:  validGVR,
			gvk:  validGVK,
		},
		{
			name:        "impersonate user name",
			impersonate: rest.ImpersonationConfig{UserName: "Zeus"},
			gvr:         validGVR,
			gvk:         validGVK,
		},
		{
			name:        "impersonate UID",
			impersonate: rest.ImpersonationConfig{UID: "z-1231qqwe-sdfas-dfs"},
			gvr:         validGVR,
			gvk:         validGVK,
		},
		{
			name: "impersonate user name and Groups",
			impersonate: rest.ImpersonationConfig{
				UserName: "Zeus",
				Groups:   []string{"Greek", "Gods", "Elemental"},
			},
			gvr: validGVR,
			gvk: validGVK,
		},
		{
			name: "impersonate user name and Groups, and extra",
			impersonate: rest.ImpersonationConfig{
				UserName: "Zeus",
				Groups:   []string{"Greek", "Gods", "Elemental"},
				Extra:    map[string][]string{"Parents": {"Cronus", "Rhea"}},
			},
			gvr: validGVR,
			gvk: validGVK,
		},
		{
			name: "impersonate Groups only",
			impersonate: rest.ImpersonationConfig{
				Groups: []string{"Greek", "Gods", "Elemental"},
			},
			gvr: validGVR,
			gvk: validGVK,
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(tt.name, func(t *testing.T) {
			factory := NewSharedClientFactoryWithImpersonation(test.impersonate, mockSharedClientFactory)

			client, err := factory.ForKind(test.gvk)
			require.NoError(t, err)
			require.Equal(t, client.Config.Impersonate, tt.impersonate, "ForKind client was not correctly mutated")
			// run a request and verify the test roundtripper got the correct impersonate information
			_ = client.RESTClient.Get().Do(context.TODO())
			require.Equal(t, tt.impersonate, getImpersonateFromRequest(testRT.lastRequest), "ForKind client was not correctly mutated")

			client, _ = factory.ForResource(test.gvr, false)
			require.NoError(t, err)
			require.Equal(t, client.Config.Impersonate, tt.impersonate, "ForResource client was not correctly mutated")
			// run a request and verify the test roundtripper got the correct impersonate information
			_ = client.RESTClient.Get().Do(context.TODO())
			require.Equal(t, tt.impersonate, getImpersonateFromRequest(testRT.lastRequest), "ForResource client was not correctly mutated")

			client = factory.ForResourceKind(test.gvr, test.gvk.Kind, false)
			require.NotNil(t, client, "client was not set")
			require.Equal(t, client.Config.Impersonate, tt.impersonate, "ForResourceKind client was not correctly mutated")
			// run a request and verify the test roundtripper got the correct impersonate information
			_ = client.RESTClient.Get().Do(context.TODO())
			require.Equal(t, tt.impersonate, getImpersonateFromRequest(testRT.lastRequest), "ForResourceKind client was not correctly mutated")
		})
	}
}

func Test_SharedClientFactoryWithMutation(t *testing.T) {
	originalClient := Client{kind: "Original"}
	testClient := originalClient
	mutatedClient := &Client{kind: "MUTATED"}
	expectedError := errors.New("expected Error")

	mockSharedClientFactory := NewMockSharedClientFactory(gomock.NewController(t))
	var retErr bool
	newMutator := func(_ *Client) (*Client, error) {
		if retErr {
			return nil, expectedError
		}
		return mutatedClient, nil
	}
	factory := sharedClientFactoryWithMutation{
		SharedClientFactory: mockSharedClientFactory,
		mutator:             newMutator,
	}

	mockSharedClientFactory.EXPECT().ForKind(validGVK).Return(&testClient, nil)
	newClient, err := factory.ForKind(validGVK)
	require.NoError(t, err, "ForKind failed to get new mutated client.")
	require.Equal(t, mutatedClient, newClient, "ForKind did not return the mutated client.")
	require.True(t, reflect.DeepEqual(originalClient, testClient), "ForKind mutated the original client.")

	mockSharedClientFactory.EXPECT().ForResource(validGVR, true).Return(&testClient, nil)
	newClient, err = factory.ForResource(validGVR, true)
	require.NoError(t, err, "ForResource failed to get new mutated client.")
	require.Equal(t, mutatedClient, newClient, "ForResource did not return the mutated client.")
	require.True(t, reflect.DeepEqual(originalClient, testClient), "ForResource mutated the original client.")

	mockSharedClientFactory.EXPECT().ForResourceKind(validGVR, validGVK.Kind, true).Return(&testClient)
	newClient = factory.ForResourceKind(validGVR, validGVK.Kind, true)
	require.Equal(t, mutatedClient, newClient, "ForResourceKind did not return the mutated client.")
	require.True(t, reflect.DeepEqual(originalClient, testClient), "ForResourceKind mutated the original client.")

	// base factory errors
	mockSharedClientFactory.EXPECT().ForKind(validGVK).Return(nil, expectedError)
	newClient, err = factory.ForKind(validGVK)
	require.Error(t, err, "ForKind should have failed to get client from factory.")
	require.Nil(t, newClient, "ForKind should not set client on error.")

	mockSharedClientFactory.EXPECT().ForResource(validGVR, true).Return(nil, expectedError)
	newClient, err = factory.ForResource(validGVR, true)
	require.Error(t, err, "ForResource should have failed to get client from factory.")
	require.Nil(t, newClient, "ForResource should not set client on error.")

	mockSharedClientFactory.EXPECT().ForResourceKind(validGVR, validGVK.Kind, true).Return(nil)
	newClient = factory.ForResourceKind(validGVR, validGVK.Kind, true)
	require.Nil(t, newClient, "ForResourceKind should return a nil client when mutation fails.")

	// mutation errors
	retErr = true
	mockSharedClientFactory.EXPECT().ForKind(validGVK).Return(&testClient, nil)
	newClient, err = factory.ForKind(validGVK)
	require.Error(t, err, "ForKind should have failed mutation.")
	require.Nil(t, newClient, "ForKind should not set client on error.")

	mockSharedClientFactory.EXPECT().ForResource(validGVR, true).Return(&testClient, nil)
	newClient, err = factory.ForResource(validGVR, true)
	require.Error(t, err, "ForResource should have failed mutation.")
	require.Nil(t, newClient, "ForResource should not set client on error.")

	mockSharedClientFactory.EXPECT().ForResourceKind(validGVR, validGVK.Kind, true).Return(&testClient)
	newClient = factory.ForResourceKind(validGVR, validGVK.Kind, true)
	require.Nil(t, newClient, "ForResourceKind should return a nil client when mutation fails.")
}

func getImpersonateFromRequest(req *http.Request) rest.ImpersonationConfig {
	gotImpersonate := rest.ImpersonationConfig{Extra: map[string][]string{}}
	gotImpersonate.UserName = req.Header.Get(transport.ImpersonateUserHeader)
	gotImpersonate.UID = req.Header.Get(transport.ImpersonateUIDHeader)
	gotImpersonate.Groups = req.Header.Values(transport.ImpersonateGroupHeader)
	for k, v := range req.Header {
		if !strings.HasPrefix(k, transport.ImpersonateUserExtraHeaderPrefix) {
			continue
		}
		gotImpersonate.Extra[k[len(transport.ImpersonateUserExtraHeaderPrefix):]] = v
	}
	// empty map break comparison so set it to nil if empty
	if len(gotImpersonate.Extra) == 0 {
		gotImpersonate.Extra = nil
	}
	return gotImpersonate
}
