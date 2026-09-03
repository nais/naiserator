package naiserator_scheme

import (
	aiven_io_v1alpha1 "github.com/nais/liberator/pkg/apis/aiven.io/v1alpha1"
	aiven_nais_io_v1 "github.com/nais/liberator/pkg/apis/aiven.nais.io/v1"
	fqdn "github.com/nais/liberator/pkg/apis/fqdnnetworkpolicies.networking.gke.io/v1alpha3"
	google_nais_io_v1 "github.com/nais/liberator/pkg/apis/google.nais.io/v1"
	iam_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	kafka_nais_io_v1 "github.com/nais/liberator/pkg/apis/kafka.nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	sql_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	storage_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	pgrator_v1 "github.com/nais/pgrator/pkg/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalev2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Maintain a list of resources that should be cleaned up during Application synchronization.
// Orphaned types with matching label 'app=NAME' but without owner references are automatically removed.
// It is too expensive to list all known Kubernetes types, so we need to have some knowledge of which types to care about.
// These are usually the types we persist to the cluster with names different than the application name.

// GenericListers returns resources that can be queried in all clusters
func GenericListers() []client.ObjectList {
	return []client.ObjectList{
		// Kubernetes internals
		&appsv1.DeploymentList{},
		&autoscalev2.HorizontalPodAutoscalerList{},
		&corev1.ConfigMapList{},
		&corev1.SecretList{},
		&corev1.ServiceAccountList{},
		&corev1.ServiceList{},
		&networkingv1.IngressList{},
		&networkingv1.NetworkPolicyList{},
		&policyv1.PodDisruptionBudgetList{},
		&rbacv1.RoleBindingList{},
		&rbacv1.RoleList{},
		&v1.CronJobList{},
		&v1.JobList{},

		// Custom resources
		&aiven_nais_io_v1.AivenApplicationList{},
		&kafka_nais_io_v1.StreamList{},
		&nais_io_v1.AzureAdApplicationList{},
		&nais_io_v1.IDPortenClientList{},
		&nais_io_v1.JwkerList{},
		&nais_io_v1.MaskinportenClientList{},
	}
}

// PostgresListers returns resources that exist only in Postgres-enabled clusters.
func PostgresListers() []client.ObjectList {
	return []client.ObjectList{
		&pgrator_v1.PostgresBindingList{},
	}
}

// GCPListers returns resources that exist only in a GCP clusters
func GCPListers() []client.ObjectList {
	return []client.ObjectList{
		&fqdn.FQDNNetworkPolicyList{},
		&iam_cnrm_cloud_google_com_v1beta1.IAMPolicyList{},
		&iam_cnrm_cloud_google_com_v1beta1.IAMPolicyMemberList{},
		&iam_cnrm_cloud_google_com_v1beta1.IAMServiceAccountList{},
		&sql_cnrm_cloud_google_com_v1beta1.SQLDatabaseList{},
		&sql_cnrm_cloud_google_com_v1beta1.SQLInstanceList{},
		&sql_cnrm_cloud_google_com_v1beta1.SQLSSLCertList{},
		&sql_cnrm_cloud_google_com_v1beta1.SQLUserList{},
		&storage_cnrm_cloud_google_com_v1beta1.StorageBucketAccessControlList{},
		&storage_cnrm_cloud_google_com_v1beta1.StorageBucketList{},

		&google_nais_io_v1.BigQueryDatasetList{},
	}
}

// AivenListers returns resources that exist only in a Aiven supported clusters
func AivenListers() []client.ObjectList {
	return []client.ObjectList{
		&aiven_io_v1alpha1.OpenSearchList{},
		&aiven_io_v1alpha1.ValkeyList{},
	}
}
