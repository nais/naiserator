package naiserator_scheme

import (
	iam_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	networking_istio_io_v1alpha3 "github.com/nais/liberator/pkg/apis/networking.istio.io/v1alpha3"
	sql_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	storage_cnrm_cloud_google_com_v1beta1 "github.com/nais/liberator/pkg/apis/storage.cnrm.cloud.google.com/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
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
func Listers() []runtime.Object {
	return []runtime.Object{

		// Kubernetes internals
		&appsv1.DeploymentList{},
		&autoscalingv1.HorizontalPodAutoscalerList{},
		&corev1.SecretList{},
		&corev1.ServiceAccountList{},
		&corev1.ServiceList{},
		&networking_istio_io_v1alpha3.ServiceEntryList{},
		&networking_istio_io_v1alpha3.VirtualServiceList{},
		&networkingv1.NetworkPolicyList{},
		&networkingv1beta1.IngressList{},
		&rbacv1.RoleBindingList{},
		&rbacv1.RoleList{},

		// Custom resources
		&iam_cnrm_cloud_google_com_v1beta1.IAMPolicyList{},
		&iam_cnrm_cloud_google_com_v1beta1.IAMPolicyMemberList{},
		&iam_cnrm_cloud_google_com_v1beta1.IAMServiceAccountList{},
		&nais_io_v1.AzureAdApplicationList{},
		&nais_io_v1.IDPortenClientList{},
		&nais_io_v1.JwkerList{},
		&nais_io_v1.MaskinportenClientList{},
		&sql_cnrm_cloud_google_com_v1beta1.SQLDatabaseList{},
		&sql_cnrm_cloud_google_com_v1beta1.SQLInstanceList{},
		&sql_cnrm_cloud_google_com_v1beta1.SQLUserList{},
		&storage_cnrm_cloud_google_com_v1beta1.StorageBucketAccessControlList{},
		&storage_cnrm_cloud_google_com_v1beta1.StorageBucketList{},
	}

}
