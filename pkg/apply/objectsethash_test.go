package apply

import (
	"crypto/fips140"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/validation"
)

var objectSetLabels = map[string]string{
	LabelID:        "addon-coredns",
	LabelGVK:       "/v1, Kind=ConfigMap",
	LabelName:      "coredns",
	LabelNamespace: "kube-system",
	LabelPrune:     "true",
}

const (
	wantSHA1Hash   = "0b1b4822889b77a3dc829ce5e8d2260128e530e8"
	wantSHA256Hash = "8b03ed8e62d97b5646dc400df89bbe3160cb9256"
)

func setFIPS(t *testing.T, enabled bool) {
	t.Helper()
	if !enabled && fips140.Enforced() {
		t.Skip("SHA-1 is not available under GODEBUG=fips140=only")
	}
	prev := fipsEnabled
	fipsEnabled = func() bool { return enabled }
	t.Cleanup(func() { fipsEnabled = prev })
}

func TestObjectSetHash_KnownAnswer(t *testing.T) {
	t.Run("fips=false", func(t *testing.T) {
		setFIPS(t, false)
		assert.Equal(t, wantSHA1Hash, objectSetHash(objectSetLabels))
	})
	t.Run("fips=true", func(t *testing.T) {
		setFIPS(t, true)
		assert.Equal(t, wantSHA256Hash, objectSetHash(objectSetLabels))
	})
}

func TestObjectSetHash_IsAValidLabelValue(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "fips=false", true: "fips=true"}[enabled], func(t *testing.T) {
			setFIPS(t, enabled)
			got := objectSetHash(objectSetLabels)
			assert.Len(t, got, 40)
			assert.Empty(t, validation.IsValidLabelValue(got))
		})
	}
}

func TestObjectSetHash_OnlyOwnerLabelsCount(t *testing.T) {
	setFIPS(t, true)
	other := map[string]string{
		LabelID:        "addon-coredns",
		LabelGVK:       "/v1, Kind=ConfigMap",
		LabelName:      "coredns",
		LabelNamespace: "kube-system",
		LabelHash:      "ignored",
	}
	assert.Equal(t, objectSetHash(objectSetLabels), objectSetHash(other))
}

// Runs against the real mode, so it covers GODEBUG=fips140=on and =only.
func TestObjectSetHash_InProcessMode(t *testing.T) {
	want := wantSHA1Hash
	if fips140.Enabled() {
		want = wantSHA256Hash
	}
	assert.Equal(t, want, objectSetHash(objectSetLabels))
}
