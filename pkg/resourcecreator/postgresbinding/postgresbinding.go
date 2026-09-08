// Package postgresbinding connects workloads to pgrator-managed Postgres bindings.
package postgresbinding

import (
	"crypto/sha256"
	"fmt"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const mountRoot = "/var/run/secrets/nais.io/postgres"

const (
	credentialAdmin     = "admin"
	credentialRead      = "read"
	credentialReadWrite = "readwrite"
)

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
			return fmt.Errorf("postgres %q is used more than once by workload %q", postgres.Name, source.GetName())
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

func workloadType(kind string) (string, error) {
	switch kind {
	case "Application":
		return "application", nil
	case "Naisjob":
		return "job", nil
	default:
		return "", fmt.Errorf("unsupported PostgresBinding workload kind %q", kind)
	}
}

func bindingCredentials(role string) ([]string, error) {
	switch role {
	case "", credentialAdmin:
		return []string{credentialAdmin, credentialReadWrite}, nil
	case credentialRead:
		return []string{credentialRead}, nil
	case credentialReadWrite:
		return []string{credentialReadWrite}, nil
	default:
		return nil, fmt.Errorf("unsupported PostgresBinding role %q", role)
	}
}

func addBinding(source Source, ast *resource.Ast, workloadType string, postgres nais_io_v1.PostgresUse, credentials []string) {
	name := bindingName(postgres.Name, source.GetName())
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = name
	binding := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "nais.io/v1", "kind": "PostgresBinding",
		"metadata": map[string]any{"name": objectMeta.Name, "namespace": objectMeta.Namespace, "labels": objectMeta.Labels, "annotations": objectMeta.Annotations},
		"spec":     map[string]any{"postgres": postgres.Name, "consumer": map[string]any{"workload": map[string]any{"name": source.GetName(), "type": workloadType}}, "credentials": stringSlice(credentials)},
	}}
	ast.AppendOperation(resource.OperationCreateOrUpdate, binding)
	volumeName := volumeName("credentials", name)
	ast.Volumes = append(ast.Volumes, pod.FromFilesSecretVolumeWithMode(volumeName, name, credentialFiles(credentials), new(int32(0o440))))
	ast.VolumeMounts = append(ast.VolumeMounts, corev1.VolumeMount{Name: volumeName, MountPath: postgresMountPath(postgres.Name), ReadOnly: true})
	for _, credential := range credentials {
		prefix := postgres.EnvPrefix + connectionEnvPrefix(credential)
		for _, key := range connectionKeys {
			ast.AppendEnv(corev1.EnvVar{Name: prefix + key, ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: name}, Key: connectionEnvPrefix(credential) + key}}})
		}
		ast.AppendEnv(corev1.EnvVar{Name: prefix + "PGSSLCERT", Value: credentialMountPath(postgres.Name, credential) + "/tls.crt"}, corev1.EnvVar{Name: prefix + "PGSSLKEY", Value: credentialMountPath(postgres.Name, credential) + "/tls.key"}, corev1.EnvVar{Name: prefix + "PGSSLROOTCERT", Value: postgresMountPath(postgres.Name) + "/ca.crt"})
	}
}

func stringSlice(values []string) []any {
	result := make([]any, len(values))
	for i := range values {
		result[i] = values[i]
	}
	return result
}

func connectionEnvPrefix(credential string) string {
	if credential == credentialAdmin {
		return ""
	}
	return "PG" + credential + "_"
}

var connectionKeys = []string{"PGHOST", "PGPORT", "PGDATABASE", "PGUSER", "PGSSLMODE"}

func bindingName(postgres, workload string) string { return fmt.Sprintf("%s-%s", postgres, workload) }

func credentialFiles(credentials []string) []corev1.KeyToPath {
	files := []corev1.KeyToPath{{Key: "ca.crt", Path: "ca.crt"}}
	for _, credential := range credentials {
		files = append(files, corev1.KeyToPath{Key: credential + ".tls.crt", Path: credential + "/tls.crt"}, corev1.KeyToPath{Key: credential + ".tls.key", Path: credential + "/tls.key"})
	}
	return files
}

func volumeName(kind, identity string) string {
	hash := sha256.Sum256([]byte(identity))
	return fmt.Sprintf("postgres-%s-%x", kind, hash[:6])
}
func postgresMountPath(postgres string) string { return fmt.Sprintf("%s/%s", mountRoot, postgres) }
func credentialMountPath(postgres, credential string) string {
	return fmt.Sprintf("%s/%s", postgresMountPath(postgres), credential)
}

var _ = metav1.ObjectMeta{}
