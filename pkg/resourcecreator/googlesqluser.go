package resourcecreator

import (
	"fmt"
	"strings"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	googlesqlcrd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/liberator/pkg/namegen"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	googleSQLHostSuffix     = "_HOST"
	googleSQLPortSuffix     = "_PORT"
	googleSQLUsernameSuffix = "_USERNAME"
	googleSQLPasswordSuffix = "_PASSWORD"
	googleSQLDatabaseSuffix = "_DATABASE"
	googleSQLURLSuffix      = "_URL"

	googleSQLPostgresHost = "127.0.0.1"
	googleSQLPostgresPort = "5432"
	googleSQLPostgresURL  = "postgres://%s:%s@%s:%s/%s"

	maxLengthShortName = 63
)

type GoogleSqlUser struct {
	Name     string
	DB       *nais.CloudSqlDatabase
	Instance *googlesqlcrd.SQLInstance
}

func SetupNewGoogleSqlUser(name string, db *nais.CloudSqlDatabase, instance *googlesqlcrd.SQLInstance) GoogleSqlUser {
	return GoogleSqlUser{
		Name:     name,
		DB:       db,
		Instance: instance,
	}
}

func (in GoogleSqlUser) KeyWithSuffixMatchingUser(vars map[string]string, suffix string) (string, error) {
	for k := range vars {
		if strings.HasSuffix(k, suffix) {
			toUpperName := googleSQLDatabaseCase(in.Name)
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

func (in GoogleSqlUser) filterDefaultUserKey(key string, suffix string) string {
	if in.prefixIsSet() && in.isDefault() {
		prefix := in.googleSqlUserPrefix()
		noPrefixSubstring := strings.Replace(key, prefix, "", -1)
		if noPrefixSubstring == suffix {
			return key
		}
	}
	return ""
}

func (in GoogleSqlUser) CreateUserEnvVars(password string) map[string]string {
	var prefix string

	prefix = in.googleSqlUserPrefix()

	return map[string]string{
		prefix + googleSQLHostSuffix:     googleSQLPostgresHost,
		prefix + googleSQLPortSuffix:     googleSQLPostgresPort,
		prefix + googleSQLDatabaseSuffix: in.DB.Name,
		prefix + googleSQLUsernameSuffix: in.Name,
		prefix + googleSQLPasswordSuffix: password,
		prefix + googleSQLURLSuffix:      fmt.Sprintf(googleSQLPostgresURL, in.Name, password, googleSQLPostgresHost, googleSQLPostgresPort, in.DB.Name),
	}
}

func (in GoogleSqlUser) googleSqlUserPrefix() string {
	prefix := in.sqlUserEnvPrefix()
	if in.prefixIsSet() && !in.isDefault() {
		prefix = fmt.Sprintf("%s_%s", prefix, googleSQLDatabaseCase(in.Name))
	}
	return prefix
}

func (in GoogleSqlUser) sqlUserEnvPrefix() string {
	if in.prefixIsSet() {
		return strings.TrimSuffix(in.DB.EnvVarPrefix, "_")
	}
	return fmt.Sprintf("NAIS_DATABASE_%s_%s", googleSQLDatabaseCase(in.Name), googleSQLDatabaseCase(in.DB.Name))
}

func (in GoogleSqlUser) prefixIsSet() bool {
	return len(in.DB.EnvVarPrefix) > 0
}

func (in GoogleSqlUser) isDefault() bool {
	return in.Instance.Name == in.Name
}

func (in GoogleSqlUser) Create(app *nais.Application, secretKeyRefEnvName string, cascadingDelete bool, projectId string) (*googlesqlcrd.SQLUser, error) {
	objectDataName, err := in.uniqueObjectName()
	if err != nil {
		return nil, fmt.Errorf("unable to create meatadata: %s", err)
	}
	objectMetadata := app.CreateObjectMetaWithName(objectDataName)
	setAnnotations(objectMetadata, cascadingDelete, projectId)
	return in.create(app, objectMetadata, secretKeyRefEnvName), nil
}

func (in GoogleSqlUser) uniqueObjectName() (string, error) {
	if in.isDefault() {
		return in.Instance.Name, nil
	}
	baseName := fmt.Sprintf("%s-%s", in.Instance.Name, in.Name)
	return namegen.ShortName(baseName, maxLengthShortName)
}

func (in GoogleSqlUser) create(app *nais.Application, objectMeta k8smeta.ObjectMeta, secretKeyRefEnvName string) *googlesqlcrd.SQLUser {
	return &googlesqlcrd.SQLUser{
		TypeMeta: k8smeta.TypeMeta{
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
						Name: GoogleSQLSecretName(app, in.Instance.Name, in.Name),
					},
				},
			},
		},
	}
}

func setAnnotations(objectMeta k8smeta.ObjectMeta, cascadingDelete bool, projectId string) {
	setAnnotation(&objectMeta, GoogleProjectIdAnnotation, projectId)
	if !cascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		setAnnotation(&objectMeta, GoogleDeletionPolicyAnnotation, GoogleDeletionPolicyAbandon)
	}
}

func GoogleSQLSecretName(app *nais.Application, instanceName string, sqlUserName string) string {
	if isDefault(instanceName, sqlUserName) {
		return fmt.Sprintf("google-sql-%s", app.Name)
	}
	return fmt.Sprintf("google-sql-%s-%s", app.Name, sqlUserName)
}

func isDefault(instanceName string, sqlUserName string) bool {
	return instanceName == sqlUserName
}

func googleSQLDatabaseCase(x string) string {
	return strings.ReplaceAll(strings.ToUpper(x), "-", "_")
}

func MergeAndFilterSQLUsers(dbUsers []nais.CloudSqlDatabaseUser, instanceName string) []nais.CloudSqlDatabaseUser {
	defaultUser := nais.CloudSqlDatabaseUser{Name: instanceName}

	if dbUsers == nil {
		return []nais.CloudSqlDatabaseUser{defaultUser}
	}

	return removeDuplicates(append(dbUsers, defaultUser))

}

func removeDuplicates(dbUsers []nais.CloudSqlDatabaseUser) []nais.CloudSqlDatabaseUser {
	keys := make(map[string]bool)
	var set []nais.CloudSqlDatabaseUser

	for _, user := range dbUsers {
		filteredUser := ignoreCaseAndReplaceFilter(user.Name)
		if _, value := keys[filteredUser]; !value {
			keys[user.Name] = true
			set = append(set, user)
		}
	}
	return set
}

func ignoreCaseAndReplaceFilter(x string) string {
	return strings.ToLower(strings.ReplaceAll(x, "_", "-"))
}

func MapEnvToVars(env map[string]string, vars map[string]string) map[string]string {
	for k, v := range env {
		vars[k] = v
	}
	return vars
}
