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

func SetupNewGoogleSqlUser(db *nais.CloudSqlDatabase, instance *googlesqlcrd.SQLInstance) GoogleSqlUser {
	return GoogleSqlUser{
		Name:     "",
		DB:       db,
		Instance: instance,
	}
}

func (in GoogleSqlUser) KeyWithSuffixMatchingUser(vars map[string]string, suffix string) (string, error) {
	for k := range vars {
		if strings.HasSuffix(k, suffix) {
			println(k)
			println(in.Name)
			toUpperName := strings.ToUpper(in.Name)
			println(toUpperName)
			println(strings.Contains(k, toUpperName))

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
	if prefixIsSet(in.DB.EnvVarPrefix) && in.isDefault() {
		prefix := in.getGoogleSecretPrefix()
		noPrefixSubstring := strings.Replace(key, prefix, "", -1)
		if noPrefixSubstring == suffix {
			return key
		}
	}
	return ""
}

func (in GoogleSqlUser) SecretEnvVars(password string) map[string]string {
	var prefix string

	prefix = in.getGoogleSecretPrefix()

	return map[string]string{
		prefix + googleSQLUsernameSuffix: in.Name,
		prefix + googleSQLPasswordSuffix: password,
		prefix + googleSQLURLSuffix:      fmt.Sprintf(googleSQLPostgresURL, in.Name, password, googleSQLPostgresHost, googleSQLPostgresPort, in.DB.Name),
	}
}

func (in GoogleSqlUser) getGoogleSecretPrefix() string {
	prefix := googleSQLPrefix(in.DB, in.Name)
	if prefixIsSet(in.DB.EnvVarPrefix) && !in.isDefault() {
		prefix = fmt.Sprintf("%s_%s", prefix, googleSQLDatabaseCase(in.Name))
	}
	return prefix
}

func (in GoogleSqlUser) uniqueObjectName() (string, error) {
	if in.isDefault() {
		return in.Instance.Name, nil
	}

	baseName := fmt.Sprintf("%s-%s-%s", in.Instance.Name, in.Instance.Namespace, in.Name)
	shortName, err := namegen.ShortName(baseName, maxLengthShortName)
	if err != nil {
		return "", err
	}

	return shortName, nil
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
	return createSQLUser(app, objectMetadata, in.Instance.Name, secretKeyRefEnvName), nil
}

func setAnnotations(objectMeta k8smeta.ObjectMeta, cascadingDelete bool, projectId string) {
	setAnnotation(&objectMeta, GoogleProjectIdAnnotation, projectId)
	if !cascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		setAnnotation(&objectMeta, GoogleDeletionPolicyAnnotation, GoogleDeletionPolicyAbandon)
	}
}

func GoogleSQLCommonEnvVars(db *nais.CloudSqlDatabase, sqlUser string) map[string]string {
	var prefix string

	prefix = googleSQLPrefix(db, sqlUser)

	return map[string]string{
		prefix + googleSQLHostSuffix:     googleSQLPostgresHost,
		prefix + googleSQLPortSuffix:     googleSQLPostgresPort,
		prefix + googleSQLDatabaseSuffix: db.Name,
	}
}

func googleSQLPrefix(db *nais.CloudSqlDatabase, sqlUser string) string {
	if prefixIsSet(db.EnvVarPrefix) {
		return strings.TrimSuffix(db.EnvVarPrefix, "_")
	}
	return fmt.Sprintf("NAIS_DATABASE_%s_%s", googleSQLDatabaseCase(sqlUser), googleSQLDatabaseCase(db.Name))
}

func googleSQLDatabaseCase(x string) string {
	return strings.ReplaceAll(strings.ToUpper(x), "-", "_")
}

func MergeStandardSQLUser(additionalUsers []nais.AdditionalUser, instanceName string) []nais.AdditionalUser {
	standardUser := nais.AdditionalUser{Name: instanceName}
	if additionalUsers == nil {
		return []nais.AdditionalUser{standardUser}
	}
	return append(additionalUsers, standardUser)
}

func prefixIsSet(envPrefix string) bool {
	return len(envPrefix) > 0
}

func MapEnvToVars(env map[string]string, vars map[string]string) map[string]string {
	for k, v := range env {
		vars[k] = v
	}
	return vars
}

func GoogleSQLSecretName(app *nais.Application) string {
	return fmt.Sprintf("google-sql-%s", app.Name)
}

func createSQLUser(app *nais.Application, objectMeta k8smeta.ObjectMeta, instanceName string, secretKeyRefEnvName string) *googlesqlcrd.SQLUser {
	return &googlesqlcrd.SQLUser{
		TypeMeta: k8smeta.TypeMeta{
			Kind:       "SQLUser",
			APIVersion: "sql.cnrm.cloud.google.com/v1beta1",
		},
		ObjectMeta: objectMeta,
		Spec: googlesqlcrd.SQLUserSpec{
			InstanceRef: googlesqlcrd.InstanceRef{Name: instanceName},
			Password: googlesqlcrd.SqlUserPasswordValue{
				ValueFrom: googlesqlcrd.SqlUserPasswordSecretKeyRef{
					SecretKeyRef: googlesqlcrd.SecretRef{
						Key:  secretKeyRefEnvName,
						Name: GoogleSQLSecretName(app),
					},
				},
			},
		},
	}
}
