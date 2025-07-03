package leaderelection

import (
	"fmt"

	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/ptr"
)

const (
	Port      = 4040
	ProbePort = 4041
)

type Source interface {
	resource.Source
	GetLeaderElection() bool
	GetPort() int
}

type Config interface {
	GetLeaderElectionImage() string
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	if !source.GetLeaderElection() {
		return nil
	}
	image := cfg.GetLeaderElectionImage()
	if image == "" {
		return fmt.Errorf("leader election image not configured")
	}

	port := source.GetPort()
	if port == Port || port == ProbePort {
		return fmt.Errorf("cannot use port '%d'; conflicts with leader election sidecar", port)
	}

	appObjectMeta := resource.CreateObjectMeta(source)
	roleBindingObjectMeta := resource.CreateObjectMeta(source)

	var err error
	roleBindingObjectMeta.Name, err = namegen.ShortName(fmt.Sprintf("elector-%s", roleBindingObjectMeta.Name), validation.DNS1123LabelMaxLength)
	if err != nil {
		return fmt.Errorf("failed to build short name for role binding: %w", err)
	}

	ast.AppendOperation(resource.OperationCreateOrRecreate, roleBinding(appObjectMeta, roleBindingObjectMeta))
	ast.InitContainers = append(ast.InitContainers, container(source.GetName(), source.GetNamespace(), image))
	ast.PrependEnv(electorEnv()...)
	return nil
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
			Kind:     "ClusterRole",
			Name:     "elector",
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

func electorEnv() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "ELECTOR_PATH",
			Value: "localhost:4040",
		},
		{
			Name:  "ELECTOR_GET_URL",
			Value: fmt.Sprintf("http://localhost:%d/", Port),
		},
		{
			Name:  "ELECTOR_SSE_URL",
			Value: fmt.Sprintf("http://localhost:%d/sse", Port),
		},
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
		RestartPolicy: ptr.To(corev1.ContainerRestartPolicyAlways),
		Ports: []corev1.ContainerPort{{
			ContainerPort: Port,
			Protocol:      corev1.ProtocolTCP,
		}},
		Args: []string{
			fmt.Sprintf("--election=%s", name),
			fmt.Sprintf("--election-namespace=%s", namespace),
			fmt.Sprintf("--http=localhost:%d", Port),
			fmt.Sprintf("--probe-address=0.0.0.0:%d", ProbePort),
		},
		Env: []corev1.EnvVar{
			{
				Name:  "ELECTOR_LOG_FORMAT",
				Value: "json",
			},
		},
		SecurityContext: pod.DefaultContainerSecurityContext(),
		StartupProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt32(ProbePort),
				},
			},
		},
	}
}
