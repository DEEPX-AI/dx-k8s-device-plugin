// Package nfd publishes DX-M1 metadata as a node-feature-discovery "local"
// feature file, which the NFD worker turns into node labels — fw/driver
// version labels without an extra DaemonSet or RBAC (the device plugin
// already runs on every NPU node).
//
// NFD's local source accepts explicitly-prefixed label names (deepx.ai/...)
// as key=value lines. The target dir is NFD's features.d, hostPath-mounted by
// the dx-npu chart.
package nfd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DEEPX-AI/dx-k8s-device-plugin/internal/dxdevice"
)

// FileName is our feature file under NFD's features.d directory.
const FileName = "deepx"

// labelInvalid matches runs of characters a Kubernetes label value may not hold.
var labelInvalid = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// labelValue makes s a valid label value (<=63 chars, alphanumeric at both
// ends). NFD silently drops a label whose value fails validation.
func labelValue(s string) string {
	s = labelInvalid.ReplaceAllString(s, "_")
	if len(s) > 63 {
		s = s[:63]
	}
	return strings.Trim(s, "_.-")
}

// Write renders the feature file atomically (temp+rename, same pattern as
// cdi.Write). Version/product labels come from the first device — cards in
// one host share driver/fw in practice.
func Write(devs []dxdevice.Device, dir string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "deepx.ai/dx-m1.count=%d\n", len(devs))
	if len(devs) > 0 {
		d := devs[0]
		// FW >= 2.7.6 reports "v2.7.6 (build: 4.90f4a6f)"; the label keeps
		// only the version, which is what docs and nodeSelectors use.
		fw := ""
		if f := strings.Fields(d.FWVersion); len(f) > 0 {
			fw = f[0]
		}
		fmt.Fprintf(&b, "deepx.ai/dx-m1.product=%s\n", labelValue(d.Product))
		fmt.Fprintf(&b, "deepx.ai/dx-m1.fw-version=%s\n", labelValue(fw))
		fmt.Fprintf(&b, "deepx.ai/dx-m1.driver-version=%s\n", labelValue(d.RTDriver))
		fmt.Fprintf(&b, "deepx.ai/dx-m1.pcie-driver-version=%s\n", labelValue(d.PCIeDriver))
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".deepx-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	// CreateTemp defaults to 0600; the NFD worker runs as a different user and
	// must be able to read the file.
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if _, err := tmp.WriteString(b.String()); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	return os.Rename(name, filepath.Join(dir, FileName))
}
