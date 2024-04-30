package google_sql

import (
	"fmt"
	"strings"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	googlesqlcrd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
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

func SetupGoogleSqlUser(name string, db *nais.CloudSqlDatabase, instance *googlesqlcrd.SQLInstance) GoogleSqlUser {
	return GoogleSqlUser{
		Name:     name,
		DB:       db,
		Instance: instance,
	}
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

func (in GoogleSqlUser) KeyWithSuffixMatchingUser(vars map[string]string, suffix string) (string, error) {
	for k := range vars {
		if strings.HasSuffix(k, suffix) {
			toUpperName := googleSQLDatabaseCase(trimPrefix(in.Name))
			key := in.filterDefaultUserKey(k, suffix)
			if len(key) > 0 {
				return key, nil
			} else if strings.Contains(k, toUpperName) {
				return k, nil
			}
		}
	}
	return "", fmt.Errorf("no variable found matching suffix %s", suffix)
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

func (in GoogleSqlUser) create(objectMeta metav1.ObjectMeta, secretKeyRefEnvName, appName string) (*googlesqlcrd.SQLUser, error) {
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
						Key:  secretKeyRefEnvName,
						Name: secretName,
					},
				},
			},
			ResourceID: in.Name,
		},
	}, nil
}

func (in GoogleSqlUser) Create(objectMeta metav1.ObjectMeta, cascadingDelete bool, secretKeyRefEnvName, appName, projectId string) (*googlesqlcrd.SQLUser, error) {
	if in.isDefault() {
		objectMeta.Name = in.Instance.Name
	} else {
		objectMeta.Name = fmt.Sprintf("%s-%s-%s", appName, in.DB.Name, replaceToLowerWithNoPrefix(in.Name))
	}
	setAnnotations(objectMeta, cascadingDelete, projectId)

	sqlUser, err := in.create(objectMeta, secretKeyRefEnvName, appName)
	if err != nil {
		return nil, err
	}

	return sqlUser, nil
}
