package resourcecreator_test

import (
	"testing"

	"github.com/magiconair/properties/assert"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	google_sql_crd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	"github.com/nais/naiserator/pkg/resourcecreator"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGoogleSQLEnvVars(t *testing.T) {
	expected := map[string]string{
		"NAIS_DATABASE_FOO_BAR_USERNAME": "username",
		"NAIS_DATABASE_FOO_BAR_PASSWORD": "password",
		"NAIS_DATABASE_FOO_BAR_HOST":     "127.0.0.1",
		"NAIS_DATABASE_FOO_BAR_PORT":     "5432",
		"NAIS_DATABASE_FOO_BAR_DATABASE": "bar",
		"NAIS_DATABASE_FOO_BAR_URL":      "postgres://username:password@127.0.0.1:5432/bar",
	}

	instance := &google_sql_crd.SQLInstance{
		ObjectMeta: v1.ObjectMeta{
			Name: "foo",
		},
	}

	db := &nais.CloudSqlDatabase{
		Name: "bar",
	}

	vars := resourcecreator.GoogleSQLEnvVars(db, instance.Name, "username", "password")

	assert.Equal(t, expected, vars)
}
