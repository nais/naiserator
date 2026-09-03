// Package postgresapi contains the minimal legacy data.nais.io Postgres API
// surface needed while workloads migrate from spec.postgres to
// spec.uses.postgres.
package postgresapi

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var GroupVersion = schema.GroupVersion{Group: "data.nais.io", Version: "v1"}

var SchemeBuilder = runtime.NewSchemeBuilder(func(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion, &Postgres{}, &PostgresList{})
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
})

var AddToScheme = SchemeBuilder.AddToScheme

// Postgres is the minimal legacy resource representation needed for engine
// detection. Its spec is intentionally omitted.
type Postgres struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Status            *PostgresStatus `json:"status,omitempty"`
}

// PostgresStatus contains the legacy engine discriminator.
type PostgresStatus struct {
	Engine string `json:"engine,omitempty"`
}

// DeepCopyObject implements runtime.Object.
func (in *Postgres) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(Postgres)
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Status != nil {
		out.Status = new(PostgresStatus)
		*out.Status = *in.Status
	}
	return out
}

// PostgresList contains legacy Postgres resources.
type PostgresList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Postgres `json:"items"`
}

// DeepCopyObject implements runtime.Object.
func (in *PostgresList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(PostgresList)
	*out = *in
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]Postgres, len(in.Items))
		for i := range in.Items {
			out.Items[i] = *in.Items[i].DeepCopyObject().(*Postgres)
		}
	}
	return out
}
