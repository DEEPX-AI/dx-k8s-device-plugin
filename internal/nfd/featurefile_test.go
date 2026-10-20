package nfd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/DEEPX-AI/dx-k8s-device-plugin/internal/dxdevice"
)

func TestWrite_LabelsFromDevices(t *testing.T) {
	dir := t.TempDir()
	devs := []dxdevice.Device{
		{ID: 0, FWVersion: "v2.7.3", RTDriver: "v2.5.1", PCIeDriver: "v2.4.1", Product: "M1", Healthy: true},
		{ID: 1, FWVersion: "v2.7.3", RTDriver: "v2.5.1", PCIeDriver: "v2.4.1", Product: "M1", Healthy: true},
	}
	if err := Write(devs, dir); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	// NFD worker runs as another user; 0600 (CreateTemp default) breaks it.
	if fi.Mode().Perm() != 0o644 {
		t.Errorf("feature file mode = %o, want 644", fi.Mode().Perm())
	}
	b, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{
		"deepx.ai/dx-m1.count=2\n",
		"deepx.ai/dx-m1.product=M1\n",
		"deepx.ai/dx-m1.fw-version=v2.7.3\n",
		"deepx.ai/dx-m1.driver-version=v2.5.1\n",
		"deepx.ai/dx-m1.pcie-driver-version=v2.4.1\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("feature file missing %q\n---\n%s", want, got)
		}
	}
}

// FW >= 2.7.6 reports "v2.7.6 (build: 4.90f4a6f)" in dxcli; NFD drops any
// label whose value fails the k8s label-value regex, so every value we write
// must be label-safe and fw-version must stay the bare version (docs/selectors).
func TestWrite_LabelSafeValues(t *testing.T) {
	dir := t.TempDir()
	devs := []dxdevice.Device{
		{ID: 0, FWVersion: "v2.7.6 (build: 4.90f4a6f)", RTDriver: "v2.7.0", PCIeDriver: "v2.7.0",
			Product: "M1 (rev: A)", Healthy: true},
	}
	if err := Write(devs, dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "deepx.ai/dx-m1.fw-version=v2.7.6\n") {
		t.Errorf("fw-version should be the bare version, got:\n%s", got)
	}
	valid := regexp.MustCompile(`^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$`)
	for _, ln := range strings.Split(strings.TrimSpace(got), "\n") {
		_, v, _ := strings.Cut(ln, "=")
		if len(v) > 63 || !valid.MatchString(v) {
			t.Errorf("invalid label value %q in line %q", v, ln)
		}
	}
}

func TestLabelValue(t *testing.T) {
	for in, want := range map[string]string{
		"v2.7.3":                       "v2.7.3",
		"M1 (rev: A)":                  "M1_rev_A",
		"(x)":                          "x",
		"":                             "",
		strings.Repeat("a", 70):        strings.Repeat("a", 63),
		strings.Repeat("a", 62) + "_b": strings.Repeat("a", 62),
	} {
		if got := labelValue(in); got != want {
			t.Errorf("labelValue(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWrite_NoDevices(t *testing.T) {
	dir := t.TempDir()
	if err := Write(nil, dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != "deepx.ai/dx-m1.count=0\n" {
		t.Errorf("zero-device file should only carry count=0, got:\n%s", got)
	}
}
