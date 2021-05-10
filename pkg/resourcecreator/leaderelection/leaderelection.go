package leaderelection

import (
	"fmt"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Create(app *nais.Application, deployment *appsv1.Deployment, operations *resource.Operations) {
	if !app.Spec.LeaderElection {
		return
	}

	*operations = append(*operations, resource.Operation{Resource: role(app), Operation: resource.OperationCreateOrUpdate})
	*operations = append(*operations, resource.Operation{Resource: roleBinding(app), Operation: resource.OperationCreateOrRecreate})
	podSpec(app, &deployment.Spec.Template.Spec)
}

func roleBinding(app *nais.Application) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: v1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: app.CreateObjectMeta(),
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     app.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      app.Name,
				Namespace: app.Namespace,
			},
		},
	}
}

func role(app *nais.Application) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: v1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: app.CreateObjectMeta(),
		Rules: []rbacv1.PolicyRule{
			{
				ResourceNames: []string{
					app.Name,
				},
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"endpoints",
				},
				Verbs: []string{
					"get",
					"update",
				},
			},
		},
	}
}

func podSpec(app *nais.Application, spec *corev1.PodSpec) {
	spec.Containers = append(spec.Containers, leaderElectionContainer(app.Name, app.Namespace))
	mainContainer := spec.Containers[0].DeepCopy()

	electorPathEnv := corev1.EnvVar{
		Name:  "ELECTOR_PATH",
		Value: "localhost:4040",
	}

	mainContainer.Env = append(mainContainer.Env, electorPathEnv)
	spec.Containers[0] = *mainContainer
}

func leaderElectionContainer(name, namespace string) corev1.Container {
	return corev1.Container{
		Name:            "elector",
		Image:           "gcr.io/google_containers/leader-elector:0.5",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: k8sResource.MustParse("100m"),
			},
		},
		Ports: []corev1.ContainerPort{{
			ContainerPort: 4040,
			Protocol:      corev1.ProtocolTCP,
		}},
		Args: []string{fmt.Sprintf("--election=%s", name), "--http=localhost:4040", fmt.Sprintf("--election-namespace=%s", namespace)},
	}
}
