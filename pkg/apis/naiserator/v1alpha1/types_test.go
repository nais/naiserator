package v1alpha1_test

import (
	"testing"

	"github.com/nais/naiserator/pkg/apis/naiserator/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplication_Hash(t *testing.T) {
	a1, err := v1alpha1.Application{Spec: v1alpha1.ApplicationSpec{Team: "banan"}}.Hash()
	a2, _ := v1alpha1.Application{Spec: v1alpha1.ApplicationSpec{Team: "banan"}, ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"a": "b"}}}.Hash()
	a3, _ := v1alpha1.Application{Spec: v1alpha1.ApplicationSpec{Team: "banan"}, ObjectMeta: v1.ObjectMeta{Labels: map[string]string{"a": "b"}}}.Hash()

	assert.NoError(t, err)
	assert.Equal(t, a1, a2, "matches, as annotations is ignored")
	assert.NotEqual(t, a2, a3, "must not match ")
}
