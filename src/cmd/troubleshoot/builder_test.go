package troubleshoot

import (
	"github.com/Dynatrace/dynatrace-operator/src/cmd/troubleshoot/checks"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/scheme/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTroubleshootCommandBuilder(t *testing.T) {
	t.Run("build command", func(t *testing.T) {
		builder := NewTroubleshootCommandBuilder()
		csiCommand := builder.Build()

		assert.NotNil(t, csiCommand)
		assert.Equal(t, use, csiCommand.Use)
		assert.NotNil(t, csiCommand.RunE)
	})

	t.Run("getAllDynakubesInNamespace", func(t *testing.T) {
		dynakube := buildTestDynakube()
		clt := fake.NewClient(&dynakube)

		troubleshootCtx := TroubleshootContext{ApiReader: clt, Namespace: checks.testNamespace}

		dynakubes, err := getAllDynakubesInNamespace(troubleshootCtx)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(dynakubes))
		assert.Equal(t, dynakube.Name, dynakubes[0].Name)
	})

	t.Run("getDynakube - only check one dynakube if set", func(t *testing.T) {
		dynakube := buildTestDynakube()
		troubleshootCtx := TroubleshootContext{
			dynakube:  dynakube,
			Namespace: checks.testNamespace,
		}

		dynakubes, err := getDynakubes(troubleshootCtx, dynakube.Name)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(dynakubes))
		assert.Equal(t, checks.testDynakube, dynakubes[0].Name)
	})
}

func buildTestDynakube() dynatracev1beta1.DynaKube {
	return dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      checks.testDynakube,
			Namespace: checks.testNamespace,
		},
	}
}
