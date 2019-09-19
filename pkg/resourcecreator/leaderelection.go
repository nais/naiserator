package resourcecreator

import (
	"fmt"

	nais "github.com/nais/naiserator/pkg/apis/nais.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func LeaderElectionRoleBinding(app *nais.Application) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
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

func LeaderElectionRole(app *nais.Application) *rbacv1.Role {
	return &rbacv1.Role{
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
					"list",
					"watch",
					"create",
					"update",
					"patch",
				},
			},
		},
	}
}

func LeaderElection(app *nais.Application, podSpec *corev1.PodSpec) (spec *corev1.PodSpec) {
	spec = podSpec.DeepCopy()
	spec.Containers = append(spec.Containers, leaderElectionContainer(app.Name, app.Namespace))
	mainContainer := spec.Containers[0].DeepCopy()

	electorPathEnv := corev1.EnvVar{
		Name:  "ELECTOR_PATH",
		Value: "localhost:4040",
	}

	mainContainer.Env = append(mainContainer.Env, electorPathEnv)
	spec.Containers[0] = *mainContainer

	return spec
}

func leaderElectionContainer(name, namespace string) corev1.Container {
	return corev1.Container{
		Name:            "elector",
		Image:           "gcr.io/google_containers/leader-elector:0.5",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("100m"),
			},
		},
		Ports: []corev1.ContainerPort{{
			ContainerPort: 4040,
			Protocol:      corev1.ProtocolTCP,
		}},
		Args: []string{fmt.Sprintf("--election=%s", name), "--http=localhost:4040", fmt.Sprintf("--election-namespace=%s", namespace)},
	}
}
