package postgres

import (
	"fmt"
	"time"

	acid_zalan_do_v1 "github.com/nais/liberator/pkg/apis/acid.zalan.do/v1"
	google_iam_crd "github.com/nais/liberator/pkg/apis/iam.cnrm.cloud.google.com/v1beta1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	cpuLimitFactor = 4

	defaultMaintenanceStart = 5 * time.Hour
	maintenanceDuration     = 1

	allowDeletionAnnotation = "nais.io/postgresqlDeleteResource"

	defaultNumInstances = int32(2)
	haNumInstances      = int32(3)

	defaultSchema = "public"

	defaultDatabaseName = "app"
)

var defaultExtensions = []string{
	"pgaudit",
}

type Source interface {
	resource.Source
	GetPostgres() *nais_io_v1.Postgres
}

type Config interface {
	GetGoogleProjectID() string
	PostgresStorageClass() string
	PostgresImage() string
}

func Create(source Source, ast *resource.Ast, cfg Config) error {
	postgres := source.GetPostgres()
	if postgres == nil {
		return nil
	}

	pgClusterName := source.GetName()
	if postgres.Cluster.Name != "" {
		pgClusterName = postgres.Cluster.Name
	}
	pgNamespace := fmt.Sprintf("pg-%s", source.GetNamespace())

	CreateClusterSpec(source, ast, cfg, pgClusterName, pgNamespace)
	createNetworkPolicies(source, ast, pgClusterName, pgNamespace)
	createIAMPolicy(source, ast, cfg.GetGoogleProjectID(), pgNamespace)

	envVars := []corev1.EnvVar{
		{
			Name:  "PGHOST",
			Value: fmt.Sprintf("%s.%s", pgClusterName, pgNamespace),
		},
		{
			Name:  "PGPORT",
			Value: "5432",
		},
		{
			Name:  "PGDATABASE",
			Value: "app",
		},
		{
			Name: "PGUSER",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "username",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("app-owner-user.%s.credentials.postgresql.acid.zalan.do", pgClusterName)}}},
		},
		{
			Name: "PGPASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "password",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("app-owner-user.%s.credentials.postgresql.acid.zalan.do", pgClusterName)}}},
		},
		{
			Name:  "PGURL",
			Value: fmt.Sprintf("postgresql://$(PGUSER):$(PGPASSWORD)@%s.%s:5432/app?sslmode=disable", pgClusterName, pgNamespace),
		},
		{
			Name:  "PGJDBCURL",
			Value: fmt.Sprintf("jdbc:postgresql://%s.%s:5432/app?user=$(PGUSER)&password=$(PGPASSWORD)&sslmode=disable", pgClusterName, pgNamespace),
		},
	}

	ast.Env = append(ast.Env, envVars...)

	return nil
}

func createIAMPolicy(source Source, ast *resource.Ast, projectId, pgNamespace string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = fmt.Sprintf("pg-%s", source.GetNamespace())
	objectMeta.Namespace = google.IAMServiceAccountNamespace
	objectMeta.OwnerReferences = nil
	delete(objectMeta.Labels, "app")

	iamPolicy := google_iam_crd.IAMPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IAMPolicy",
			APIVersion: google.IAMAPIVersion,
		},
		ObjectMeta: objectMeta,
		Spec: google_iam_crd.IAMPolicySpec{
			ResourceRef: &google_iam_crd.ResourceRef{
				ApiVersion: google.IAMAPIVersion,
				Kind:       "IAMServiceAccount",
				Name:       ptr.To("postgres-pod"),
			},
			Bindings: []google_iam_crd.Bindings{
				{
					Role: "roles/iam.workloadIdentityUser",
					Members: []string{
						fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/postgres-pod]", projectId, pgNamespace),
					},
				},
			},
		},
	}

	util.SetAnnotation(&iamPolicy, google.ProjectIdAnnotation, projectId)

	ast.AppendOperation(resource.OperationCreateIfNotExists, &iamPolicy)
}

func createNetworkPolicies(source Source, ast *resource.Ast, pgClusterName, pgNamespace string) {
	createPostgresNetworkPolicy(source, ast, pgClusterName, pgNamespace)
	createSourceNetworkPolicy(source, ast, pgClusterName, pgNamespace)
}

func createPostgresNetworkPolicy(source Source, ast *resource.Ast, pgClusterName string, pgNamespace string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.OwnerReferences = nil
	objectMeta.Name = pgClusterName
	objectMeta.Namespace = pgNamespace

	pgNetpol := &networkingv1.NetworkPolicy{
		ObjectMeta: objectMeta,
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"application": "spilo",
					"app":         source.GetName(),
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"application": "spilo",
									"app":         source.GetName(),
								},
							},
						},
					},
				},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"application": "spilo",
									"app":         source.GetName(),
								},
							},
						},
					},
				},
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "nais-system",
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/name": "postgres-operator",
								},
							},
						},
					},
				},
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": source.GetNamespace(),
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app": source.GetName(),
								},
							},
						},
					},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
				networkingv1.PolicyTypeIngress,
			},
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, pgNetpol)
}

