package google

const (
	IAMAPIVersion              = "iam.cnrm.cloud.google.com/v1beta1"
	IAMServiceAccountNamespace = "serviceaccounts"
	StorageAPIVersion          = "storage.cnrm.cloud.google.com/v1beta1"
	BigQueryAPIVersion         = "google.nais.io/v1"
	Region                     = "europe-north1"
	DeletionPolicyAnnotation   = "cnrm.cloud.google.com/deletion-policy"
	DeletionPolicyAbandon      = "abandon"
	CascadingDeleteAnnotation  = "cnrm.cloud.google.com/delete-contents-on-destroy"
	ProjectIdAnnotation        = "cnrm.cloud.google.com/project-id"
	CloudSQLProxyTermTimeout   = "30s"
)