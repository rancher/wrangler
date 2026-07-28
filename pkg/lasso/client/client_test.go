package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
)

var accessor = meta.NewAccessor()
var testSchema = runtime.NewScheme()
var negotiatedSerializer = serializer.NewCodecFactory(testSchema)

func TestMain(m *testing.M) {
	// Setup Schemas for serializer before we run our test
	metav1.AddMetaToScheme(testSchema)
	v1.SchemeBuilder.AddToScheme(testSchema)
	m.Run()
}

// Object is generic object to wrap runtime and metav1.
type Object interface {
	metav1.Object
	runtime.Object
}

type testInfo struct {
	name    string
	fields  clientFields
	args    testArgs
	wantErr bool
}
type clientFields struct {
	Namespaced bool
	GVR        schema.GroupVersionResource
	Kind       string
}
type testArgs struct {
	obj       Object
	namespace string
	desired   runtime.Object
	result    runtime.Object
}

func TestClient_Get(t *testing.T) {
	t.Parallel()
	tests := []*testInfo{
		{
			name: "base test",
			fields: clientFields{
				GVR: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				Namespaced: true,
				Kind:       "Pod",
			},
			args: testArgs{
				obj: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
				},
				namespace: "bar",
				result:    &v1.Pod{},
				desired: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "bar",
						Annotations: map[string]string{"superhero": "batman"},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
				},
			},
		},
		{
			name:    "empty name",
			wantErr: true,
			fields: clientFields{
				GVR: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				Namespaced: true,
				Kind:       "Pod",
			},
			args: testArgs{
				obj: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "bar",
					},
				},
				namespace: "bar",
				result:    &v1.Pod{},
				desired: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "bar",
						Annotations: map[string]string{"superhero": "batman"},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			mockRESTClient := &fake.RESTClient{
				GroupVersion:         test.fields.GVR.GroupVersion(),
				NegotiatedSerializer: negotiatedSerializer,
			}
			c := NewClient(test.fields.GVR, test.fields.Kind, test.fields.Namespaced, mockRESTClient, 0)
			handler := newRequestHandler(c, test.args.desired, test.args.namespace, false, false)
			mockRESTClient.Client = fake.CreateHTTPClient(handler)
			err := c.Get(context.TODO(), test.args.namespace, test.args.obj.GetName(), test.args.result, metav1.GetOptions{})
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.args.desired, test.args.result)
		})
	}
}

func TestClient_List(t *testing.T) {
	t.Parallel()
	tests := []*testInfo{
		{
			name: "base test",
			fields: clientFields{
				GVR: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				Namespaced: true,
				Kind:       "Pod",
			},
			args: testArgs{
				namespace: "bar",
				result:    &v1.PodList{},
				desired: &v1.PodList{
					Items: []v1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:        "foo",
								Namespace:   "bar",
								Annotations: map[string]string{"superhero": "batman"},
							},
							TypeMeta: metav1.TypeMeta{
								Kind:       "Pod",
								APIVersion: "v1",
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			mockRESTClient := &fake.RESTClient{
				GroupVersion:         test.fields.GVR.GroupVersion(),
				NegotiatedSerializer: negotiatedSerializer,
			}
			c := NewClient(test.fields.GVR, test.fields.Kind, test.fields.Namespaced, mockRESTClient, 0)
			handler := newRequestHandler(c, test.args.desired, test.args.namespace, true, false)
			mockRESTClient.Client = fake.CreateHTTPClient(handler)
			err := c.List(context.TODO(), test.args.namespace, test.args.result, metav1.ListOptions{})
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.args.desired, test.args.result)
		})
	}
}

