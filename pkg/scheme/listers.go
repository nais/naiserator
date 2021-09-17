package naiserator_scheme

import (
	azure_microsoft_com_v1alpha1 "github.com/nais/liberator/pkg/apis/azure.microsoft.com/v1alpha1"
	iam_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	networking_istio_io_v1alpha3 "github.com/nais/liberator/pkg/apis/networking.istio.io/v1alpha3"
	security_istio_io_v1beta1 "github.com/nais/liberator/pkg/apis/security.istio.io/v1beta1"
	sql_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	storage_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Maintain a list of resources that should be cleaned up during Application synchronization.
// Orphaned types with matching label 'app=NAME' but without owner references are automatically removed.
// It is too expensive to list all known Kubernetes types, so we need to have some knowledge of which types to care about.
// These are usually the types we persist to the cluster with names different than the application name.

// Resources that can be queried in all clusters
func GenericListers() []runtime.Object {
	return []runtime.Object{
		// Kubernetes internals
		&appsv1.DeploymentList{},
		&v2beta2.HorizontalPodAutoscalerList{},
		&corev1.SecretList{},
		&corev1.ServiceAccountList{},
		&corev1.ServiceList{},
		&networkingv1.NetworkPolicyList{},
		&networkingv1beta1.IngressList{},
		&rbacv1.RoleBindingList{},
		&rbacv1.RoleList{},

		// Custom resources
		&nais_io_v1.AzureAdApplicationList{},
		&nais_io_v1.IDPortenClientList{},
		&nais_io_v1.JwkerList{},
		&nais_io_v1.MaskinportenClientList{},
	}
}

// Resources that exist only in GCP clusters
func GCPListers() []runtime.Object {
	return []runtime.Object{
		&iam_cnrm_cloud_google_com_v1beta1.IAMPolicyList{},
		&iam_cnrm_cloud_google_com_v1beta1.IAMPolicyMemberList{},
		&iam_cnrm_cloud_google_com_v1beta1.IAMServiceAccountList{},
		&sql_cnrm_cloud_google_com_v1beta1.SQLDatabaseList{},
		&sql_cnrm_cloud_google_com_v1beta1.SQLInstanceList{},
		&sql_cnrm_cloud_google_com_v1beta1.SQLUserList{},
		&storage_cnrm_cloud_google_com_v1beta1.StorageBucketAccessControlList{},
		&storage_cnrm_cloud_google_com_v1beta1.StorageBucketList{},
	}
}

func ASOListers() []runtime.Object {
	return []runtime.Object{
		&azure_microsoft_com_v1alpha1.PostgreSQLDatabaseList{},
		&azure_microsoft_com_v1alpha1.PostgreSQLUserList{},
	}
}

func IstioListers() [] runtime.Object {
	return []runtime.Object{
		&security_istio_io_v1beta1.AuthorizationPolicyList{},
		&networking_istio_io_v1alpha3.ServiceEntryList{},
		&networking_istio_io_v1alpha3.VirtualServiceList{},
	}
}