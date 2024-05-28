package google_sql

import (
	"fmt"
	"strings"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	googlesqlcrd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/resourcecreator/secret"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	Name     string
	DB       *nais.CloudSqlDatabase
	Instance *googlesqlcrd.SQLInstance
}

func CreateGoogleSQLUsers(source Source, ast *resource.Ast, cfg Config, naisSqlDatabase *nais_io_v1.CloudSqlDatabase, naisSqlInstance *nais_io_v1.CloudSqlInstance, googleSqlInstance *googlesqlcrd.SQLInstance) error {
	sqlUsers, err := MergeAndFilterDatabaseSQLUsers(naisSqlDatabase.Users, googleSqlInstance.Name)
	if err != nil {
		return err
	}

	for _, user := range sqlUsers {
		googleSqlUser := GoogleSqlUser{
			Name:     user.Name,
			DB:       naisSqlDatabase,
			Instance: googleSqlInstance,
		}

		err := googleSqlUser.createSqlUserDBResources(resource.CreateObjectMeta(source), ast, naisSqlInstance.CascadingDelete, source.GetName(), cfg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (in GoogleSqlUser) isDefault() bool {
	return in.Instance.Name == in.Name
}

func (in GoogleSqlUser) prefixIsSet() bool {
	return len(in.DB.EnvVarPrefix) > 0
}

func (in GoogleSqlUser) googleSqlUserPrefix() string {
	prefix := in.sqlUserEnvPrefix()
	if in.prefixIsSet() && !in.isDefault() {
		prefix = fmt.Sprintf("%s_%s", prefix, googleSQLDatabaseCase(trimPrefix(in.Name)))
	}
	return prefix
}

func (in GoogleSqlUser) createSqlUserDBResources(objectMeta metav1.ObjectMeta, ast *resource.Ast, cascadingDelete bool, appName string, cfg Config) error {
	secretName, err := GoogleSQLSecretName(
		appName, in.Instance.Name, in.DB.Name, in.Name,
	)
	if err != nil {
		return fmt.Errorf("unable to create sql secret name: %s", err)
	}

	sqlUser, err := in.Create(objectMeta, cascadingDelete, appName, cfg.GetGoogleTeamProjectID())
	if err != nil {
		return fmt.Errorf("unable to create sql user: %s", err)
	}

	if cfg != nil && cfg.ShouldCreateSqlInstanceInSharedVpc() && usingPrivateIP(in.Instance) {
		util.SetAnnotation(sqlUser, "sqeletor.nais.io/env-var-prefix", in.googleSqlUserPrefix())
		util.SetAnnotation(sqlUser, "sqeletor.nais.io/database-name", in.DB.Name)
	} else {
		password, err := util.GeneratePassword()
		if err != nil {
			return err
		}

		vars := in.CreateUserEnvVars(password)
		ast.AppendOperation(resource.OperationCreateIfNotExists, secret.OpaqueSecret(objectMeta, secretName, vars))
	}

	ast.AppendOperation(resource.AnnotateIfExists, secret.OpaqueSecret(objectMeta, secretName, nil))
	ast.AppendOperation(resource.OperationCreateIfNotExists, sqlUser)
	return nil
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
	return fmt.Sprintf("NAIS_DATABASE_%s_%s", googleSQLDatabaseCase(trimPrefix(in.Name)), googleSQLDatabaseCase(in.DB.Name))
}

func (in GoogleSqlUser) CreateUserEnvVars(password string) map[string]string {
	prefix := in.googleSqlUserPrefix()

	return map[string]string{
		prefix + googleSQLHostSuffix:     googleSQLPostgresHost,
		prefix + googleSQLPortSuffix:     googleSQLPostgresPort,
		prefix + googleSQLDatabaseSuffix: in.DB.Name,
		prefix + googleSQLUsernameSuffix: in.Name,
		prefix + GoogleSQLPasswordSuffix: password,
		prefix + googleSQLURLSuffix:      fmt.Sprintf(googleSQLPostgresURL, in.Name, password, googleSQLPostgresHost, googleSQLPostgresPort, in.DB.Name),
		prefix + googleSQLJDBCURLSuffix:  fmt.Sprintf(googleSQLPostgresJDBCURL, googleSQLPostgresHost, googleSQLPostgresPort, in.DB.Name, in.Name, password),
	}
}

func (in GoogleSqlUser) create(objectMeta metav1.ObjectMeta, appName string) (*googlesqlcrd.SQLUser, error) {
	secretName, err := GoogleSQLSecretName(appName, in.Instance.Name, in.DB.Name, in.Name)
	if err != nil {
		return nil, err
	}

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
			ResourceID: in.Name,
		},
	}, nil
}

func (in GoogleSqlUser) Create(objectMeta metav1.ObjectMeta, cascadingDelete bool, appName, projectId string) (*googlesqlcrd.SQLUser, error) {
	if in.isDefault() {
		objectMeta.Name = in.Instance.Name
	} else {
		objectMeta.Name = fmt.Sprintf("%s-%s-%s", appName, in.DB.Name, replaceToLowerWithNoPrefix(in.Name))
	}
	setAnnotations(objectMeta, cascadingDelete, projectId)

	sqlUser, err := in.create(objectMeta, appName)
	if err != nil {
		return nil, err
	}

	return sqlUser, nil
}
