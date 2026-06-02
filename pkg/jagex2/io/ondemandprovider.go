package io

// OnDemandProvider is the model loader's hook back to the cache subsystem.
// Java: jagex2.io.OnDemandProvider (a base class with a single requestModel).
type OnDemandProvider interface {
	RequestModel(id int)
}
