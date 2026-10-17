package controlapi

import "runtime"

// ProductIdentity is the runtime product shown by the shared GUI.
// Darwin stays OpenSurge for Mac; this fork's Linux control plane is fnOS.
type ProductIdentity struct {
	Name    string `json:"name"`
	Edition string `json:"edition"`
}

func runtimeProduct() ProductIdentity {
	if runtime.GOOS == "linux" {
		return ProductIdentity{Name: "OpenSurge", Edition: "fnos"}
	}
	return ProductIdentity{Name: "OpenSurge", Edition: "mac"}
}