func TestClient_Update(t *testing.T) {
	t.Parallel()
	tests := []*testInfo{
		{
			name: "base test",
			fields: clientFields{
				GVR: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				Namespaced: true,
				Kind:       "Pod",
			},
			args: testArgs{
				obj: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
				},
				namespace: "bar",
				result:    &v1.Pod{},
				desired: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "bar",
						Annotations: map[string]string{"superhero": "batman"},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			mockRESTClient := &fake.RESTClient{
				GroupVersion:         test.fields.GVR.GroupVersion(),
				NegotiatedSerializer: negotiatedSerializer,
			}
			c := NewClient(test.fields.GVR, test.fields.Kind, test.fields.Namespaced, mockRESTClient, 0)
			handler := newRequestHandler(c, test.args.desired, test.args.namespace, false, false)
			mockRESTClient.Client = fake.CreateHTTPClient(handler)
			err := c.Update(context.TODO(), test.args.namespace, test.args.obj, test.args.result, metav1.UpdateOptions{})
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.args.desired, test.args.result)

			require.True(t, equality.Semantic.DeepEqual(test.args.desired, test.args.result))
		})
	}
}
func TestClient_UpdateStatus(t *testing.T) {
	t.Parallel()
	tests := []*testInfo{
		{
			name: "base test",
			fields: clientFields{
				GVR: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				Namespaced: true,
				Kind:       "Pod",
			},
			args: testArgs{
				obj: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
				},
				namespace: "bar",
				result:    &v1.Pod{},
				desired: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
					Status: v1.PodStatus{
						Message: "Good to Go",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			mockRESTClient := &fake.RESTClient{
				GroupVersion:         test.fields.GVR.GroupVersion(),
				NegotiatedSerializer: negotiatedSerializer,
			}
			c := NewClient(test.fields.GVR, test.fields.Kind, test.fields.Namespaced, mockRESTClient, 0)
			handler := newRequestHandler(c, test.args.desired, test.args.namespace, false, true)
			mockRESTClient.Client = fake.CreateHTTPClient(handler)
			err := c.UpdateStatus(context.TODO(), test.args.namespace, test.args.obj, test.args.result, metav1.UpdateOptions{})
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.args.desired, test.args.result)

			require.True(t, equality.Semantic.DeepEqual(test.args.desired, test.args.result))
		})
	}
}
func TestClient_Create(t *testing.T) {
	t.Parallel()
	tests := []*testInfo{
		{
			name: "base test",
			fields: clientFields{
				GVR: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				Namespaced: true,
				Kind:       "Pod",
			},
			args: testArgs{
				obj: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
				},
				namespace: "bar",
				result:    &v1.Pod{},
				desired: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			mockRESTClient := &fake.RESTClient{
				GroupVersion:         test.fields.GVR.GroupVersion(),
				NegotiatedSerializer: negotiatedSerializer,
			}
			c := NewClient(test.fields.GVR, test.fields.Kind, test.fields.Namespaced, mockRESTClient, 0)
			handler := newRequestHandler(c, test.args.desired, test.args.namespace, false, false)
			mockRESTClient.Client = fake.CreateHTTPClient(handler)
			err := c.Create(context.TODO(), test.args.namespace, test.args.obj, test.args.result, metav1.CreateOptions{})
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.args.desired, test.args.result)
		})
	}
}

