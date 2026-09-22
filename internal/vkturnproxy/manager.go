package vkturnproxy

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/config"
)

const Version = "v1.8.3"

const (
	linuxAMD64SHA256 = "dfe4c9602d6cc21da0f8c5fc289b75ecfcc7e4862d53a7c7aced37e2a115c42e"
	linuxARM64SHA256 = "5b51916deb83f361ea6544f8cbb9f72f2db552ff0acbadff3797862793fc0472"
)

// Spec connects one public UDP listener to a local WireGuard inbound.
type Spec struct {
	InboundID int    `json:"inboundId"`
	Listen    string `json:"listen"`
	Connect   string `json:"connect"`
}

type Status struct {
	Spec    Spec   `json:"spec"`
	Running bool   `json:"running"`
	Error   string `json:"error,omitempty"`
}

type process struct {
	cmd  *exec.Cmd
	done chan struct{}

	mu      sync.Mutex
	exitErr error
	stopped bool
}

func (p *process) running() bool {
	select {
	case <-p.done:
		return false
	default:
		return true
	}
}

func (p *process) errorText() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.exitErr != nil {
		return p.exitErr.Error()
	}
	return ""
}

func (p *process) stop() {
	p.mu.Lock()
	p.stopped = true
	p.mu.Unlock()
	if !p.running() {
		return
	}
	_ = p.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-p.done:
	case <-time.After(3 * time.Second):
		_ = p.cmd.Process.Kill()
		<-p.done
	}
}

type managed struct {
	spec Spec
	proc *process
	err  string
}

// Manager owns only processes started by this instance. Call Reconcile again
// after a process exit to restart it; call StopAll when the panel shuts down.
type Manager struct {
	mu           sync.Mutex
	items        map[int]*managed
	binaryPath   string
	expectedHash string
}

func BinaryName() string {
	return "vk-turn-proxy-linux-" + runtime.GOARCH
}

func BinaryPath() string {
	return filepath.Join(config.GetBinFolderPath(), BinaryName())
}

func NewManager() *Manager {
	hash := ""
	if runtime.GOOS == "linux" {
		switch runtime.GOARCH {
		case "amd64":
			hash = linuxAMD64SHA256
		case "arm64":
			hash = linuxARM64SHA256
		}
	}
	return &Manager{items: make(map[int]*managed), binaryPath: BinaryPath(), expectedHash: hash}
}

func validateAddress(address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	if host != "" && net.ParseIP(host) == nil {
		return fmt.Errorf("host %q must be an IP address", host)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid UDP port %q", portText)
	}
	return nil
}

func validateSpec(spec Spec) error {
	if spec.InboundID <= 0 {
		return errors.New("inbound ID must be positive")
	}
	if err := validateAddress(spec.Listen); err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	if err := validateAddress(spec.Connect); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	return nil
}

func verifyBinary(path, expectedHash string) error {
	if expectedHash == "" {
		return fmt.Errorf("vk-turn-proxy %s has no pinned binary for this platform", Version)
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("%s is not an executable regular file", path)
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != expectedHash {
		return fmt.Errorf("vk-turn-proxy %s SHA-256 mismatch: got %s", Version, got)
	}
	return nil
}

func start(path string, spec Spec) (*process, error) {
	cmd := exec.Command(path, "-listen", spec.Listen, "-connect", spec.Connect)
	attachChildLifetime(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &process{cmd: cmd, done: make(chan struct{})}
	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		if !p.stopped {
			if err == nil {
				err = errors.New("vk-turn-proxy exited unexpectedly")
			}
			p.exitErr = err
		}
		p.mu.Unlock()
		close(p.done)
	}()
	return p, nil
}

// Reconcile makes running processes match specs. Invalid or failed entries are
// retained in Statuses so the administrator can see why they are not running.
func (m *Manager) Reconcile(specs []Spec) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.items == nil {
		m.items = make(map[int]*managed)
	}
	wanted := make(map[int]Spec, len(specs))
	var errs []error
	for _, spec := range specs {
		if _, exists := wanted[spec.InboundID]; exists {
			errs = append(errs, fmt.Errorf("duplicate inbound ID %d", spec.InboundID))
			continue
		}
		wanted[spec.InboundID] = spec
	}
	for id, item := range m.items {
		if spec, keep := wanted[id]; keep && item.spec == spec {
			continue
		}
		if item.proc != nil {
			item.proc.stop()
		}
		delete(m.items, id)
	}
	for id, spec := range wanted {
		if item := m.items[id]; item != nil && item.proc != nil && item.proc.running() {
			continue
		}
		item := &managed{spec: spec}
		m.items[id] = item
		if err := validateSpec(spec); err != nil {
			item.err = err.Error()
			errs = append(errs, fmt.Errorf("inbound %d: %w", id, err))
			continue
		}
		if err := verifyBinary(m.binaryPath, m.expectedHash); err != nil {
			item.err = err.Error()
			errs = append(errs, fmt.Errorf("inbound %d: %w", id, err))
			continue
		}
		proc, err := start(m.binaryPath, spec)
		if err != nil {
			item.err = err.Error()
			errs = append(errs, fmt.Errorf("inbound %d: %w", id, err))
			continue
		}
		item.proc = proc
	}
	return errors.Join(errs...)
}

func (m *Manager) Statuses() []Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	statuses := make([]Status, 0, len(m.items))
	for _, item := range m.items {
		status := Status{Spec: item.spec, Error: item.err}
		if item.proc != nil {
			status.Running = item.proc.running()
			if !status.Running {
				status.Error = item.proc.errorText()
			}
		}
		statuses = append(statuses, status)
	}
	slices.SortFunc(statuses, func(a, b Status) int { return a.Spec.InboundID - b.Spec.InboundID })
	return statuses
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, item := range m.items {
		if item.proc != nil {
			item.proc.stop()
		}
		delete(m.items, id)
	}
}
