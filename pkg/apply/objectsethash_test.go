package apply

import (
	"crypto/fips140"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const wantObjectSetHash = "0b1b4822889b77a3dc829ce5e8d2260128e530e8"

var objectSetLabels = map[string]string{
	LabelID:        "addon-coredns",
	LabelGVK:       "/v1, Kind=ConfigMap",
	LabelName:      "coredns",
	LabelNamespace: "kube-system",
	LabelPrune:     "true",
}

func TestObjectSetHash_SHA1UnderFIPSOnly(t *testing.T) {
	if os.Getenv("OBJECTSETHASH_CHILD") == "" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestObjectSetHash_SHA1UnderFIPSOnly$")
		cmd.Env = append(os.Environ(), "OBJECTSETHASH_CHILD=1", "GODEBUG=fips140=only")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return
	}
	require.True(t, fips140.Enforced())
	assert.Equal(t, wantObjectSetHash, objectSetHash(objectSetLabels))
}

func TestObjectSetHash_SHA1WithFIPSOff(t *testing.T) {
	if fips140.Enforced() {
		t.Skip("covered by the fips140=only subprocess")
	}
	assert.Equal(t, wantObjectSetHash, objectSetHash(objectSetLabels))
}
