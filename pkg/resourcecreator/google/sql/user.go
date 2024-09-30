package google_sql

import (
	"fmt"
	"strings"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	googlesqlcrd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/secret"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

const (
	googleSQLHostSuffix     = "_HOST"
	googleSQLPortSuffix     = "_PORT"
	googleSQLUsernameSuffix = "_USERNAME"
	GoogleSQLPasswordSuffix = "_PASSWORD"
	googleSQLDatabaseSuffix = "_DATABASE"
	googleSQLURLSuffix      = "_URL"
	googleSQLJDBCURLSuffix  = "_JDBC_URL"

	googleSQLPostgresHost    = "127.0.0.1"
	googleSQLPostgresPort    = "5432"
	googleSQLPostgresURL     = "postgres://%s:%s@%s:%s/%s"
	googleSQLPostgresJDBCURL = "jdbc:postgres://%s:%s/%s?user=%s&password=%s"
)

type GoogleSqlUser struct {
	Username string
	AppName  string
	DB       *nais.CloudSqlDatabase
	Instance *googlesqlcrd.SQLInstance
}

func CreateGoogleSQLUsers(source Source, ast *resource.Ast, cfg Config, naisSqlDatabase *nais_io_v1.CloudSqlDatabase, naisSqlInstance *nais_io_v1.CloudSqlInstance, googleSqlInstance *googlesqlcrd.SQLInstance) {
	sqlUsers := MergeAndFilterDatabaseSQLUsers(naisSqlDatabase.Users, googleSqlInstance.Name)

	for _, user := range sqlUsers {
		googleSqlUser := GoogleSqlUser{
			Username: user.Name,
			AppName:  source.GetName(),
			DB:       naisSqlDatabase,
			Instance: googleSqlInstance,
		}

		// Populate AST with Secret and SQLUser resources.
		googleSqlUser.createSqlUserDBResources(
			resource.CreateObjectMeta(source),
			ast,
			naisSqlInstance.CascadingDelete,
			source.GetName(),
			cfg,
		)

		// Inject a reference to the SQL user secret into the pod.
		secretName := googleSqlUser.googleSQLSecretName()
		ast.EnvFrom = append(ast.EnvFrom, pod.EnvFromSecret(secretName))
	}
}

func (in GoogleSqlUser) isDefault() bool {
	return in.Instance.Name == in.Username
}

func (in GoogleSqlUser) prefixIsSet() bool {
	return len(in.DB.EnvVarPrefix) > 0
}

func (in GoogleSqlUser) googleSqlUserPrefix() string {
	prefix := in.sqlUserEnvPrefix()
	if in.prefixIsSet() && !in.isDefault() {
		prefix = fmt.Sprintf("%s_%s", prefix, googleSQLDatabaseCase(trimLeadingUnderscore(in.Username)))
	}
	return prefix
}

func (in GoogleSqlUser) createSqlUserDBResources(objectMeta metav1.ObjectMeta, ast *resource.Ast, cascadingDelete bool, appName string, cfg Config) {
	secretName := in.googleSQLSecretName()
	googleSqlUser := in.Create(objectMeta, cascadingDelete, appName, cfg.GetGoogleTeamProjectID())

	if cfg != nil && cfg.ShouldCreateSqlInstanceInSharedVpc() && usingPrivateIP(in.Instance) {
		util.SetAnnotation(googleSqlUser, "sqeletor.nais.io/env-var-prefix", in.googleSqlUserPrefix())
		util.SetAnnotation(googleSqlUser, "sqeletor.nais.io/database-name", in.DB.Name)
	} else {
		password, err := util.GeneratePassword()
		if err != nil {
			// Will never happen
			panic(err)
		}

		vars := in.CreateUserEnvVars(password)
		ast.AppendOperation(resource.OperationCreateIfNotExists, secret.OpaqueSecret(objectMeta, secretName, vars))
	}

	ast.AppendOperation(resource.AnnotateIfExists, secret.OpaqueSecret(objectMeta, secretName, nil))

	ast.AppendOperation(resource.AnnotateIfExists, googleSqlUser)
	ast.AppendOperation(resource.OperationCreateIfNotExists, googleSqlUser)
}

func (in GoogleSqlUser) filterDefaultUserKey(key string, suffix string) string {
	if in.prefixIsSet() && in.isDefault() {
		prefix := in.googleSqlUserPrefix()
		noPrefixSubstring := strings.TrimPrefix(key, prefix)
		if noPrefixSubstring == suffix {
			return key
		}
	}
	return ""
}

func (in GoogleSqlUser) sqlUserEnvPrefix() string {
	if in.prefixIsSet() {
		return strings.TrimSuffix(in.DB.EnvVarPrefix, "_")
	}
	return fmt.Sprintf("NAIS_DATABASE_%s_%s", googleSQLDatabaseCase(trimLeadingUnderscore(in.Username)), googleSQLDatabaseCase(in.DB.Name))
}

func (in GoogleSqlUser) CreateUserEnvVars(password string) map[string]string {
	prefix := in.googleSqlUserPrefix()

	return map[string]string{
		prefix + googleSQLHostSuffix:     googleSQLPostgresHost,
		prefix + googleSQLPortSuffix:     googleSQLPostgresPort,
		prefix + googleSQLDatabaseSuffix: in.DB.Name,
		prefix + googleSQLUsernameSuffix: in.Username,
		prefix + GoogleSQLPasswordSuffix: password,
		prefix + googleSQLURLSuffix:      fmt.Sprintf(googleSQLPostgresURL, in.Username, password, googleSQLPostgresHost, googleSQLPostgresPort, in.DB.Name),
		prefix + googleSQLJDBCURLSuffix:  fmt.Sprintf(googleSQLPostgresJDBCURL, googleSQLPostgresHost, googleSQLPostgresPort, in.DB.Name, in.Username, password),
	}
}

func (in GoogleSqlUser) googleSQLSecretName() string {
	if in.Instance.Name == in.Username {
		return fmt.Sprintf("google-sql-%s", in.AppName)
	}

	shortName, err := namegen.ShortName(
		fmt.Sprintf(
			"google-sql-%s-%s-%s",
			in.AppName,
			in.DB.Name,
			replaceToLowerWithNoPrefix(in.Username),
		),
		validation.DNS1035LabelMaxLength)
	if err != nil {
		panic(err) // happens when Naiserator is out of memory
	}

	return shortName
}

func (in GoogleSqlUser) Create(objectMeta metav1.ObjectMeta, cascadingDelete bool, appName, projectId string) *googlesqlcrd.SQLUser {
	if in.isDefault() {
		objectMeta.Name = in.Instance.Name
	} else {
		objectMeta.Name = fmt.Sprintf("%s-%s-%s", appName, in.DB.Name, replaceToLowerWithNoPrefix(in.Username))
	}
	setAnnotations(objectMeta, cascadingDelete, projectId)

	secretName := in.googleSQLSecretName()

	return &googlesqlcrd.SQLUser{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SQLUser",
			APIVersion: "sql.cnrm.cloud.google.com/v1beta1",
		},
		ObjectMeta: objectMeta,
		Spec: googlesqlcrd.SQLUserSpec{
			InstanceRef: googlesqlcrd.InstanceRef{Name: in.Instance.Name},
			Password: googlesqlcrd.SqlUserPasswordValue{
				ValueFrom: googlesqlcrd.SqlUserPasswordSecretKeyRef{
					SecretKeyRef: googlesqlcrd.SecretRef{
						Key:  in.googleSqlUserPrefix() + GoogleSQLPasswordSuffix,
						Name: secretName,
					},
				},
			},
			ResourceID: in.Username,
		},
	}
}
