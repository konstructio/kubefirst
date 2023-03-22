package argocd

const ArgoCDAPIVersion string = "argoproj.io/v1alpha1"

// PatchStringValue specifies a patch operation for a string
type PatchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}
