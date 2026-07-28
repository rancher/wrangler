package controller

//go:generate mockgen --build_flags=--mod=mod -package controller -destination ./mocks_test.go github.com/rancher/lasso/pkg/controller SharedController

import (
	"net/http"
	"testing"

	"github.com/rancher/lasso/pkg/client"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
)

func TestNewSharedControllerFactoryWithAgent(t *testing.T) {
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

	testSharedController := NewMockSharedController(gomock.NewController(t))
	testClient := &client.Client{
		RESTClient: testRestClient,
		Config:     testConfig,
	}
	testSharedController.EXPECT().Client().DoAndReturn(func() *client.Client {
		return testClient
	}).AnyTimes()

	testSharedControllerWithAgent := NewSharedControllerWithAgent(testUserAgent, testSharedController)
	testClientWithAgent := testSharedControllerWithAgent.Client().RESTClient.(*rest.RESTClient)
	testClientWithAgent.Client.Transport.RoundTrip(&http.Request{})

	testErrSharedController := NewMockSharedController(gomock.NewController(t))
	testErrSharedController.EXPECT().Client().DoAndReturn(func() *client.Client {
		return nil
	}).AnyTimes()
	testSharedControllerWithAgent = NewSharedControllerWithAgent(testUserAgent, testErrSharedController)
	assert.Nil(t, testSharedControllerWithAgent.Client())
}

type userAgentRoundTripper struct {
	expectedUserAgent string
	t                 *testing.T
}

func (u userAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	assert.Equal(u.t, u.expectedUserAgent, req.Header.Get("User-Agent"))
	return &http.Response{}, nil
}
