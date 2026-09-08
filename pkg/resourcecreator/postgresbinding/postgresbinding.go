// Package postgresbinding connects workloads to pgrator-managed Postgres bindings.
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

	seen := make(map[string]struct{}, len(uses.Postgres))
	for _, postgres := range uses.Postgres {
		if _, ok := seen[postgres.Name]; ok {
			return fmt.Errorf("Postgres %q is used more than once by workload %q", postgres.Name, source.GetName())
		}
		seen[postgres.Name] = struct{}{}

		credentials, err := bindingCredentials(postgres.Role)
		if err != nil {
			return err
		}
		addBinding(source, ast, workloadType, postgres, credentials)
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

func bindingCredentials(role string) ([]pgrator_v1.PostgresBindingCredential, error) {
	switch role {
	case "", string(pgrator_v1.PostgresBindingCredentialAdmin):
		return []pgrator_v1.PostgresBindingCredential{
			pgrator_v1.PostgresBindingCredentialAdmin,
			pgrator_v1.PostgresBindingCredentialReadWrite,
		}, nil
	case string(pgrator_v1.PostgresBindingCredentialRead):
		return []pgrator_v1.PostgresBindingCredential{pgrator_v1.PostgresBindingCredentialRead}, nil
	case string(pgrator_v1.PostgresBindingCredentialReadWrite):
		return []pgrator_v1.PostgresBindingCredential{pgrator_v1.PostgresBindingCredentialReadWrite}, nil
	default:
		return nil, fmt.Errorf("unsupported PostgresBinding role %q", role)
	}
}

func addBinding(source Source, ast *resource.Ast, workloadType pgrator_v1.PostgresBindingWorkloadType, postgres nais_io_v1.PostgresUse, credentials []pgrator_v1.PostgresBindingCredential) {
	name := bindingName(postgres.Name, source.GetName())
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = name

	binding := &pgrator_v1.PostgresBinding{
		TypeMeta:   metav1.TypeMeta{APIVersion: pgrator_v1.GroupVersion.String(), Kind: "PostgresBinding"},
		ObjectMeta: objectMeta,
		Spec: pgrator_v1.PostgresBindingSpec{
			Postgres: postgres.Name,
			Consumer: pgrator_v1.PostgresBindingConsumer{Workload: &pgrator_v1.PostgresBindingWorkload{
				Name: source.GetName(), Type: workloadType,
			}},
			Credentials: credentials,
		},
	}
	ast.AppendOperation(resource.OperationCreateOrUpdate, binding)

	volumeName := volumeName("credentials", name)
	ast.Volumes = append(ast.Volumes, pod.FromFilesSecretVolumeWithMode(volumeName, name, credentialFiles(credentials), new(int32(0o440))))
	ast.VolumeMounts = append(ast.VolumeMounts, corev1.VolumeMount{Name: volumeName, MountPath: postgresMountPath(postgres.Name), ReadOnly: true})

	for _, credential := range credentials {
		prefix := postgres.EnvPrefix + pgrator_v1.ConnectionEnvPrefix(credential)
		for _, key := range connectionKeys {
			ast.AppendEnv(corev1.EnvVar{Name: prefix + key, ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: name}, Key: pgrator_v1.ConnectionEnvPrefix(credential) + key},
			}})
		}
		ast.AppendEnv(
			corev1.EnvVar{Name: prefix + "PGSSLCERT", Value: credentialMountPath(postgres.Name, credential) + "/tls.crt"},
			corev1.EnvVar{Name: prefix + "PGSSLKEY", Value: credentialMountPath(postgres.Name, credential) + "/tls.key"},
			corev1.EnvVar{Name: prefix + "PGSSLROOTCERT", Value: postgresMountPath(postgres.Name) + "/ca.crt"},
		)
	}
}

var connectionKeys = []string{"PGHOST", "PGPORT", "PGDATABASE", "PGUSER", "PGSSLMODE"}

func bindingName(postgres, workload string) string { return fmt.Sprintf("%s-%s", postgres, workload) }

func credentialFiles(credentials []pgrator_v1.PostgresBindingCredential) []corev1.KeyToPath {
	files := []corev1.KeyToPath{{Key: "ca.crt", Path: "ca.crt"}}
	for _, credential := range credentials {
		files = append(files,
			corev1.KeyToPath{Key: string(credential) + ".tls.crt", Path: string(credential) + "/tls.crt"},
			corev1.KeyToPath{Key: string(credential) + ".tls.key", Path: string(credential) + "/tls.key"},
		)
	}
	return files
}

func volumeName(kind, identity string) string {
	hash := sha256.Sum256([]byte(identity))
	return fmt.Sprintf("postgres-%s-%x", kind, hash[:6])
}

func postgresMountPath(postgres string) string { return fmt.Sprintf("%s/%s", mountRoot, postgres) }
func credentialMountPath(postgres string, credential pgrator_v1.PostgresBindingCredential) string {
	return fmt.Sprintf("%s/%s", postgresMountPath(postgres), credential)
}
