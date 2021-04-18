package resourcecreator_test

import (
	"testing"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	googlesqlcrd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGoogleSQLEnvVars(t *testing.T) {
	expected := map[string]string{
		"NAIS_DATABASE_FOO_BAR_HOST":     "127.0.0.1",
		"NAIS_DATABASE_FOO_BAR_PORT":     "5432",
		"NAIS_DATABASE_FOO_BAR_DATABASE": "bar",
	}

	instance := &googlesqlcrd.SQLInstance{
		ObjectMeta: v1.ObjectMeta{
			Name: "foo",
		},
	}

	db := &nais.CloudSqlDatabase{
		Name: "bar",
	}

	sqlUser := resourcecreator.SetupNewGoogleSqlUser(instance.Name, db, instance)
	vars := sqlUser.GoogleSQLCommonEnvVars()

	assert.Equal(t, expected, vars)
}

func TestGoogleSQLSecretEnvVars(t *testing.T) {
	expected := map[string]string{
		"NAIS_DATABASE_FOO_BAR_USERNAME": "foo",
		"NAIS_DATABASE_FOO_BAR_PASSWORD": "password",
		"NAIS_DATABASE_FOO_BAR_URL":      "postgres://foo:password@127.0.0.1:5432/bar",
	}

	instance := &googlesqlcrd.SQLInstance{
		ObjectMeta: v1.ObjectMeta{
			Name: "foo",
		},
	}

	db := &nais.CloudSqlDatabase{
		Name: "bar",
	}

	sqlUser := resourcecreator.SetupNewGoogleSqlUser(instance.Name, db, instance)
	vars := sqlUser.SecretEnvVars("password")

	assert.Equal(t, expected, vars)
}

func TestGoogleSQLSecretEnvVarsWithAdditionalSqlUsers(t *testing.T) {
	instance := &googlesqlcrd.SQLInstance{
		ObjectMeta: v1.ObjectMeta{
			Name: "foo",
		},
	}

	db := &nais.CloudSqlDatabase{
		Name:         "bar",
		EnvVarPrefix: "YOLO",
	}

	sqlUsers := []nais.AdditionalUser{
		{
			Name: instance.Name,
		},
		{
			Name: "additional",
		},
	}

	expected := map[string]string{
		"YOLO_PASSWORD":            "password",
		"YOLO_URL":                 "postgres://foo:password@127.0.0.1:5432/bar",
		"YOLO_USERNAME":            "foo",
		"YOLO_ADDITIONAL_USERNAME": "additional",
		"YOLO_ADDITIONAL_PASSWORD": "password",
		"YOLO_ADDITIONAL_URL":      "postgres://additional:password@127.0.0.1:5432/bar",
	}

	result := make(map[string]string)

	for _, sqlUser := range sqlUsers {
		googleSqlUser := resourcecreator.SetupNewGoogleSqlUser(sqlUser.Name, db, instance)
		vars := googleSqlUser.SecretEnvVars("password")
		result = resourcecreator.MapEnvToVars(vars, result)
	}

	assert.Equal(t, len(expected), len(result))
	assert.Equal(t, expected, result)
}

func TestKeyWithSuffixMatchingUser(t *testing.T) {
	instance := &googlesqlcrd.SQLInstance{
		ObjectMeta: v1.ObjectMeta{
			Name: "foo",
		},
	}

	db := &nais.CloudSqlDatabase{
		Name:         "bar",
		EnvVarPrefix: "YOLO",
	}

	sqlUsers := []nais.AdditionalUser{
		{
			Name: instance.Name,
		},
		{
			Name: "additional",
		},
	}

	envs := map[string]string{
		"YOLO_URL":                 "postgres://foo:password@127.0.0.1:5432/bar",
		"YOLO_USERNAME":            "foo",
		"YOLO_ADDITIONAL_USERNAME": "additional",
		"YOLO_ADDITIONAL_PASSWORD": "password",
		"YOLO_PASSWORD":            "password",
		"YOLO_ADDITIONAL_URL":      "postgres://additional:password@127.0.0.1:5432/bar",
	}

	googleSqlUser := resourcecreator.SetupNewGoogleSqlUser(sqlUsers[0].Name, db, instance)
	key, nil := googleSqlUser.KeyWithSuffixMatchingUser(envs, "_PASSWORD")
	assert.Nil(t, nil)
	assert.Equal(t, "YOLO_PASSWORD", key)

	googleSqlUser.Name = sqlUsers[1].Name
	key, nil = googleSqlUser.KeyWithSuffixMatchingUser(envs, "_PASSWORD")
	assert.Nil(t, nil)
	assert.Equal(t, "YOLO_ADDITIONAL_PASSWORD", key)
}