func TestClient_Delete(t *testing.T) {
	t.Parallel()
	tests := []*testInfo{
		{
			name: "base test",
			fields: clientFields{
				GVR: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				Namespaced: true,
				Kind:       "Pod",
			},
			args: testArgs{
				obj: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
				},
				namespace: "bar",
			},
		},
		{
			name:    "test empty resource name",
			wantErr: true,
			fields: clientFields{
				GVR: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				Namespaced: true,
				Kind:       "Pod",
			},
			args: testArgs{
				obj: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "bar",
					},
				},
				namespace: "bar",
			},
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockRESTClient := &fake.RESTClient{
				GroupVersion:         test.fields.GVR.GroupVersion(),
				NegotiatedSerializer: negotiatedSerializer,
			}
			c := NewClient(test.fields.GVR, test.fields.Kind, test.fields.Namespaced, mockRESTClient, 0)
			handler := newRequestHandler(c, test.args.obj, test.args.namespace, false, false)
			mockRESTClient.Client = fake.CreateHTTPClient(handler)
			err := c.Delete(context.TODO(), test.args.namespace, test.args.obj.GetName(), metav1.DeleteOptions{})
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestClient_DeleteCollection(t *testing.T) {
	t.Parallel()
	tests := []*testInfo{
		{
			name: "base test",
			fields: clientFields{
				GVR: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				Namespaced: true,
				Kind:       "Pod",
			},
			args: testArgs{
				namespace: "bar",
			},
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			mockRESTClient := &fake.RESTClient{
				GroupVersion:         test.fields.GVR.GroupVersion(),
				NegotiatedSerializer: negotiatedSerializer,
			}
			c := NewClient(test.fields.GVR, test.fields.Kind, test.fields.Namespaced, mockRESTClient, 0)
			handler := newRequestHandler(c, test.args.desired, test.args.namespace, true, false)
			mockRESTClient.Client = fake.CreateHTTPClient(handler)
			err := c.DeleteCollection(context.TODO(), test.args.namespace, metav1.DeleteOptions{}, metav1.ListOptions{})
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestClient_Watch(t *testing.T) {
	t.Parallel()
	tests := []*testInfo{
		{
			name: "base test",
			fields: clientFields{
				GVR: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				Namespaced: true,
				Kind:       "Pod",
			},
			args: testArgs{
				namespace: "bar",
			},
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockRESTClient := &fake.RESTClient{
				GroupVersion:         test.fields.GVR.GroupVersion(),
				NegotiatedSerializer: negotiatedSerializer,
			}
			c := NewClient(test.fields.GVR, test.fields.Kind, test.fields.Namespaced, mockRESTClient, 0)
			handler := newRequestHandler(c, test.args.desired, test.args.namespace, true, false)
			mockRESTClient.Client = fake.CreateHTTPClient(handler)
			_, err := c.Watch(context.TODO(), test.args.namespace, metav1.ListOptions{})
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestClient_Watch_Stop(t *testing.T) {
	t.Parallel()
	client := &Client{kind: "testkind"}
	mockWatcher := watch.NewFake()
	w, err := client.injectKind(mockWatcher, nil)
	require.NoError(t, err)

	resultChan := w.ResultChan()
	mockWatcher.Add(nil)
	w.Stop()

	// Check that ResultChan is properly closed when Stop() is called
	select {
	case _, ok := <-resultChan:
		if ok {
			t.Error("expected result channel to be closed")
		}
	default:
		t.Error("expected result channel to be closed")
	}
}

func TestClient_Patch(t *testing.T) {
	t.Parallel()
	tests := []*testInfo{
		{
			name: "base test",
			fields: clientFields{
				GVR: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				Namespaced: true,
				Kind:       "Pod",
			},
			args: testArgs{
				obj: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
				},
				namespace: "bar",
				result:    &v1.Pod{},
				desired: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "bar",
						Annotations: map[string]string{"superhero": "batman"},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
				},
			},
		},
		{
			name:    "missing namespace test",
			wantErr: true,
			fields: clientFields{
				GVR: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				Namespaced: true,
				Kind:       "Pod",
			},
			args: testArgs{
				obj: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "bar",
					},
				},
				result: &v1.Pod{},
				desired: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "foo",
						Namespace:   "bar",
						Annotations: map[string]string{"superhero": "batman"},
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "Pod",
						APIVersion: "v1",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		test := tt
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			mockRESTClient := &fake.RESTClient{
				GroupVersion:         test.fields.GVR.GroupVersion(),
				NegotiatedSerializer: negotiatedSerializer,
			}
			client := NewClient(test.fields.GVR, test.fields.Kind, test.fields.Namespaced, mockRESTClient, 0)
			handler := newRequestHandler(client, test.args.desired, test.args.namespace, false, false)
			mockRESTClient.Client = fake.CreateHTTPClient(handler)
			err := client.Patch(context.TODO(), test.args.namespace, test.args.obj.GetName(), types.JSONPatchType, nil, test.args.result, metav1.PatchOptions{})
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.args.desired, test.args.result)
		})
	}
}

func TestClient_WithAgent(t *testing.T) {
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}
	const agentName = "test-agent"
	var testRT fakeRoundTripper
	testConfig := rest.Config{
		ContentConfig: rest.ContentConfig{
			NegotiatedSerializer: serializer.WithoutConversionCodecFactory{},
		},
		Transport: &testRT,
	}
	testRestClient, err := rest.UnversionedRESTClientFor(&testConfig)
	require.NoError(t, err)
	client := NewClient(gvr, "Pod", true, testRestClient, 0)
	client.Config = testConfig
	originalConfig := client.Config
	newClient, err := client.WithAgent(agentName)
	require.NoError(t, err, "failed to call WithAgent on client.")

	// Verify that the original client was not changed
	require.NotEqual(t, client, newClient, "WithAgent failed to create a new client")
	require.Equal(t, originalConfig, client.Config, "WithAgent mutated the original client configuration")

	// Verify that the new client correctly set User-Agent
	require.Equal(t, newClient.Config.UserAgent, agentName, "WithAgent did not set the correct agent name.")
	newClient.Config.UserAgent = originalConfig.UserAgent
	require.Equal(t, originalConfig, newClient.Config, "WithAgent did not persist original configuration")
	_ = newClient.RESTClient.Get().Do(context.TODO())
	require.Equal(t, agentName, testRT.lastRequest.Header.Get("User-Agent"), "Client created by WithAgent did not set the header")

	client.Config.NegotiatedSerializer = nil
	_, err = client.WithAgent(agentName)
	require.Error(t, err, "expected failure when calling with agent with out a serializer set")
}

