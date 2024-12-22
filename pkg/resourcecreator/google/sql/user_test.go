package google_sql_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	nais "github.com/nais/liberator/pkg/apis/nais.io/v1"
	googlesqlcrd "github.com/nais/liberator/pkg/apis/sql.cnrm.cloud.google.com/v1beta1"
	google_sql "github.com/nais/naiserator/pkg/resourcecreator/google/sql"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGoogleSQLUserEnvVars(t *testing.T) {
	instance := &googlesqlcrd.SQLInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}

	tests := []struct {
		name    string
		sqlUser google_sql.GoogleSqlUser
		want    map[string]string
	}{
		{
			name: "default user",
			sqlUser: google_sql.GoogleSqlUser{
				Username: instance.Name,
				AppName:  instance.Name,
				DB:       &nais.CloudSqlDatabase{Name: "bar"},
				Instance: instance,
			},
			want: map[string]string{
				"NAIS_DATABASE_FOO_BAR_HOST":     "127.0.0.1",
				"NAIS_DATABASE_FOO_BAR_PORT":     "5432",
				"NAIS_DATABASE_FOO_BAR_DATABASE": "bar",
				"NAIS_DATABASE_FOO_BAR_USERNAME": "foo",
				"NAIS_DATABASE_FOO_BAR_PASSWORD": "password",
				"NAIS_DATABASE_FOO_BAR_URL":      "postgres://foo:password@127.0.0.1:5432/bar",
				"NAIS_DATABASE_FOO_BAR_JDBC_URL": "jdbc:postgresql://127.0.0.1:5432/bar?user=foo&password=password",
			},
		},
		{
			name: "with env var prefix",
			sqlUser: google_sql.GoogleSqlUser{
				Username: instance.Name,
				AppName:  instance.Name,
				DB:       &nais.CloudSqlDatabase{Name: "bar", EnvVarPrefix: "YOLO"},
				Instance: instance,
			},
			want: map[string]string{
				"YOLO_PASSWORD": "password",
				"YOLO_URL":      "postgres://foo:password@127.0.0.1:5432/bar",
				"YOLO_JDBC_URL": "jdbc:postgresql://127.0.0.1:5432/bar?user=foo&password=password",
				"YOLO_USERNAME": "foo",
				"YOLO_HOST":     "127.0.0.1",
				"YOLO_PORT":     "5432",
				"YOLO_DATABASE": "bar",
			},
		},
		{
			name: "with env var prefix and non-instance username",
			sqlUser: google_sql.GoogleSqlUser{
				Username: "user-two",
				AppName:  instance.Name,
				DB:       &nais.CloudSqlDatabase{Name: "bar", EnvVarPrefix: "YOLO"},
				Instance: instance,
			},
			want: map[string]string{
				"YOLO_USER_TWO_PASSWORD": "password",
				"YOLO_USER_TWO_URL":      "postgres://user-two:password@127.0.0.1:5432/bar",
				"YOLO_USER_TWO_JDBC_URL": "jdbc:postgresql://127.0.0.1:5432/bar?user=user-two&password=password",
				"YOLO_USER_TWO_USERNAME": "user-two",
				"YOLO_USER_TWO_HOST":     "127.0.0.1",
				"YOLO_USER_TWO_PORT":     "5432",
				"YOLO_USER_TWO_DATABASE": "bar",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sqlUser.CreateUserEnvVars("password")
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("CreateUserEnvVars() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMergeDefaultSQLUser(t *testing.T) {
	instance := &googlesqlcrd.SQLInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}

	tests := []struct {
		name    string
		dbUsers []nais.CloudSqlDatabaseUser
		want    []nais.CloudSqlDatabaseUser
	}{
		{
			name:    "nil users",
			dbUsers: nil,
			want: []nais.CloudSqlDatabaseUser{
				{Name: instance.Name},
			},
		},
		{
			name: "no users",
			dbUsers: []nais.CloudSqlDatabaseUser{
				{Name: "user-two"},
				{Name: "user_three"},
				{Name: "user_three"},
				{Name: instance.Name},
				{Name: instance.Name},
			},

			want: []nais.CloudSqlDatabaseUser{
				{Name: "user-two"},
				{Name: "user_three"},
				{Name: instance.Name},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := google_sql.MergeAndFilterDatabaseSQLUsers(tt.dbUsers, instance.Name)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("MergeAndFilterDatabaseSQLUsers() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