func createSourceNetworkPolicy(source Source, ast *resource.Ast, pgClusterName string, pgNamespace string) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = fmt.Sprintf("pg-%s", source.GetName())

	sourceNetpol := &networkingv1.NetworkPolicy{
		ObjectMeta: objectMeta,
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: "networking.k8s.io/v1",
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": source.GetName(),
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": pgNamespace,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"application": "spilo",
									"app":         source.GetName(),
								},
							},
						},
					},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, sourceNetpol)
}

func CreateClusterSpec(source Source, ast *resource.Ast, cfg Config, pgClusterName string, pgNamespace string) {
	postgres := source.GetPostgres()
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.OwnerReferences = nil
	objectMeta.Name = pgClusterName
	objectMeta.Namespace = pgNamespace
	objectMeta.Labels["apiserver-access"] = "enabled"

	if postgres.Cluster.AllowDeletion {
		objectMeta.Annotations[allowDeletionAnnotation] = pgClusterName
	}

	cpuLimit := postgres.Cluster.Resources.Cpu.DeepCopy()
	cpuLimit.Mul(cpuLimitFactor)

	numberOfInstances := defaultNumInstances
	if postgres.Cluster.HighAvailability {
		numberOfInstances = haNumInstances
	}

	maintenanceWindows := []acid_zalan_do_v1.MaintenanceWindow{}
	if postgres.MaintenanceWindow != nil && postgres.MaintenanceWindow.Day != 0 && postgres.MaintenanceWindow.Hour != nil {
		startTime := time.Hour * time.Duration(*postgres.MaintenanceWindow.Hour)

		maintenanceStartTime := metav1.NewTime(time.Time{}.Add(startTime))
		maintenanceEndTime := metav1.NewTime(maintenanceStartTime.Add(maintenanceDuration * time.Hour))

		maintenanceWindows = append(maintenanceWindows, acid_zalan_do_v1.MaintenanceWindow{
			Everyday:  postgres.MaintenanceWindow.Day == 0,
			Weekday:   makeWeekday(postgres),
			StartTime: maintenanceStartTime,
			EndTime:   maintenanceEndTime,
		})
	}

	extensions := map[string]string{}
	if postgres.Database != nil && postgres.Database.Extensions != nil {
		for _, extension := range postgres.Database.Extensions {
			extensions[extension.Name] = defaultSchema
		}
	}
	for _, extension := range defaultExtensions {
		extensions[extension] = defaultSchema
	}

	collation := "en_US.UTF-8"
	if postgres.Database != nil && postgres.Database.Collation != "" {
		collation = fmt.Sprintf("%s.UTF-8", postgres.Database.Collation)
	}
	cluster := &acid_zalan_do_v1.Postgresql{
		TypeMeta: metav1.TypeMeta{
			Kind:       "postgresql",
			APIVersion: "acid.zalan.do/v1",
		},
		ObjectMeta: objectMeta,
		Spec: acid_zalan_do_v1.PostgresSpec{
			PostgresqlParam: acid_zalan_do_v1.PostgresqlParam{
				PgVersion: postgres.Cluster.MajorVersion,
				Parameters: map[string]string{
					"log_destination": "jsonlog",
				},
			},
			Volume: acid_zalan_do_v1.Volume{
				Size:         postgres.Cluster.Resources.DiskSize.String(),
				StorageClass: cfg.PostgresStorageClass(),
			},
			Patroni: acid_zalan_do_v1.Patroni{
				InitDB: map[string]string{
					"encoding": "UTF8",
					"locale":   collation,
				},
				SynchronousMode:       true,
				SynchronousModeStrict: true,
			},
			Resources: &acid_zalan_do_v1.Resources{
				ResourceRequests: acid_zalan_do_v1.ResourceDescription{
					CPU:    ptr.To(postgres.Cluster.Resources.Cpu.String()),
					Memory: ptr.To(postgres.Cluster.Resources.Memory.String()),
				},
				ResourceLimits: acid_zalan_do_v1.ResourceDescription{
					CPU:    ptr.To(cpuLimit.String()),
					Memory: ptr.To(postgres.Cluster.Resources.Memory.String()),
				},
			},
			TeamID:             source.GetNamespace(),
			DockerImage:        cfg.PostgresImage(),
			NumberOfInstances:  numberOfInstances,
			MaintenanceWindows: maintenanceWindows,
			PreparedDatabases: map[string]acid_zalan_do_v1.PreparedDatabase{
				defaultDatabaseName: {
					DefaultUsers:    true,
					Extensions:      extensions,
					SecretNamespace: source.GetNamespace(),
				},
			},
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, cluster)
}

// makeWeekday creates a weekday from an integer day
// Weekday is Sun 0-6 Sat, while Day is Mon 1-7 Sun
func makeWeekday(postgres *nais_io_v1.Postgres) time.Weekday {
	if postgres.MaintenanceWindow == nil {
		return time.Tuesday
	}
	return time.Weekday(postgres.MaintenanceWindow.Day % 7)
}
