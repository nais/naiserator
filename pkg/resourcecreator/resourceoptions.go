package resourcecreator

// ResourceOptions defines customizations for resource objects.
type ResourceOptions struct {
	NumReplicas   int32
	AccessPolicy  bool
	AccessPolicyExceptIPs []string
	NativeSecrets bool
}

// NewResourceOptions creates a struct with the default resource options.
func NewResourceOptions() ResourceOptions {
	return ResourceOptions{
		NumReplicas: 1,
	}
}
