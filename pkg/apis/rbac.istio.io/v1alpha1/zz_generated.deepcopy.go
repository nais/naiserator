// +build !ignore_autogenerated

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AccessRule) DeepCopyInto(out *AccessRule) {
	*out = *in
	if in.Services != nil {
		in, out := &in.Services, &out.Services
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Methods != nil {
		in, out := &in.Methods, &out.Methods
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Paths != nil {
		in, out := &in.Paths, &out.Paths
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AccessRule.
func (in *AccessRule) DeepCopy() *AccessRule {
	if in == nil {
		return nil
	}
	out := new(AccessRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RoleRef) DeepCopyInto(out *RoleRef) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RoleRef.
func (in *RoleRef) DeepCopy() *RoleRef {
	if in == nil {
		return nil
	}
	out := new(RoleRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceRole) DeepCopyInto(out *ServiceRole) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceRole.
func (in *ServiceRole) DeepCopy() *ServiceRole {
	if in == nil {
		return nil
	}
	out := new(ServiceRole)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ServiceRole) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceRoleBinding) DeepCopyInto(out *ServiceRoleBinding) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceRoleBinding.
func (in *ServiceRoleBinding) DeepCopy() *ServiceRoleBinding {
	if in == nil {
		return nil
	}
	out := new(ServiceRoleBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ServiceRoleBinding) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceRoleBindingList) DeepCopyInto(out *ServiceRoleBindingList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ServiceRoleBinding, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceRoleBindingList.
func (in *ServiceRoleBindingList) DeepCopy() *ServiceRoleBindingList {
	if in == nil {
		return nil
	}
	out := new(ServiceRoleBindingList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ServiceRoleBindingList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceRoleBindingSpec) DeepCopyInto(out *ServiceRoleBindingSpec) {
	*out = *in
	if in.Subjects != nil {
		in, out := &in.Subjects, &out.Subjects
		*out = make([]*Subject, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(Subject)
				**out = **in
			}
		}
	}
	if in.RoleRef != nil {
		in, out := &in.RoleRef, &out.RoleRef
		*out = new(RoleRef)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceRoleBindingSpec.
func (in *ServiceRoleBindingSpec) DeepCopy() *ServiceRoleBindingSpec {
	if in == nil {
		return nil
	}
	out := new(ServiceRoleBindingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceRoleList) DeepCopyInto(out *ServiceRoleList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ServiceRole, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceRoleList.
func (in *ServiceRoleList) DeepCopy() *ServiceRoleList {
	if in == nil {
		return nil
	}
	out := new(ServiceRoleList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ServiceRoleList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceRoleSpec) DeepCopyInto(out *ServiceRoleSpec) {
	*out = *in
	if in.Rules != nil {
		in, out := &in.Rules, &out.Rules
		*out = make([]*AccessRule, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(AccessRule)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceRoleSpec.
func (in *ServiceRoleSpec) DeepCopy() *ServiceRoleSpec {
	if in == nil {
		return nil
	}
	out := new(ServiceRoleSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Subject) DeepCopyInto(out *Subject) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Subject.
func (in *Subject) DeepCopy() *Subject {
	if in == nil {
		return nil
	}
	out := new(Subject)
	in.DeepCopyInto(out)
	return out
}
