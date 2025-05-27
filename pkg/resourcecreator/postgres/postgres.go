package postgres

import (
	"fmt"
	"time"

	acid_zalan_do_v1 "github.com/nais/liberator/pkg/apis/acid.zalan.do/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
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

	createClusterSpec(source, ast, cfg, pgClusterName, pgNamespace, postgres)

	return nil
}

func createClusterSpec(source Source, ast *resource.Ast, cfg Config, pgClusterName string, pgNamespace string, postgres *nais_io_v1.Postgres) {
	objectMeta := resource.CreateObjectMeta(source)
	objectMeta.Name = pgClusterName
	objectMeta.Namespace = pgNamespace

	if postgres.Cluster.AllowDeletion {
		objectMeta.Annotations[allowDeletionAnnotation] = pgClusterName
	}

	cpuLimit := postgres.Cluster.Resources.Cpu.DeepCopy()
	cpuLimit.Mul(cpuLimitFactor)

	numberOfInstances := defaultNumInstances
	if postgres.Cluster.HighAvailability {
		numberOfInstances = haNumInstances
	}

	startTime := defaultMaintenanceStart
	if postgres.MaintenanceWindow.Hour != nil {
		startTime = time.Hour * time.Duration(*postgres.MaintenanceWindow.Hour)
	}
	maintenanceStartTime := metav1.NewTime(time.Time{}.Add(startTime))
	maintenanceEndTime := metav1.NewTime(maintenanceStartTime.Add(maintenanceDuration * time.Hour))

	extensions := map[string]string{}
	for _, extension := range postgres.Database.Extensions {
		extensions[extension.Name] = defaultSchema
	}
	for _, extension := range defaultExtensions {
		extensions[extension] = defaultSchema
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
					"locale":   fmt.Sprintf("%s.UTF-8", postgres.Database.Collation),
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
			TeamID:            source.GetNamespace(),
			DockerImage:       cfg.PostgresImage(),
			NumberOfInstances: numberOfInstances,
			MaintenanceWindows: []acid_zalan_do_v1.MaintenanceWindow{
				{
					Everyday:  postgres.MaintenanceWindow.Day == 0,
					Weekday:   makeWeekday(postgres),
					StartTime: maintenanceStartTime,
					EndTime:   maintenanceEndTime,
				},
			},
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
	return time.Weekday(postgres.MaintenanceWindow.Day % 7)
}
