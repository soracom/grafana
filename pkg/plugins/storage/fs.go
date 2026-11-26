package storage

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/grafana/grafana/pkg/plugins/logger"
)

var _ Manager = (*FS)(nil)

var windowsDeviceName = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|CONIN\$|CONOUT\$|(COM|LPT)[0-9¹²³])$`)

var (
	ErrUninstallOutsideOfPluginDir = errors.New("cannot uninstall a plugin outside of the plugins directory")
	ErrUninstallInvalidPluginDir   = errors.New("cannot recognize as plugin folder")
)

type FS struct {
	store      map[string]string
	mu         sync.Mutex
	pluginsDir string
	log        logger.Logger
}

func FileSystem(logger logger.Logger, pluginsDir string) *FS {
	return &FS{
		store:      make(map[string]string),
		pluginsDir: pluginsDir,
		log:        logger,
	}
}

func (fs *FS) Add(ctx context.Context, pluginID string, pluginArchive *zip.ReadCloser) (
	*ExtractedPluginArchive, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	pluginDir, err := fs.extractFiles(ctx, pluginArchive, pluginID)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", "failed to extract plugin archive", err)
	}

	res, err := toPluginDTO(pluginID, pluginDir)
	if err != nil {
		fs.removePartialPlugin(pluginDir)
		return nil, fmt.Errorf("%v: %w", "failed to convert to plugin DTO", err)
	}

	fs.log.Successf("Downloaded and extracted %s v%s zip successfully to %s", res.ID, res.Info.Version, pluginDir)

	var deps []*Dependency
	for _, plugin := range res.Dependencies.Plugins {
		deps = append(deps, &Dependency{
			ID:      plugin.ID,
			Version: plugin.Version,
		})
	}

	return &ExtractedPluginArchive{
		ID:           res.ID,
		Version:      res.Info.Version,
		Dependencies: deps,
		Path:         pluginDir,
	}, nil
}

func (fs *FS) Register(_ context.Context, pluginID, pluginDir string) error {
	fs.mu.Lock()
	fs.store[pluginID] = pluginDir
	fs.mu.Unlock()

	return nil
}

func (fs *FS) Remove(_ context.Context, pluginID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	pluginDir, exists := fs.store[pluginID]
	if !exists {
		return fmt.Errorf("%s does not exist", pluginID)
	}

	// extra security check to ensure we only remove plugins that are located in the configured plugins directory
	path, err := filepath.Rel(fs.pluginsDir, pluginDir)
	if err != nil || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return ErrUninstallOutsideOfPluginDir
	}

	if _, err = os.Stat(filepath.Join(pluginDir, "plugin.json")); os.IsNotExist(err) {
		if _, err = os.Stat(filepath.Join(pluginDir, "dist/plugin.json")); os.IsNotExist(err) {
			return ErrUninstallInvalidPluginDir
		}
	}

	fs.log.Infof("Uninstalling plugin %v", pluginDir)
	return os.RemoveAll(pluginDir)
}

func (fs *FS) extractFiles(_ context.Context, pluginArchive *zip.ReadCloser, pluginID string) (_ string, err error) {
	defer func() {
		if err := pluginArchive.Close(); err != nil {
			fs.log.Warn("failed to close zip file", "err", err)
		}
	}()

	if err := ValidateArchive(pluginID, &pluginArchive.Reader); err != nil {
		return "", err
	}
	installDir := filepath.Join(fs.pluginsDir, pluginID)

	// RemoveAll removes the directory entry itself, including a dangling symlink.
	if err := os.RemoveAll(installDir); err != nil {
		return "", err
	}
	defer func() {
		if err != nil {
			fs.removePartialPlugin(installDir)
		}
	}()

	for _, zf := range pluginArchive.File {
		dstPath := archiveDestination(installDir, zf.Name)
		if zf.FileInfo().IsDir() {
			// We can ignore gosec G304 here since it makes sense to give all users read access
			// nolint:gosec
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				if os.IsPermission(err) {
					return "", ErrPermissionDenied{Path: dstPath}
				}

				return "", err
			}
			continue
		}

		// Create needed directories to extract file
		// We can ignore gosec G304 here since it makes sense to give all users read access
		// nolint:gosec
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return "", fmt.Errorf("%v: %w", "failed to create directory to extract plugin files", err)
		}

		if err := extractFile(zf, dstPath); err != nil {
			return "", fmt.Errorf("%v: %w", "failed to extract file", err)
		}
	}

	return installDir, nil
}

// ValidateArchive checks the archive headers before an installation can be replaced.
func ValidateArchive(pluginID string, archive *zip.Reader) error {
	if strings.Contains(pluginID, "/") || !validPluginPath(pluginID) {
		return fmt.Errorf("invalid plugin directory name %q", pluginID)
	}
	for _, member := range archive.File {
		if member.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive member %q is a symbolic link", member.Name)
		}
		if !validPluginPath(strings.TrimSuffix(member.Name, "/")) {
			return fmt.Errorf("invalid plugin archive member %q", member.Name)
		}
		if !strings.Contains(member.Name, "/") {
			return fmt.Errorf("archive member %q is missing a plugin directory", member.Name)
		}
	}
	return nil
}

// Grafana release archives wrap plugin contents in a top-level directory, often a Git build name.
func archiveDestination(installDir, name string) string {
	_, relative, _ := strings.Cut(name, "/")
	return filepath.Join(installDir, filepath.FromSlash(relative))
}

func validPluginPath(name string) bool {
	if !fs.ValidPath(name) || name == "." || strings.ContainsAny(name, `\:`) {
		return false
	}
	if runtime.GOOS == "windows" {
		// Win32 normalizes trailing spaces and dots, and interprets reserved names as devices.
		for _, part := range strings.Split(name, "/") {
			stem, _, _ := strings.Cut(part, ".")
			if strings.TrimRight(part, " .") != part || windowsDeviceName.MatchString(strings.TrimRight(stem, " ")) {
				return false
			}
		}
	}
	return true
}

func (fs *FS) removePartialPlugin(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		fs.log.Warn("failed to remove partially extracted plugin", "path", dir, "err", err)
	}
}

func extractFile(file *zip.File, filePath string) (err error) {
	fileMode := file.Mode()
	// This is entry point for backend plugins so we want to make them executable
	if strings.HasSuffix(filePath, "_linux_amd64") || strings.HasSuffix(filePath, "_darwin_amd64") {
		fileMode = os.FileMode(0755)
	}

	// We can ignore the gosec G304 warning on this one, since the variable part of the file path stems
	// from command line flag "pluginsDir", and the only possible damage would be writing to the wrong directory.
	// If the user shouldn't be writing to this directory, they shouldn't have the permission in the file system.
	// nolint:gosec
	dst, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("could not create %q, permission denied, make sure you have write access to plugin dir", filePath)
		}

		unwrappedError := errors.Unwrap(err)
		if unwrappedError != nil && strings.EqualFold(unwrappedError.Error(), "text file busy") {
			return fmt.Errorf("file %q is in use - please stop Grafana, install the plugin and restart Grafana", filePath)
		}

		return fmt.Errorf("%v: %w", "failed to open file", err)
	}
	defer func() {
		if closeErr := dst.Close(); err == nil {
			err = closeErr
		}
	}()

	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("%v: %w", "failed to extract file", err)
	}
	defer func() {
		if closeErr := src.Close(); err == nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(dst, src)
	return err
}

func toPluginDTO(pluginID, pluginDir string) (InstalledPlugin, error) {
	distPluginDataPath := filepath.Join(pluginDir, "dist", "plugin.json")

	// It's safe to ignore gosec warning G304 since the file path suffix is hardcoded
	// nolint:gosec
	data, err := os.ReadFile(distPluginDataPath)
	if err != nil {
		pluginDataPath := filepath.Join(pluginDir, "plugin.json")
		// It's safe to ignore gosec warning G304 since the file path suffix is hardcoded
		// nolint:gosec
		data, err = os.ReadFile(pluginDataPath)
		if err != nil {
			return InstalledPlugin{}, fmt.Errorf("could not find dist/plugin.json or plugin.json for %s in %s", pluginID, pluginDir)
		}
	}

	res := InstalledPlugin{}
	if err = json.Unmarshal(data, &res); err != nil {
		return res, err
	}

	if res.ID == "" {
		return InstalledPlugin{}, fmt.Errorf("could not find valid plugin %s in %s", pluginID, pluginDir)
	}

	if res.Info.Version == "" {
		res.Info.Version = "0.0.0"
	}

	return res, nil
}
