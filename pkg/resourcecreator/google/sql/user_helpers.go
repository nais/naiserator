package google_sql

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	"github.com/nais/naiserator/pkg/resourcecreator/google"
	"github.com/nais/naiserator/pkg/util"
)

func setAnnotations(objectMeta metav1.ObjectMeta, cascadingDelete bool, projectId string) {
	util.SetAnnotation(&objectMeta, google.ProjectIdAnnotation, projectId)
	if !cascadingDelete {
		// Prevent out-of-band objects from being deleted when the Kubernetes resource is deleted.
		util.SetAnnotation(&objectMeta, google.DeletionPolicyAnnotation, google.DeletionPolicyAbandon)
	}
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

func MergeAndFilterDatabaseSQLUsers(dbUsers []nais.CloudSqlDatabaseUser, instanceName string) ([]nais.CloudSqlDatabaseUser, error) {
	defaultUser := nais.CloudSqlDatabaseUser{Name: instanceName}

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
