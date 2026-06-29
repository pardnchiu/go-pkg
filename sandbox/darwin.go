//go:build darwin

package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// * if in macOS, always be true
func CheckDependence() error {
	return nil
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

	writeRoot := home
	var extraWrites strings.Builder
	if opt.MinimalBinds != nil {
		if opt.MinimalBinds.WriteScope == WriteHome {
			writeRoot = home
		} else {
			writeRoot = workDir
		}
		for _, p := range opt.MinimalBinds.ReadWrite {
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
`, writeRoot, extraWrites.String(), keychainDir, keychainDir, deny.String(), networkRule)
}

func Wrap(ctx context.Context, binary string, args []string, workDir string, opt *Option) (*exec.Cmd, error) {
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
		absDir, err := resolveDir(workDir)
		if err != nil {
			return nil, err
		}
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = absDir
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
