package credentials

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	serviceName = "OpenTunnel CLI"
	accountName = "access-token"
)

var ErrNotFound = errors.New("credential not found")

type commandRunner func(name string, args ...string) ([]byte, error)

// Store prefers the native OS credential manager and falls back to a
// user-only file when no supported credential manager is available.
type Store struct {
	fallbackPath string
	goos         string
	run          commandRunner
}

func NewStore() (*Store, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("find user config directory: %w", err)
	}
	return NewStoreAt(filepath.Join(root, "opentunnel", "credentials"), runtime.GOOS), nil
}

func NewStoreAt(path, goos string) *Store {
	return &Store{
		fallbackPath: path,
		goos:         goos,
		run: func(name string, args ...string) ([]byte, error) {
			return exec.Command(name, args...).CombinedOutput()
		},
	}
}

func (s *Store) Get() (string, error) {
	token, err := s.getNative()
	if err == nil && token != "" {
		return token, nil
	}

	data, fileErr := os.ReadFile(s.fallbackPath)
	if errors.Is(fileErr, os.ErrNotExist) {
		return "", ErrNotFound
	}
	if fileErr != nil {
		return "", fmt.Errorf("read protected credentials: %w", fileErr)
	}
	token = strings.TrimSpace(string(data))
	if token == "" {
		return "", ErrNotFound
	}
	return token, nil
}

func (s *Store) Set(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("refusing to store an empty token")
	}
	if err := s.setNative(token); err == nil {
		_ = os.Remove(s.fallbackPath)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.fallbackPath), 0o700); err != nil {
		return fmt.Errorf("create credential directory: %w", err)
	}
	if err := os.Chmod(filepath.Dir(s.fallbackPath), 0o700); err != nil {
		return fmt.Errorf("protect credential directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(s.fallbackPath), ".credentials-*")
	if err != nil {
		return fmt.Errorf("create protected credential file: %w", err)
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("protect credential file: %w", err)
	}
	if _, err := temp.WriteString(token + "\n"); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write protected credentials: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync protected credentials: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close protected credentials: %w", err)
	}
	if err := os.Rename(tempPath, s.fallbackPath); err != nil {
		return fmt.Errorf("install protected credentials: %w", err)
	}
	return nil
}

func (s *Store) Delete() error {
	nativeErr := s.deleteNative()
	fileErr := os.Remove(s.fallbackPath)
	if errors.Is(fileErr, os.ErrNotExist) {
		fileErr = nil
	}
	if nativeErr != nil && fileErr != nil {
		return fmt.Errorf("delete credentials: native: %v; fallback: %w", nativeErr, fileErr)
	}
	return nil
}

func (s *Store) getNative() (string, error) {
	var output []byte
	var err error
	switch s.goos {
	case "darwin":
		output, err = s.run("security", "find-generic-password", "-s", serviceName, "-a", accountName, "-w")
	case "linux":
		output, err = s.run("secret-tool", "lookup", "service", serviceName, "account", accountName)
	case "windows":
		script := "$v=[Windows.Security.Credentials.PasswordVault,Windows.Security.Credentials,ContentType=WindowsRuntime]::new();$c=$v.Retrieve($args[0],$args[1]);$c.RetrievePassword();$c.Password"
		output, err = s.run("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script, serviceName, accountName)
	default:
		return "", errors.New("unsupported credential manager")
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *Store) setNative(token string) error {
	switch s.goos {
	case "darwin":
		_, err := s.run("security", "add-generic-password", "-U", "-s", serviceName, "-a", accountName, "-w", token)
		return err
	case "linux":
		command := exec.Command("secret-tool", "store", "--label", serviceName, "service", serviceName, "account", accountName)
		command.Stdin = strings.NewReader(token)
		_, err := command.CombinedOutput()
		return err
	case "windows":
		script := "$v=[Windows.Security.Credentials.PasswordVault,Windows.Security.Credentials,ContentType=WindowsRuntime]::new();try{$o=$v.Retrieve($args[0],$args[1]);$v.Remove($o)}catch{};$c=[Windows.Security.Credentials.PasswordCredential,Windows.Security.Credentials,ContentType=WindowsRuntime]::new($args[0],$args[1],$args[2]);$v.Add($c)"
		_, err := s.run("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script, serviceName, accountName, token)
		return err
	default:
		return errors.New("unsupported credential manager")
	}
}

func (s *Store) deleteNative() error {
	switch s.goos {
	case "darwin":
		_, err := s.run("security", "delete-generic-password", "-s", serviceName, "-a", accountName)
		return err
	case "linux":
		_, err := s.run("secret-tool", "clear", "service", serviceName, "account", accountName)
		return err
	case "windows":
		script := "$v=[Windows.Security.Credentials.PasswordVault,Windows.Security.Credentials,ContentType=WindowsRuntime]::new();$o=$v.Retrieve($args[0],$args[1]);$v.Remove($o)"
		_, err := s.run("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script, serviceName, accountName)
		return err
	default:
		return errors.New("unsupported credential manager")
	}
}
