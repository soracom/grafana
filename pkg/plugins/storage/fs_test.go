package storage

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {
	const pluginID = "test-datasource"
	testDir := t.TempDir()
	store := FileSystem(&fakeLogger{}, testDir)

	// Package the existing signed plugin fixture with a normal release archive prefix.
	source := "../manager/testdata/valid-v2-signature/plugin"
	archivePath := filepath.Join(t.TempDir(), "plugin.zip")
	file, err := os.Create(archivePath)
	require.NoError(t, err)
	writer := zip.NewWriter(file)
	for _, name := range []string{"MANIFEST.txt", "plugin.json"} {
		data, err := os.ReadFile(filepath.Join(source, name))
		require.NoError(t, err)
		member, err := writer.Create("release/" + name)
		require.NoError(t, err)
		_, err = member.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())

	archive, err := store.Add(context.Background(), pluginID, zipFile(t, archivePath))
	require.NoError(t, err)
	require.Equal(t, pluginID, archive.ID)
	require.Equal(t, "1.0.0", archive.Version)
	require.Equal(t, filepath.Join(testDir, pluginID), archive.Path)
	for _, name := range []string{"MANIFEST.txt", "plugin.json"} {
		want, err := os.ReadFile(filepath.Join(source, name))
		require.NoError(t, err)
		got, err := os.ReadFile(filepath.Join(archive.Path, name))
		require.NoError(t, err)
		require.Equal(t, want, got)
	}

	t.Run("a rejected symlink archive leaves the installed copy intact", func(t *testing.T) {
		_, err := store.Add(context.Background(), pluginID, zipFile(t, "testdata/plugin-with-symlinks.zip"))
		require.ErrorContains(t, err, "symbolic link")
		files, err := os.ReadDir(archive.Path)
		require.NoError(t, err)
		require.Len(t, files, 2)
		want, err := os.ReadFile(filepath.Join(source, "plugin.json"))
		require.NoError(t, err)
		got, err := os.ReadFile(filepath.Join(archive.Path, "plugin.json"))
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("an invalid plugin ID leaves the plugins directory intact", func(t *testing.T) {
		_, err := store.Add(context.Background(), ".", zipFile(t, archivePath))
		require.ErrorContains(t, err, "invalid plugin directory name")
		require.FileExists(t, filepath.Join(archive.Path, "plugin.json"))
	})

	t.Run("missing metadata removes the partial installation", func(t *testing.T) {
		const id = "missing-metadata"
		_, err := store.Add(context.Background(), id, zipFile(t, "testdata/grafana-simple-json-datasource-ec18fa4da8096a952608a7e4c7782b4260b41bcf.zip"))
		require.ErrorContains(t, err, "failed to convert to plugin DTO")
		require.NoDirExists(t, filepath.Join(testDir, id))
	})

	t.Run("a failed copy removes the partial installation", func(t *testing.T) {
		broken := zipFile(t, archivePath)
		broken.File[1].CRC32 ^= 1
		_, err := store.Add(context.Background(), "failed-copy", broken)
		require.ErrorIs(t, err, zip.ErrChecksum)
		require.NoDirExists(t, filepath.Join(testDir, "failed-copy"))
	})
}

func TestRemove(t *testing.T) {
	pluginDir := t.TempDir()
	pluginJSON := filepath.Join(pluginDir, "plugin.json")
	//nolint:gosec
	_, err := os.Create(pluginJSON)
	require.NoError(t, err)

	pluginID := "test-datasource"
	i := &FS{
		pluginsDir: filepath.Dir(pluginDir),
		store: map[string]string{
			pluginID: pluginDir,
		},
		log: &fakeLogger{},
	}

	err = i.Remove(context.Background(), pluginID)
	require.NoError(t, err)

	_, err = os.Stat(pluginDir)
	require.True(t, os.IsNotExist(err))

	t.Run("Uninstall will search in nested dir folder for plugin.json", func(t *testing.T) {
		pluginDistDir := filepath.Join(t.TempDir(), "dist")
		err = os.Mkdir(pluginDistDir, os.ModePerm)
		require.NoError(t, err)
		pluginJSON = filepath.Join(pluginDistDir, "plugin.json")
		//nolint:gosec
		_, err = os.Create(pluginJSON)
		require.NoError(t, err)

		pluginDir = filepath.Dir(pluginDistDir)

		i = &FS{
			pluginsDir: filepath.Dir(pluginDir),
			store: map[string]string{
				pluginID: pluginDir,
			},
			log: &fakeLogger{},
		}

		err = i.Remove(context.Background(), pluginID)
		require.NoError(t, err)

		_, err = os.Stat(pluginDir)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("Uninstall will not delete folder if cannot recognize plugin structure", func(t *testing.T) {
		pluginDir = t.TempDir()
		i = &FS{
			pluginsDir: filepath.Dir(pluginDir),
			store: map[string]string{
				pluginID: pluginDir,
			},
			log: &fakeLogger{},
		}

		err = i.Remove(context.Background(), pluginID)
		require.EqualError(t, err, "cannot recognize as plugin folder")

		_, err = os.Stat(pluginDir)
		require.False(t, os.IsNotExist(err))
	})

	t.Run("Uninstall will not delete folder if plugin's directory is not a subdirectory of specified plugins directory", func(t *testing.T) {
		pluginDir = t.TempDir()
		i = &FS{
			pluginsDir: "/some/other/path",
			store: map[string]string{
				pluginID: pluginDir,
			},
			log: &fakeLogger{},
		}

		err = i.Remove(context.Background(), pluginID)
		require.EqualError(t, err, "cannot uninstall a plugin outside of the plugins directory")

		_, err = os.Stat(pluginDir)
		require.False(t, os.IsNotExist(err))
	})
}

func TestExtractFiles(t *testing.T) {
	pluginsDir := setupFakePluginsDir(t)

	i := &FS{log: &fakeLogger{}, pluginsDir: pluginsDir}

	t.Run("Should preserve file permissions for plugin backend binaries for linux and darwin", func(t *testing.T) {
		skipWindows(t)

		pluginID := "grafana-simple-json-datasource"
		path, err := i.extractFiles(context.Background(), zipFile(t, "testdata/grafana-simple-json-datasource-ec18fa4da8096a952608a7e4c7782b4260b41bcf.zip"), pluginID)
		require.Equal(t, filepath.Join(pluginsDir, pluginID), path)
		require.NoError(t, err)

		// File in zip has permissions 755
		fileInfo, err := os.Stat(filepath.Join(pluginsDir, "grafana-simple-json-datasource", "simple-plugin_darwin_amd64"))
		require.NoError(t, err)
		require.Equal(t, "-rwxr-xr-x", fileInfo.Mode().String())

		// File in zip has permission 755
		fileInfo, err = os.Stat(pluginsDir + "/grafana-simple-json-datasource/simple-plugin_linux_amd64")
		require.NoError(t, err)
		require.Equal(t, "-rwxr-xr-x", fileInfo.Mode().String())

		// File in zip has permission 644
		fileInfo, err = os.Stat(pluginsDir + "/grafana-simple-json-datasource/simple-plugin_windows_amd64.exe")
		require.NoError(t, err)
		require.Equal(t, "-rw-r--r--", fileInfo.Mode().String())

		// File in zip has permission 755
		fileInfo, err = os.Stat(pluginsDir + "/grafana-simple-json-datasource/non-plugin-binary")
		require.NoError(t, err)
		require.Equal(t, "-rwxr-xr-x", fileInfo.Mode().String())
	})

	for _, pluginID := range []string{"plugin-with-symlink", "plugin-with-symlink-dir", "plugin-with-absolute-symlink", "plugin-with-absolute-symlink-dir"} {
		t.Run(pluginID, func(t *testing.T) {
			path, err := i.extractFiles(context.Background(), zipFile(t, "testdata/"+pluginID+".zip"), pluginID)
			require.Empty(t, path)
			require.ErrorContains(t, err, "symbolic link")
			require.NoDirExists(t, filepath.Join(pluginsDir, pluginID))
		})
	}

	for _, pluginID := range []string{"plugin-with-parent-member", "plugin-with-absolute-member"} {
		t.Run(pluginID, func(t *testing.T) {
			path, err := i.extractFiles(context.Background(), zipFile(t, "testdata/"+pluginID+".zip"), pluginID)
			require.Empty(t, path)
			require.ErrorContains(t, err, "invalid plugin archive member")
			require.NoDirExists(t, filepath.Join(pluginsDir, pluginID))
		})
	}
}

func zipFile(t *testing.T, zipPath string) *zip.ReadCloser {
	rc, err := zip.OpenReader(zipPath)
	require.NoError(t, err)

	return rc
}

func TestArchiveDestination(t *testing.T) {
	installDir := filepath.Join("plugins", "test-plugin")
	got, err := archiveDestination(installDir, "release-123/partials/config.html")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(installDir, "partials", "config.html"), got)

	_, err = archiveDestination(installDir, "README.md")
	require.Error(t, err)
}

func TestWindowsPluginPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Win32 path normalization")
	}
	for _, name := range []string{"release/asset. ", "release/NUL.txt", "CONOUT$"} {
		require.False(t, validPluginPath(name))
	}
}

func setupFakePluginsDir(t *testing.T) string {
	dir := "testdata/fake-plugins-dir"
	err := os.RemoveAll(dir)
	require.NoError(t, err)

	err = os.MkdirAll(dir, 0750)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = os.RemoveAll(dir)
		require.NoError(t, err)
	})

	dir, err = filepath.Abs(dir)
	require.NoError(t, err)

	return dir
}

func skipWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}
}

type fakeLogger struct{}

func (f *fakeLogger) Successf(_ string, _ ...interface{}) {}
func (f *fakeLogger) Failuref(_ string, _ ...interface{}) {}
func (f *fakeLogger) Info(_ ...interface{})               {}
func (f *fakeLogger) Infof(_ string, _ ...interface{})    {}
func (f *fakeLogger) Debug(_ ...interface{})              {}
func (f *fakeLogger) Debugf(_ string, _ ...interface{})   {}
func (f *fakeLogger) Warn(_ ...interface{})               {}
func (f *fakeLogger) Warnf(_ string, _ ...interface{})    {}
func (f *fakeLogger) Error(_ ...interface{})              {}
func (f *fakeLogger) Errorf(_ string, _ ...interface{})   {}
