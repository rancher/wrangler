package types

import "sigs.k8s.io/controller-runtime/pkg/client"

// LegacyCondition maps to logic related to the wrangler GenericCondition
type LegacyCondition interface {
	GetStatus(obj interface{}) string
	SetStatus(obj interface{}, status string)
	SetStatusBool(obj interface{}, val bool)
	True(obj interface{})
	IsTrue(obj interface{}) bool
	False(obj interface{})
	IsFalse(obj interface{}) bool
	Unknown(obj interface{})
	IsUnknown(obj interface{}) bool
	LastUpdated(obj interface{}, ts string)
	GetLastUpdated(obj interface{}) string
	SetError(obj interface{}, reason string, err error)
	MatchesError(obj interface{}, reason string, err error) bool
	GetReason(obj interface{}) string
	Reason(obj interface{}, reason string)
	GetMessage(obj interface{}) string
	Message(obj interface{}, msg string)
	SetMessageIfBlank(obj interface{}, message string)
}

// Condition maps to logic related to the k8s native KEP-1623
type Condition interface {
	HasCondition(obj client.Object) bool
	GetStatus(obj client.Object) string
	SetStatus(obj client.Object, status string)
	True(obj client.Object)
	IsTrue(obj client.Object) bool
	False(obj client.Object)
	IsFalse(obj client.Object) bool
	Unknown(obj client.Object)
	IsUnknown(obj client.Object) bool
	SetStatusBool(obj client.Object, val bool)
	SetError(obj client.Object, reason string, err error)
	MatchesError(obj client.Object, reason string, err error) bool
	GetReason(obj client.Object) string
	SetReason(obj client.Object, reason string)
	GetMessage(obj client.Object) string
	SetMessage(obj client.Object, msg string)
	SetMessageIfBlank(obj client.Object, message string)
}

type FluentCondition interface {
	Target(obj client.Object) FluentCondition
	SetStatus(status string) FluentCondition
	True() FluentCondition
	False() FluentCondition
	Unknown() FluentCondition
	SetStatusBool(val bool) FluentCondition
	SetError(reason string, err error) FluentCondition
	SetReason(reason string) FluentCondition
	SetMessage(message string) FluentCondition
	SetMessageIfBlank(message string) FluentCondition
	Apply(obj client.Object) bool
}
