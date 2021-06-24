package pod

import (
	"testing"

	nais_io_v1 "github.com/nais/liberator/pkg/apis/nais.io/v1"
	nais_io_v1alpha1 "github.com/nais/liberator/pkg/apis/nais.io/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestPod(t *testing.T) {
	t.Run("check if default lifecycle is used", func(t *testing.T) {
		lifecycle, err := lifecycle("", nil)
		assert.NoError(t, err)

		assert.Equal(t, []string{"sleep", "5"}, lifecycle.PreStop.Exec.Command)
	})

	t.Run("chech that not both preStopHookPath and preStopHook is used", func(t *testing.T) {
		preStopHook := &nais_io_v1.PreStopHook{}
		lifecycle, err := lifecycle("/my-path", preStopHook)
		assert.Error(t, err)
		assert.Nil(t, lifecycle)
	})

	t.Run("chech that not both preStopHook.exec and preStopHook.http is used", func(t *testing.T) {
		preStopHook := &nais_io_v1.PreStopHook{
			Exec: &nais_io_v1.ExecAction{
				Command: []string{"./hello", "world"},
			},
			Http: &nais_io_v1.HttpGetAction{
				Path: "/helloworld",
				Port: 1337,
			},
		}
		lifecycle, err := lifecycle("", preStopHook)
		assert.Error(t, err)
		assert.Nil(t, lifecycle)
	})

	t.Run("chech that preStopHook.http uses default port if not specified", func(t *testing.T) {
		preStopHook := &nais_io_v1.PreStopHook{
			Http: &nais_io_v1.HttpGetAction{
				Path: "/helloworld",
			},
		}
		lifecycle, err := lifecycle("", preStopHook)
		assert.NoError(t, err)
		assert.Equal(t, nais_io_v1alpha1.DefaultPortName, lifecycle.PreStop.HTTPGet.Port.String())
	})

	t.Run("chech that preStopHook.http uses specified port", func(t *testing.T) {

		preStopHook := &nais_io_v1.PreStopHook{
			Http: &nais_io_v1.HttpGetAction{
				Path: "/helloworld",
				Port: 1337,
			},
		}
		lifecycle, err := lifecycle("", preStopHook)
		assert.NoError(t, err)
		assert.Equal(t, 1337, lifecycle.PreStop.HTTPGet.Port.IntValue())
	})
}
