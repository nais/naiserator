package google_sql

import (
	"fmt"
	"strings"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/liberator/pkg/namegen"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/resourcecreator/pod"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	"github.com/nais/naiserator/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func setAnnotations(objectMeta metav1.ObjectMeta, cascadingDelete bool, projectId string) {
	util.SetAnnotation(&objectMeta, google.ProjectIdAnnotation, projectId)
	if !cascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		util.SetAnnotation(&objectMeta, google.DeletionPolicyAnnotation, google.DeletionPolicyAbandon)
	}
}

func GoogleSQLSecretName(appName, instanceName, dbName, sqlUserName string) (string, error) {
	if isDefault(instanceName, sqlUserName) {
		return fmt.Sprintf("google-sql-%s", appName), nil
	}
	return namegen.ShortName(fmt.Sprintf("google-sql-%s-%s-%s", appName, dbName, replaceToLowerWithNoPrefix(sqlUserName)), validation.DNS1035LabelMaxLength)
}

// isDefault is a legacy compatibility function
func isDefault(instanceName string, sqlUserName string) bool {
	return instanceName == sqlUserName
}

func googleSQLDatabaseCase(x string) string {
	return strings.ReplaceAll(strings.ToUpper(x), "-", "_")
}

func replaceToLowerWithNoPrefix(x string) string {
	noPrefixX := trimPrefix(x)
	return strings.ToLower(strings.ReplaceAll(noPrefixX, "_", "-"))
}

func trimPrefix(x string) string {
	return strings.TrimPrefix(x, "_")
}

func MergeAndFilterDatabaseSQLUsers(dbUsers []nais.CloudSqlDatabaseUser, instanceName string, dbNum int) ([]nais.CloudSqlDatabaseUser, error) {
	defaultUser := nais.CloudSqlDatabaseUser{Name: instanceName}

	if dbNum != 0 {
		if dbUsers != nil {
			return removeDuplicates(dbUsers), nil
		} else {
			return nil, fmt.Errorf("must to specify users for extra databases, can not have several databases with default user")
		}
	}

	if dbUsers == nil {
		return []nais.CloudSqlDatabaseUser{defaultUser}, nil
	}

	return removeDuplicates(append(dbUsers, defaultUser)), nil
}

func removeDuplicates(dbUsers []nais.CloudSqlDatabaseUser) []nais.CloudSqlDatabaseUser {
	keys := make(map[string]bool)
	var set []nais.CloudSqlDatabaseUser

	for _, user := range dbUsers {
		ignoreCaseUser := ignoreCase(user.Name)
		if _, value := keys[ignoreCaseUser]; !value {
			keys[user.Name] = true
			set = append(set, user)
		}
	}
	return set
}

func ignoreCase(x string) string {
	return strings.ToLower(x)
}

func MapEnvToVars(env map[string]string, vars map[string]string) map[string]string {
	for k, v := range env {
		vars[k] = v
	}
	return vars
}

func AppendGoogleSQLUserSecretEnvs(ast *resource.Ast, naisSqlInstance nais.CloudSqlInstance, sourceName string) error {
	for dbNum, db := range naisSqlInstance.Databases {

		googleSQLUsers, err := MergeAndFilterDatabaseSQLUsers(db.Users, naisSqlInstance.Name, dbNum)
		if err != nil {
			return err
		}

		for _, user := range googleSQLUsers {
			secretName, err := GoogleSQLSecretName(sourceName, naisSqlInstance.Name, db.Name, user.Name)
			if err != nil {
				return err
			}
			ast.EnvFrom = append(ast.EnvFrom, pod.EnvFromSecret(secretName))
		}
	}
	return nil
}

func BuildUniquesNameWithPredicate(predicate bool, defaultReturn, basename string) (string, error) {
	if predicate {
		return defaultReturn, nil
	}
	return namegen.ShortName(basename, validation.DNS1035LabelMaxLength)
}
