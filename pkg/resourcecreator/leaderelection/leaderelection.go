package leaderelection

import (
	"fmt"

	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Create(source resource.Source, ast *resource.Ast, leaderElection bool) {
	if !leaderElection {
		return
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, role(resource.CreateObjectMeta(source)))
	ast.AppendOperation(resource.OperationCreateOrRecreate, roleBinding(resource.CreateObjectMeta(source)))
	ast.Containers = append(ast.Containers, container(source.GetName(), source.GetNamespace()))
	ast.Env = append(ast.Env, electorPathEnv())
}

func roleBinding(objectMeta metav1.ObjectMeta) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: objectMeta,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     objectMeta.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      objectMeta.Name,
				Namespace: objectMeta.Namespace,
			},
		},
	}
}

func role(objectMeta metav1.ObjectMeta) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: objectMeta,
		Rules: []rbacv1.PolicyRule{
			{
				ResourceNames: []string{
					objectMeta.Name,
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

func electorPathEnv() corev1.EnvVar {
	return corev1.EnvVar{
		Name:  "ELECTOR_PATH",
		Value: "localhost:4040",
	}
}

func container(name, namespace string) corev1.Container {
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
