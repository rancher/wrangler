package types

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
	HasCondition(obj interface{}) bool
	GetStatus(obj interface{}) string
	SetStatus(obj interface{}, status string)
	True(obj interface{})
	IsTrue(obj interface{}) bool
	False(obj interface{})
	IsFalse(obj interface{}) bool
	Unknown(obj interface{})
	IsUnknown(obj interface{}) bool
	SetStatusBool(obj interface{}, val bool)
	SetError(obj interface{}, reason string, err error)
	MatchesError(obj interface{}, reason string, err error) bool
	GetReason(obj interface{}) string
	SetReason(obj interface{}, reason string)
	GetMessage(obj interface{}) string
	SetMessage(obj interface{}, msg string)
	SetMessageIfBlank(obj interface{}, message string)
}

type FluentCondition interface {
	Target(obj interface{}) FluentCondition
	SetStatus(status string) FluentCondition
	True() FluentCondition
	False() FluentCondition
	Unknown() FluentCondition
	SetStatusBool(val bool) FluentCondition
	SetError(reason string, err error) FluentCondition
	SetReason(reason string) FluentCondition
	SetMessage(message string) FluentCondition
	SetMessageIfBlank(message string) FluentCondition
	Apply(obj interface{}) bool
}
