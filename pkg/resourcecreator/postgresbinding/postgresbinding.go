// Package postgresbinding connects workloads to CloudNativePG Postgres instances.
package postgresbinding

import (
	"crypto/sha256"
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	pgrator_v1 "github.com/nais/pgrator/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const mountRoot = "/var/run/secrets/nais.io/postgres"

type Source interface {
	resource.Source
	GetUses() *nais_io_v1.Uses
}

func Create(source Source, ast *resource.Ast) error {
	uses := source.GetUses()
	if uses == nil {
		return nil
	}

	workloadType, err := workloadType(source.GetOwnerReference().Kind)
	if err != nil {
		return err
	}

	for _, postgres := range uses.Postgres {
		roles, err := bindingRoles(postgres.Role)
		if err != nil {
			return err
		}

		addCAVolume(ast, postgres.Name)
		for _, role := range roles {
			addBinding(source, ast, workloadType, postgres, role)
		}
	}
	return nil
}

func workloadType(kind string) (pgrator_v1.PostgresBindingWorkloadType, error) {
	switch kind {
	case "Application":
		return pgrator_v1.PostgresBindingWorkloadTypeApplication, nil
	case "Naisjob":
		return pgrator_v1.PostgresBindingWorkloadTypeJob, nil
	default:
		return "", fmt.Errorf("unsupported PostgresBinding workload kind %q", kind)
	}
}

func bindingRoles(role string) ([]pgrator_v1.PostgresBindingRole, error) {
	switch role {
	case "", string(pgrator_v1.PostgresBindingRoleAdmin):
		return []pgrator_v1.PostgresBindingRole{
			pgrator_v1.PostgresBindingRoleAdmin,
			pgrator_v1.PostgresBindingRoleReadWrite,
		}, nil
	case string(pgrator_v1.PostgresBindingRoleRead):
		return []pgrator_v1.PostgresBindingRole{pgrator_v1.PostgresBindingRoleRead}, nil
	case string(pgrator_v1.PostgresBindingRoleReadWrite):
		return []pgrator_v1.PostgresBindingRole{pgrator_v1.PostgresBindingRoleReadWrite}, nil
	default:
		return nil, fmt.Errorf("unsupported PostgresBinding role %q", role)
	}
}

func addBinding(source Source, ast *resource.Ast, workloadType pgrator_v1.PostgresBindingWorkloadType, postgres nais_io_v1.PostgresUse, role pgrator_v1.PostgresBindingRole) {
	name := bindingName(postgres.Name, source.GetName(), role)
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = name

	binding := &pgrator_v1.PostgresBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: pgrator_v1.GroupVersion.String(),
			Kind:       "PostgresBinding",
		},
		ObjectMeta: objectMeta,
		Spec: pgrator_v1.PostgresBindingSpec{
			Postgres: postgres.Name,
			Consumer: pgrator_v1.PostgresBindingConsumer{
				Workload: &pgrator_v1.PostgresBindingWorkload{
					Name: source.GetName(),
					Type: workloadType,
				},
			},
			SecretName: name + "-client-cert",
			Role:       role,
		},
	}
	ast.AppendOperation(resource.OperationCreateOrUpdate, binding)

	ast.EnvFrom = append(ast.EnvFrom, corev1.EnvFromSource{
		Prefix: postgres.EnvPrefix,
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: name},
		},
	})

	addClientCertificate(ast, postgres, role, binding.Spec.SecretName)
}

func bindingName(postgres, workload string, role pgrator_v1.PostgresBindingRole) string {
	return fmt.Sprintf("%s-%s-%s", postgres, workload, role)
}

func roleEnvPrefix(role pgrator_v1.PostgresBindingRole) string {
	switch role {
	case pgrator_v1.PostgresBindingRoleRead:
		return "READ_"
	case pgrator_v1.PostgresBindingRoleReadWrite:
		return "READWRITE_"
	default:
		return ""
	}
}

func addCAVolume(ast *resource.Ast, postgres string) {
	name := volumeName("ca", postgres)
	ast.Volumes = append(ast.Volumes, pod.FromFilesSecretVolumeWithMode(
		name,
		postgres+"-ca",
		[]corev1.KeyToPath{{Key: "ca.crt", Path: "ca.crt"}},
		new(int32(0o440)),
	))
	ast.VolumeMounts = append(ast.VolumeMounts, corev1.VolumeMount{
		Name:      name,
		MountPath: caMountPath(postgres),
		ReadOnly:  true,
	})
}

func addClientCertificate(ast *resource.Ast, postgres nais_io_v1.PostgresUse, role pgrator_v1.PostgresBindingRole, secretName string) {
	name := volumeName("client", secretName)
	mountPath := clientMountPath(postgres.Name, role)
	ast.Volumes = append(ast.Volumes, pod.FromFilesSecretVolumeWithMode(
		name,
		secretName,
		[]corev1.KeyToPath{
			{Key: "tls.crt", Path: "tls.crt"},
			{Key: "tls.key", Path: "tls.key"},
		},
		new(int32(0o440)),
	))
	ast.VolumeMounts = append(ast.VolumeMounts, corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
		ReadOnly:  true,
	})

	prefix := postgres.EnvPrefix + roleEnvPrefix(role)
	ast.AppendEnv(
		corev1.EnvVar{Name: prefix + "PGSSLCERT", Value: mountPath + "/tls.crt"},
		corev1.EnvVar{Name: prefix + "PGSSLKEY", Value: mountPath + "/tls.key"},
		corev1.EnvVar{Name: prefix + "PGSSLROOTCERT", Value: caMountPath(postgres.Name) + "/ca.crt"},
	)
}

func volumeName(kind, identity string) string {
	hash := sha256.Sum256([]byte(identity))
	return fmt.Sprintf("postgres-%s-%x", kind, hash[:6])
}

func caMountPath(postgres string) string {
	return fmt.Sprintf("%s/%s/ca", mountRoot, postgres)
}

func clientMountPath(postgres string, role pgrator_v1.PostgresBindingRole) string {
	return fmt.Sprintf("%s/%s/%s", mountRoot, postgres, role)
}
