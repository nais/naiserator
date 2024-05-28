package google_sql_test

import (
	"testing"

	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	googlesqlcrd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	google_sql "github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGoogleSQLUserEnvVars(t *testing.T) {
	expected := map[string]string{
		"NAIS_DATABASE_FOO_BAR_HOST":     "127.0.0.1",
		"NAIS_DATABASE_FOO_BAR_PORT":     "5432",
		"NAIS_DATABASE_FOO_BAR_DATABASE": "bar",
		"NAIS_DATABASE_FOO_BAR_USERNAME": "foo",
		"NAIS_DATABASE_FOO_BAR_PASSWORD": "password",
		"NAIS_DATABASE_FOO_BAR_URL":      "postgres://foo:password@127.0.0.1:5432/bar",
		"NAIS_DATABASE_FOO_BAR_JDBC_URL": "jdbc:postgres://127.0.0.1:5432/bar?user=foo&password=password",
	}

	instance := &googlesqlcrd.SQLInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}

	db := &nais.CloudSqlDatabase{
		Name: "bar",
	}

	sqlUser := google_sql.GoogleSqlUser{
		Name:     instance.Name,
		DB:       db,
		Instance: instance,
	}
	vars := sqlUser.CreateUserEnvVars("password")

	assert.Equal(t, expected, vars)
}

func TestGoogleSQLSecretEnvVarsWithAdditionalSqlUsers(t *testing.T) {
	instance := &googlesqlcrd.SQLInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}

	db := &nais.CloudSqlDatabase{
		Name:         "bar",
		EnvVarPrefix: "YOLO",
	}

	sqlUsers := []nais.CloudSqlDatabaseUser{
		{
			Name: instance.Name,
		},
		{
			Name: "user-two",
		},
	}

	expectedDefault := map[string]string{
		"YOLO_PASSWORD": "password",
		"YOLO_URL":      "postgres://foo:password@127.0.0.1:5432/bar",
		"YOLO_JDBC_URL": "jdbc:postgres://127.0.0.1:5432/bar?user=foo&password=password",
		"YOLO_USERNAME": "foo",
		"YOLO_HOST":     "127.0.0.1",
		"YOLO_PORT":     "5432",
		"YOLO_DATABASE": "bar",
	}

	result := make(map[string]string)
	defaultUser := google_sql.GoogleSqlUser{
		Name:     sqlUsers[0].Name,
		DB:       db,
		Instance: instance,
	}
	vars := defaultUser.CreateUserEnvVars("password")
	result = google_sql.MapEnvToVars(vars, result)

	assert.Equal(t, expectedDefault, result)

	expectedUserTwo := map[string]string{
		"YOLO_USER_TWO_USERNAME": "user-two",
		"YOLO_USER_TWO_PASSWORD": "password",
		"YOLO_USER_TWO_URL":      "postgres://user-two:password@127.0.0.1:5432/bar",
		"YOLO_USER_TWO_JDBC_URL": "jdbc:postgres://127.0.0.1:5432/bar?user=user-two&password=password",
		"YOLO_USER_TWO_HOST":     "127.0.0.1",
		"YOLO_USER_TWO_PORT":     "5432",
		"YOLO_USER_TWO_DATABASE": "bar",
	}

	result = make(map[string]string)
	userTwo := google_sql.GoogleSqlUser{
		Name:     sqlUsers[1].Name,
		DB:       db,
		Instance: instance,
	}
	vars = userTwo.CreateUserEnvVars("password")
	result = google_sql.MapEnvToVars(vars, result)

	assert.Equal(t, expectedUserTwo, result)
}

func TestMergeDefaultSQLUser(t *testing.T) {
	instance := &googlesqlcrd.SQLInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}

	dbUsers := []nais.CloudSqlDatabaseUser{
		{
			Name: "user-two",
		},
		{
			Name: "user_three",
		},
		{
			Name: "user_three",
		},
		{
			Name: instance.Name,
		},
		{
			Name: instance.Name,
		},
	}

	expected := []nais.CloudSqlDatabaseUser{
		{
			Name: "user-two",
		},
		{
			Name: "user_three",
		},
		{
			Name: instance.Name,
		},
	}

	mergedUsers, err := google_sql.MergeAndFilterDatabaseSQLUsers(nil, instance.Name)
	assert.NoError(t, err)
	assert.Equal(t, []nais.CloudSqlDatabaseUser{{Name: instance.Name}}, mergedUsers)

	mergedUsers, err = google_sql.MergeAndFilterDatabaseSQLUsers(dbUsers, instance.Name)
	assert.NoError(t, err)
	assert.Equal(t, expected, mergedUsers)
}
