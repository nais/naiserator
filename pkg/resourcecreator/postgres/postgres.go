package postgres

import (
	"fmt"
	"time"

	acid_zalan_do_v1 "github.com/nais/liberator/pkg/apis/acid.zalan.do/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	// Max length is 63, but we need to save space for suffixes added by Zalando operator or StatefulSets
	maxClusterNameLength = 50

	cpuLimitFactor = 4

	maintenanceDuration = 1

	allowDeletionAnnotation = "nais.io/postgresqlDeleteResource"

	defaultNumInstances = int32(2)
	haNumInstances      = int32(3)

	defaultSchema = "public"

	defaultDatabaseName = "app"

	sharedPreloadLibraries = "bg_mon,pg_stat_statements,pgextwlist,pg_auth_mon,set_user,timescaledb,pg_cron,pg_stat_kcache,pgaudit"

	runAsUser  = int64(101)
	runAsGroup = int64(103)
	fsGroup    = int64(103)
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

	var err error
	pgClusterName := source.GetName()
	if postgres.Cluster.Name != "" {
		pgClusterName = postgres.Cluster.Name
	}
	if len(pgClusterName) > maxClusterNameLength {
		pgClusterName, err = namegen.ShortName(pgClusterName, maxClusterNameLength)
		if err != nil {
			return fmt.Errorf("failed to shorten PostgreSQL cluster name: %w", err)
		}
	}
	pgNamespace := fmt.Sprintf("pg-%s", source.GetNamespace())

	CreateClusterSpec(source, ast, cfg, pgClusterName, pgNamespace)
	createNetworkPolicies(source, ast, pgClusterName, pgNamespace)
	err = createIAMPolicyMember(source, ast, cfg.GetGoogleProjectID(), pgNamespace)
	if err != nil {
		return fmt.Errorf("failed to create IAMPolicyMember: %w", err)
	}

	envVars := []corev1.EnvVar{
		{
			Name:  "PGHOST",
			Value: fmt.Sprintf("%s-pooler.%s", pgClusterName, pgNamespace),
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
						Name: fmt.Sprintf("app-owner-user.%s.credentials.postgresql.acid.zalan.do", pgClusterName),
					},
				},
			},
		},
		{
			Name: "PGPASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "password",
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("app-owner-user.%s.credentials.postgresql.acid.zalan.do", pgClusterName),
					},
				},
			},
		},
		{
			Name:  "PGURL",
			Value: fmt.Sprintf("postgresql://$(PGUSER):$(PGPASSWORD)@%s-pooler.%s:5432/app", pgClusterName, pgNamespace),
		},
		{
			Name:  "PGJDBCURL",
			Value: fmt.Sprintf("jdbc:postgresql://%s-pooler.%s:5432/app?user=$(PGUSER)&password=$(PGPASSWORD)", pgClusterName, pgNamespace),
		},
	}

	ast.Env = append(ast.Env, envVars...)

	return nil
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
			EnableConnectionPooler:        ptr.To(true),
			EnableReplicaConnectionPooler: ptr.To(false),
			ConnectionPooler: &acid_zalan_do_v1.ConnectionPooler{
				Resources: &acid_zalan_do_v1.Resources{
					ResourceRequests: acid_zalan_do_v1.ResourceDescription{
						CPU:    ptr.To("50m"),
						Memory: ptr.To("50Mi"),
					},
				},
			},
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "nais.io/type",
									Operator: "In",
									Values:   []string{"postgres"},
								},
							},
						},
					},
				},
			},
			PostgresqlParam: acid_zalan_do_v1.PostgresqlParam{
				PgVersion:  postgres.Cluster.MajorVersion,
				Parameters: makePostgresParameters(postgres.Cluster.Audit),
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
					PreparedSchemas: map[string]acid_zalan_do_v1.PreparedSchema{
						defaultSchema: {},
					},
				},
			},
			SpiloRunAsUser:  ptr.To(runAsUser),
			SpiloRunAsGroup: ptr.To(runAsGroup),
			SpiloFSGroup:    ptr.To(fsGroup),
		},
	}

	ast.AppendOperation(resource.OperationCreateOrUpdate, cluster)
}

func makePostgresParameters(audit *nais_io_v1.PostgresAudit) map[string]string {
	postgresParameters := map[string]string{
		"log_destination":          "jsonlog",
		"log_filename":             "postgresql.log",
		"shared_preload_libraries": sharedPreloadLibraries,
		"pg_stat_statements.track": "all",
		"track_io_timing":          "on",
	}
	if audit != nil && audit.Enabled {
		classes := ""
		if len(audit.StatementClasses) == 0 {
			classes = "write,ddl,role"
		}
		for _, statementClass := range audit.StatementClasses {
			if classes != "" {
				classes += ","
			}
			classes += string(statementClass)
		}
		postgresParameters["pgaudit.log"] = classes
		postgresParameters["pgaudit.log_parameter"] = "on"
	}
	return postgresParameters
}

// makeWeekday creates a weekday from an integer day
// Weekday is Sun 0-6 Sat, while Day is Mon 1-7 Sun
func makeWeekday(postgres *nais_io_v1.Postgres) time.Weekday {
	if postgres.MaintenanceWindow == nil {
		return time.Tuesday
	}
	return time.Weekday(postgres.MaintenanceWindow.Day % 7)
}
