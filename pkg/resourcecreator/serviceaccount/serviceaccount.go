package serviceaccount

import (
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// TokenVolumeName is the name of the projected volume containing the workload's
	// service account token used for authentication against Nais services.
	TokenVolumeName = "nais-token"
	// TokenMountPath is the directory where the projected service account token
	// is mounted in the container.
	TokenMountPath = "/var/run/secrets/nais.io/serviceaccount"
	// TokenFileName is the file name of the projected service account token within TokenMountPath.
	TokenFileName = "token"
	// TokenAudience is the audience claim of the projected service account token.
	// A single shared audience is used for all internal Nais services; consumers
	// validate the sub-claim and perform their own access control.
	TokenAudience = "nais"
	// TokenExpirationSeconds is the requested lifetime of the projected token.
	// The kubelet refreshes the token automatically before it expires.
	TokenExpirationSeconds int64 = 600
	// TokenPathEnvVar exposes the on-disk path of the projected token to workloads.
	TokenPathEnvVar = "NAIS_SERVICE_ACCOUNT_TOKEN_PATH"
)

type Config interface {
	GetGoogleProjectID() string
	IsGCPEnabled() bool
}

// ManagedByLabel marks a ServiceAccount as created and managed by naiserator.
// Used by the ValidatingAdmissionPolicy that locks the SA to its owning workload.
const (
	ManagedByLabel = "nais.io/managed-by"
	ManagedByValue = "naiserator"
)

func Create(source resource.Source, ast *resource.Ast, cfg Config) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Labels[ManagedByLabel] = ManagedByValue

	if cfg.IsGCPEnabled() {
		googleProjectID := cfg.GetGoogleProjectID()
		objectMeta.Annotations["iam.gke.io/gcp-service-account"] = google.GcpServiceAccountName(resource.CreateAppNamespaceHash(source), googleProjectID)
	}

	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
		ObjectMeta: objectMeta,
	}

	ast.AppendOperation(resource.OperationCreateIfNotExists, serviceAccount)

	mountToken(ast)
}

// mountToken injects a projected service account token into the workload's pod spec.
// The token is intended for authenticating against Nais-internal services that accept
// Kubernetes-issued tokens (workload identity).
func mountToken(ast *resource.Ast) {
	ast.Volumes = append(ast.Volumes, corev1.Volume{
		Name: TokenVolumeName,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
							Path:              TokenFileName,
							Audience:          TokenAudience,
							ExpirationSeconds: new(TokenExpirationSeconds),
						},
					},
				},
			},
		},
	})

	ast.VolumeMounts = append(ast.VolumeMounts, corev1.VolumeMount{
		Name:      TokenVolumeName,
		MountPath: TokenMountPath,
		ReadOnly:  true,
	})

	ast.AppendEnv(corev1.EnvVar{
		Name:  TokenPathEnvVar,
		Value: TokenMountPath + "/" + TokenFileName,
	})
}
