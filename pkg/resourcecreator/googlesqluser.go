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
			toUpperName := strings.ToUpper(in.Name)
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
	if in.prefixIsSet() && in.IsDefault() {
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
	prefix := in.googleSQLPrefix()
	if in.prefixIsSet() && !in.IsDefault() {
		prefix = fmt.Sprintf("%s_%s", prefix, googleSQLDatabaseCase(in.Name))
	}
	return prefix
}

func (in GoogleSqlUser) uniqueObjectName() (string, error) {
	if in.IsDefault() {
		return in.Instance.Name, nil
	}

	baseName := fmt.Sprintf("%s-%s-%s", in.Instance.Name, in.Instance.Namespace, in.Name)
	shortName, err := namegen.ShortName(baseName, maxLengthShortName)
	if err != nil {
		return "", err
	}

	return shortName, nil
}

func (in GoogleSqlUser) IsDefault() bool {
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

func (in GoogleSqlUser) GoogleSQLCommonEnvVars() map[string]string {
	var prefix string

	prefix = in.googleSQLPrefix()

	return map[string]string{
		prefix + googleSQLHostSuffix:     googleSQLPostgresHost,
		prefix + googleSQLPortSuffix:     googleSQLPostgresPort,
		prefix + googleSQLDatabaseSuffix: in.DB.Name,
	}
}

func (in GoogleSqlUser) googleSQLPrefix() string {
	if in.prefixIsSet() {
		return strings.TrimSuffix(in.DB.EnvVarPrefix, "_")
	}
	return fmt.Sprintf("NAIS_DATABASE_%s_%s", googleSQLDatabaseCase(in.Name), googleSQLDatabaseCase(in.DB.Name))
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

func (in GoogleSqlUser) prefixIsSet() bool {
	return len(in.DB.EnvVarPrefix) > 0
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
