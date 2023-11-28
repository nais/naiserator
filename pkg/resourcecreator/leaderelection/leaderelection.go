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
	"k8s.io/utils/pointer"
)

type ElectionMode int

const (
	ModeEndpoint ElectionMode = iota
	ModeLease
)

const endpointImage = "gcr.io/google_containers/leader-elector:0.5"

type Source interface {
	resource.Source
	GetLeaderElection() bool
}

type Config interface {
	GetLeaderElectionImage() string
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	if !source.GetLeaderElection() {
		return nil
	}

	electionMode := mode(cfg)
	appObjectMeta := resource.CreateObjectMeta(source)
	roleObjectMeta := resource.CreateObjectMeta(source)
	if electionMode == ModeLease {
		var err error
		roleObjectMeta.Name, err = namegen.ShortName(fmt.Sprintf("elector-%s", roleObjectMeta.Name), validation.DNS1123LabelMaxLength)
		if err != nil {
			return fmt.Errorf("failed to build short name for role: %w", err)
		}
	}

	var image string
	if electionMode == ModeLease {
		image = cfg.GetLeaderElectionImage()
	} else {
		image = endpointImage
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, role(roleObjectMeta, electionMode, source.GetName()))
	ast.AppendOperation(resource.OperationCreateOrRecreate, roleBinding(appObjectMeta, roleObjectMeta))
	ast.Containers = append(ast.Containers, container(source.GetName(), source.GetNamespace(), image))
	ast.Env = append(ast.Env, electorPathEnv())
	return nil
}

func mode(cfg Config) ElectionMode {
	if len(cfg.GetLeaderElectionImage()) != 0 {
		return ModeLease
	}
	return ModeEndpoint
}

func roleBinding(appObjectMeta, roleObjectMeta metav1.ObjectMeta) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: roleObjectMeta,
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleObjectMeta.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      appObjectMeta.Name,
				Namespace: appObjectMeta.Namespace,
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
					APIGroups: []string{
						"coordination.k8s.io",
					},
					Resources: []string{
						"leases",
					},
					Verbs: []string{
						"get",
						"create",
						"list",
						"watch",
					},
				},
				{
					APIGroups: []string{
						"",
					},
					Resources: []string{
						"pods",
					},
					Verbs: []string{
						"get",
						"list",
						"watch",
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

func container(name, namespace, image string) corev1.Container {
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
		Env: []corev1.EnvVar{
			{
				Name:  "ELECTOR_LOG_FORMAT",
				Value: "json",
			},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:                pointer.Int64(1069),
			RunAsGroup:               pointer.Int64(1069),
			RunAsNonRoot:             pointer.Bool(true),
			Privileged:               pointer.Bool(false),
			AllowPrivilegeEscalation: pointer.Bool(false),
			ReadOnlyRootFilesystem:   pointer.Bool(true),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
	}
}
