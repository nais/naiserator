package v1alpha1

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestApplication_Hash(t *testing.T) {
	a1,err := Application{Spec:ApplicationSpec{Team:"banan"}}.Hash()
	a2,_ := Application{Spec:ApplicationSpec{Team:"banan"}}.Hash()
	a3,_ := Application{Spec:ApplicationSpec{Team:"banana"}}.Hash()

	assert.NoError(t, err)
	assert.Equal(t, a1, a2, "should match")
	assert.NotEqual(t, a2, a3, "should not match")
}

