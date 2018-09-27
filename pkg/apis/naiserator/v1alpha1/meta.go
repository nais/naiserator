package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func (in *Application) CreateObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      in.Name,
		Namespace: in.Namespace,
		Labels: map[string]string{
			"app":   in.Name,
			"team": in.Spec.Team,
		},
		OwnerReferences: in.OwnerReferences(in),
	}
}

func (in *Application) OwnerReferences(app *Application) []metav1.OwnerReference {
	return []metav1.OwnerReference{app.GetOwnerReference()}
}
