package v1alpha1

import "k8s.io/apimachinery/pkg/runtime"

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *NaisDeployment) DeepCopyInto(out *NaisDeployment) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = NaisDeploymentSpec{
		A: in.Spec.A,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *NaisDeployment) DeepCopyObject() runtime.Object {
	out := NaisDeployment{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *NaisDeploymentList) DeepCopyObject() runtime.Object {
	out := NaisDeploymentList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]NaisDeployment, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}
