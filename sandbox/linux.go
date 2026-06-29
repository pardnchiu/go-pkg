//go:build linux

package sandbox

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// * if is nil, then install bubblewrap first
func CheckDependence() error {
	if _, err := exec.LookPath("bwrap"); err == nil {
		return nil
	}

	fmt.Println("please install bwrap first")

	var cmd *exec.Cmd
	switch {
	case checkBinary("apt-get"):
		cmd = exec.Command("sudo", "apt-get", "install", "-y", "bubblewrap")
	case checkBinary("dnf"):
		cmd = exec.Command("sudo", "dnf", "install", "-y", "bubblewrap")
	case checkBinary("yum"):
		cmd = exec.Command("sudo", "yum", "install", "-y", "bubblewrap")
	case checkBinary("pacman"):
		cmd = exec.Command("sudo", "pacman", "-S", "--noconfirm", "bubblewrap")
	case checkBinary("apk"):
		cmd = exec.Command("sudo", "apk", "add", "bubblewrap")
	default:
		return fmt.Errorf("os not supported")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cmd.Run: %w", err)
	}

	if _, err := exec.LookPath("bwrap"); err != nil {
		return fmt.Errorf("exec.LookPath")
	}

	return nil
}

func checkBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

var (
	bwrapOnce    sync.Once
	isAvailable  bool
	unshareFlags []string
)

func checkBwrap() {
	if exec.Command("bwrap", "--ro-bind", "/", "/", "--", "/bin/true").Run() != nil {
		return
	}
	isAvailable = true

	candidates := []string{
		"--unshare-user",
		"--unshare-pid",
		"--unshare-ipc",
		"--unshare-uts",
		"--unshare-cgroup",
	}
	for _, flag := range candidates {
		cmd := exec.Command("bwrap", "--ro-bind", "/", "/", flag, "--", "/bin/true")
		if cmd.Run() == nil {
			unshareFlags = append(unshareFlags, flag)
		} else {
			slog.Warn("bwrap unavailable",
				slog.String("flag", flag))
		}
	}
}

func buildBwrapArgs(homeDir, workDir string, opt *Option) []string {
	var args []string

	if opt.MinimalBinds != nil {
		for _, p := range minimalRoBinds {
			args = append(args, "--ro-bind-try", p, p)
		}
		for _, p := range opt.MinimalBinds.ReadOnly {
			args = append(args, "--ro-bind-try", p, p)
		}
		writeRoot := workDir
		if opt.MinimalBinds.WriteScope == WriteHome {
			writeRoot = homeDir
		}
		args = append(args, "--bind", writeRoot, writeRoot)
		for _, p := range opt.MinimalBinds.ReadWrite {
			args = append(args, "--bind", p, p)
		}
	} else {
		args = append(args,
			"--ro-bind", "/", "/",
			"--bind", homeDir, homeDir,
		)
	}

	args = append(args,
		"--tmpfs", "/tmp",
		"--dev", "/dev",
		"--proc", "/proc",
		"--new-session",
		"--die-with-parent",
	)

	if opt.Network == NetworkDeny {
		args = append(args, "--unshare-net")
	} else {
		args = append(args, "--share-net")
	}
	args = append(args, unshareFlags...)

	if opt.DropCaps {
		args = append(args, "--cap-drop", "ALL")
	}

	deniedDirs, deniedFiles := deniedPaths(homeDir)
	if opt.MinimalBinds != nil {
		writeRoot := workDir
		if opt.MinimalBinds.WriteScope == WriteHome {
			writeRoot = homeDir
		}
		roots := append([]string{writeRoot}, opt.MinimalBinds.ReadWrite...)
		isVisible := func(p string) bool {
			for _, r := range roots {
				if p == r || strings.HasPrefix(p, r+"/") {
					return true
				}
			}
			return false
		}
		deniedDirs = filterVisible(deniedDirs, isVisible)
		deniedFiles = filterVisible(deniedFiles, isVisible)
	}
	for _, d := range deniedDirs {
		args = append(args, "--tmpfs", d)
	}
	for _, f := range deniedFiles {
		args = append(args, "--ro-bind", "/dev/null", f)
	}
	return args
}

func filterVisible(paths []string, keep func(string) bool) []string {
	var out []string
	for _, p := range paths {
		if keep(p) {
			out = append(out, p)
		}
	}
	return out
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

	if opt.AllowAll {
		absDir, err := resolveDir(workDir)
		if err != nil {
			return nil, err
		}
		if opt.CPUPercent > 0 || opt.MemoryMB > 0 {
			if !checkBinary("systemd-run") {
				return nil, fmt.Errorf("systemd-run required for CPU/Memory limits (needs a running user systemd session)")
			}
			sdArgs := []string{"--user", "--scope", "--collect"}
			if opt.MemoryMB > 0 {
				sdArgs = append(sdArgs, fmt.Sprintf("--property=MemoryMax=%dM", opt.MemoryMB))
			}
			if opt.CPUPercent > 0 {
				sdArgs = append(sdArgs, fmt.Sprintf("--property=CPUQuota=%d%%", opt.CPUPercent))
			}
			sdArgs = append(sdArgs, binary)
			sdArgs = append(sdArgs, args...)
			cmd := exec.CommandContext(ctx, "systemd-run", sdArgs...)
			cmd.Dir = absDir
			return cmd, nil
		}
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = absDir
		return cmd, nil
	}

	homeDir, absWorkDir, err := validateDir(workDir)
	if err != nil {
		return nil, err
	}

	bwrapOnce.Do(func() {
		checkBwrap()
		if !isAvailable {
			slog.Warn("bwrap unavailable")
		}
	})

	if !isAvailable {
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = absWorkDir
		return cmd, nil
	}

	bwrapArgs := buildBwrapArgs(homeDir, absWorkDir, opt)
	bwrapArgs = append(bwrapArgs, "--", binary)
	bwrapArgs = append(bwrapArgs, args...)

	if opt.CPUPercent == 0 && opt.MemoryMB == 0 {
		cmd := exec.CommandContext(ctx, "bwrap", bwrapArgs...)
		cmd.Dir = absWorkDir
		return cmd, nil
	}

	if !checkBinary("systemd-run") {
		return nil, fmt.Errorf("systemd-run required for CPU/Memory limits (needs a running user systemd session)")
	}

	sdArgs := []string{"--user", "--scope", "--collect"}
	if opt.MemoryMB > 0 {
		sdArgs = append(sdArgs, fmt.Sprintf("--property=MemoryMax=%dM", opt.MemoryMB))
	}
	if opt.CPUPercent > 0 {
		sdArgs = append(sdArgs, fmt.Sprintf("--property=CPUQuota=%d%%", opt.CPUPercent))
	}
	sdArgs = append(sdArgs, "bwrap")
	sdArgs = append(sdArgs, bwrapArgs...)

	cmd := exec.CommandContext(ctx, "systemd-run", sdArgs...)
	cmd.Dir = absWorkDir
	return cmd, nil
}
