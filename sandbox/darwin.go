//go:build darwin

package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// * if in macOS, always be true
func CheckDependence() error {
	return nil
}

func tempRoots() []string {
	seen := map[string]bool{}
	var out []string
	for _, one := range []string{os.TempDir(), "/tmp", "/var/tmp"} {
		if one == "" {
			continue
		}
		candidates := []string{one}
		if resolved, err := filepath.EvalSymlinks(one); err == nil {
			candidates = append(candidates, resolved)
		}
		for _, candidate := range candidates {
			candidate = filepath.Clean(candidate)
			if candidate == "" || candidate == string(filepath.Separator) || seen[candidate] {
				continue
			}
			seen[candidate] = true
			out = append(out, candidate)
		}
	}
	return out
}

func seatbeltProfile(home, workDir string, opt *Option) string {
	deniedDirs, deniedFiles := deniedPaths(home)

	var deny strings.Builder
	for _, d := range deniedDirs {
		fmt.Fprintf(&deny, "(deny file-read* (subpath %q))\n", d)
		fmt.Fprintf(&deny, "(deny file-write* (subpath %q))\n", d)
	}
	for _, f := range deniedFiles {
		fmt.Fprintf(&deny, "(deny file-read* (literal %q))\n", f)
		fmt.Fprintf(&deny, "(deny file-write* (literal %q))\n", f)
	}

	keychainDir := filepath.Join(home, "Library", "Keychains")

	networkRule := "(allow network*)"
	if opt.Network == NetworkDeny {
		networkRule = "(deny network*)"
	}

	var tempWrites strings.Builder
	for _, one := range tempRoots() {
		fmt.Fprintf(&tempWrites, "(allow file-write* (subpath %q))\n", one)
	}

	writeRoot := home
	var extraWrites strings.Builder
	if opt.MinimalBinds != nil {
		if opt.MinimalBinds.WriteScope == WriteHome {
			writeRoot = home
		} else {
			writeRoot = workDir
		}
		for _, p := range resolveBinds(opt.MinimalBinds.ReadWrite) {
			fmt.Fprintf(&extraWrites, "(allow file-write* (subpath %q))\n", p)
		}
	}

	return fmt.Sprintf(`(version 1)
(deny default)
(allow process-exec)
(allow process-fork)
(allow sysctl-read)
(allow mach-lookup)
(allow signal)
(allow ipc-posix*)

;; baseline: read-only filesystem
(allow file-read*)

;; baseline: writable scope
(allow file-write*
    (subpath %q))
%s
;; baseline: system temp directories
%s
;; baseline: keychain access (required for keyring/Security framework)
(allow file-read* (subpath %q))
(allow file-write* (subpath %q))

;; baseline: /dev access (required for /dev/null, /dev/random, etc.)
(allow file-read* (subpath "/dev"))
(allow file-write* (subpath "/dev"))

;; caller-specified denies (override baselines above)
%s
;; network
%s
`, writeRoot, extraWrites.String(), tempWrites.String(), keychainDir, keychainDir, deny.String(), networkRule)
}

func wrap(ctx context.Context, binary string, args []string, workDir string, opt *Option) (*exec.Cmd, error) {
	if ctx == nil {
		return nil, fmt.Errorf("ctx is nil")
	}
	if opt == nil {
		opt = &Option{}
	}
	if opt.CPUPercent < 0 || opt.MemoryMB < 0 {
		return nil, fmt.Errorf("negative limit: cpu=%d mem=%d", opt.CPUPercent, opt.MemoryMB)
	}
	if opt.MemoryMB > 0 {
		return nil, fmt.Errorf("MemoryMB not supported on darwin: RLIMIT_AS conflicts with Go runtime's virtual address reservation")
	}

	if opt.AllowAll {
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = resolveDir(workDir)
		return cmd, nil
	}

	homeDir, absWorkDir, err := validateDir(workDir)
	if err != nil {
		return nil, err
	}

	profile := seatbeltProfile(homeDir, absWorkDir, opt)
	sbArgs := []string{"-p", profile, binary}
	sbArgs = append(sbArgs, args...)

	cmd := exec.CommandContext(ctx, "sandbox-exec", sbArgs...)
	cmd.Dir = absWorkDir
	return cmd, nil
}
