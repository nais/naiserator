package leaderelection

import (
	"fmt"

	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

type ElectionMode int

const (
	ModeEndpoint ElectionMode = iota
	ModeLease
)

type Source interface {
	resource.Source
	GetLeaderElection() bool
}

func Create(source resource.Source, ast *resource.Ast, cfg Config) error {
	if !source.GetLeaderElection() {
		return nil
	}

	electionMode := mode(options)
	roleObjectMeta := resource.CreateObjectMeta(source)
	if electionMode == ModeLease {
		var err error
		roleObjectMeta.Name, err = namegen.ShortName(fmt.Sprintf("elector-%s", roleObjectMeta.Name), validation.DNS1123LabelMaxLength)
		if err != nil {
			return fmt.Errorf("failed to build short name for role: %w", err)
		}
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, role(roleObjectMeta, electionMode, source.GetName()))
	ast.AppendOperation(resource.OperationCreateOrRecreate, roleBinding(roleObjectMeta))
	ast.Containers = append(ast.Containers, container(source.GetName(), source.GetNamespace(), options))
	ast.Env = append(ast.Env, electorPathEnv())
	return nil
}

func mode(options resource.Options) ElectionMode {
	if options.LeaderElection.Image != "" {
		return ModeLease
	}
	return ModeEndpoint
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

func role(objectMeta metav1.ObjectMeta, electionMode ElectionMode, resourceName string) *rbacv1.Role {
	if electionMode == ModeEndpoint {
		return &rbacv1.Role{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "Role",
			},
			ObjectMeta: objectMeta,
			Rules: []rbacv1.PolicyRule{
				{
					ResourceNames: []string{
						resourceName,
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
	} else {
		return &rbacv1.Role{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "Role",
			},
			ObjectMeta: objectMeta,
			Rules: []rbacv1.PolicyRule{
				{
					ResourceNames: []string{
						resourceName,
					},
					APIGroups: []string{
						"coordination.k8s.io",
					},
					Resources: []string{
						"leases",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
						"create",
					},
				},
			},
		}
	}
}

func electorPathEnv() corev1.EnvVar {
	return corev1.EnvVar{
		Name:  "ELECTOR_PATH",
		Value: "localhost:4040",
	}
}

func container(name, namespace string, options resource.Options) corev1.Container {
	image := "gcr.io/google_containers/leader-elector:0.5"
	if options.LeaderElection.Image != "" {
		image = options.LeaderElection.Image
	}
	return corev1.Container{
		Name:            "elector",
		Image:           image,
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
