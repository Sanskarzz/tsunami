package isolation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
)

// Runtime runs only in a trusted diagnostic Job. Host-side CRI inspection must
// separately verify io.containerd.runsc.v1; a pod's self-report is not proof.
func Runtime(ctx context.Context, output io.Writer) error {
	if err := RuntimeSmoke(output); err != nil {
		return err
	}
	// Keep the sandbox alive briefly for an independent host-side CRI check.
	fmt.Fprintln(output, "READY for host runtime inspection")
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(15 * time.Second):
		return nil
	}
}

func RuntimeSmoke(output io.Writer) error {
	version, err := os.ReadFile("/proc/version")
	if err != nil {
		return err
	}
	fmt.Fprintf(output, "Kernel: %s", version)
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); !os.IsNotExist(err) {
		return errors.New("service-account token check failed")
	}
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_ICMP)
	if err == nil {
		_ = syscall.Close(fd)
		return errors.New("raw socket unexpectedly allowed")
	}
	if !errors.Is(err, syscall.EPERM) && !errors.Is(err, syscall.EACCES) {
		return fmt.Errorf("raw socket check inconclusive: %w", err)
	}
	f, err := os.Create("/tsunami-write-check")
	if err == nil {
		_ = f.Close()
		_ = os.Remove("/tsunami-write-check")
		return errors.New("root filesystem write unexpectedly allowed")
	}
	if !errors.Is(err, syscall.EROFS) && !errors.Is(err, syscall.EACCES) {
		return fmt.Errorf("filesystem write check inconclusive: %w", err)
	}
	fmt.Fprintln(output, "PASS no API token, raw socket denied, root filesystem write denied")
	return nil
}