func TestClient_WithImpersonation(t *testing.T) {
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "pods",
	}

	impersonateConfig := rest.ImpersonationConfig{
		UserName: "Zeus",
		Groups:   []string{"Greek", "Gods", "Elemental"},
		Extra:    map[string][]string{"Parents": {"Cronus", "Rhea"}},
	}

	var testRT fakeRoundTripper
	testConfig := rest.Config{
		ContentConfig: rest.ContentConfig{
			NegotiatedSerializer: serializer.WithoutConversionCodecFactory{},
		},
		Transport: &testRT,
	}
	testRestClient, err := rest.UnversionedRESTClientFor(&testConfig)
	require.NoError(t, err)
	client := NewClient(gvr, "Pod", true, testRestClient, 0)
	client.Config = testConfig
	originalConfig := client.Config
	newClient, err := client.WithImpersonation(impersonateConfig)
	require.NoError(t, err, "failed to call WithImpersonation on client.")

	// Verify that the original client was not changed
	require.NotEqual(t, client, newClient, "WithImpersonation failed to create a new client")
	require.Equal(t, originalConfig, client.Config, "WithImpersonation mutated the original client configuration")

	// Verify that the new client correctly set impersonation
	require.Equal(t, newClient.Config.Impersonate, impersonateConfig, "WithImpersonation did not set the correct impersonation config.")
	newClient.Config.Impersonate = originalConfig.Impersonate
	require.Equal(t, originalConfig, newClient.Config, "WithImpersonation did not persist original configuration")
	_ = newClient.RESTClient.Get().Do(context.TODO())
	require.Equal(t, impersonateConfig, getImpersonateFromRequest(testRT.lastRequest), "Client created by WithImpersonation did not set the correct headers")

	client.Config.NegotiatedSerializer = nil
	_, err = client.WithImpersonation(impersonateConfig)
	require.Error(t, err, "expected failure when calling with agent with out a serializer set")
}

func newRequestHandler(client *Client, retObj runtime.Object, namespace string, isCollection, isStatus bool) func(req *http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		// dynamically create expected path
		expectedPath := path.Join("/", path.Join(client.prefix...))
		if client.Namespaced {
			expectedPath = path.Join(expectedPath, "namespaces", namespace)
		}
		expectedPath = path.Join(expectedPath, client.resource)
		if req.Method != http.MethodPost && !isCollection {
			name, err := accessor.Name(retObj)
			if err != nil {
				return nil, fmt.Errorf("failed to get object name: %s", err)
			}
			expectedPath = path.Join(expectedPath, name)
		}
		if isStatus {
			expectedPath = path.Join(expectedPath, "status")
		}

		if !strings.EqualFold(req.URL.Path, expectedPath) {
			return nil, fmt.Errorf("unexpected request got: '%s' wanted: '%s'", req.URL.Path, expectedPath)
		}
		retData, err := json.Marshal(retObj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshall desired return Object: %w", err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(retData)),
		}, nil
	}
}
