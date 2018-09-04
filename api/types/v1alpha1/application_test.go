package v1alpha1

import (
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestApplication_Hash(t *testing.T) {
	a1, err := Application{Spec: ApplicationSpec{Team: "banan"}}.Hash()
	a2, _ := Application{Spec: ApplicationSpec{Team: "banan"}, ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"a": "b"}}}.Hash()
	a3, _ := Application{Spec: ApplicationSpec{Team: "banan"}, ObjectMeta: v1.ObjectMeta{Labels: map[string]string{"a": "b"}}}.Hash()

	assert.NoError(t, err)
	assert.Equal(t, a1, a2, "matches, as annotations is ignored")
	assert.NotEqual(t, a2, a3, "must not match ")
}
