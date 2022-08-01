package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesbehr/stowaway/filesystem"
	"github.com/stretchr/testify/require"
)

func tmpDir(t *testing.T, testName string, paths []string) filesystem.Path {
	pattern := "stowaway_" + testName
	dir, err := os.MkdirTemp(os.TempDir(), pattern)
	if err != nil {
		t.Fatalf("TempDir %s: %s", pattern, err)
	}

	for _, name := range paths {
		path := filepath.Join(dir, name)

		if strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(path, 0755); err != nil {
				os.RemoveAll(dir)
				t.Fatalf("MkdirAll %s: %s", dir, err)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				os.RemoveAll(dir)
				t.Fatalf("MkdirAll %s: %s", dir, err)
			}

			if err := os.WriteFile(path, []byte{}, 0755); err != nil {
				os.RemoveAll(dir)
				t.Fatalf("WriteFile %s: %s", path, err)
			}
		}
	}

	return filesystem.MakePath(dir)
}

func assertLink(t *testing.T, path string, target string) {
	link, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("Readlink %s: %s", path, err)
	}

	require.Equal(t, target, link)
}

type Links map[string]string

func assertLinks(t *testing.T, root filesystem.Path, links Links) {
	for source, target := range links {
		if !filepath.IsAbs(source) {
			source = root.Join(source).String()
		}

		if !filepath.IsAbs(target) {
			target = root.Join(target).String()
		}

		assertLink(t, source, target)
	}
}

func assertMissing(t *testing.T, root filesystem.Path, files []string) {
	for _, file := range files {
		if !filepath.IsAbs(file) {
			file = root.Join(file).String()
		}

		_, err := os.Lstat(file)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			t.Fatalf("Stat %s: %s", file, err)
		}

		t.Fatalf("file %s exists", file)
	}
}

func createLinks(t *testing.T, root filesystem.Path, links Links) {
	for source, target := range links {
		if !filepath.IsAbs(source) {
			source = root.Join(source).String()
		}

		if !filepath.IsAbs(target) {
			target = root.Join(target).String()
		}

		if err := os.MkdirAll(filepath.Dir(source), 0755); err != nil {
			t.Fatalf("MkdirAll %s: %s", source, err)
		}

		if err := os.Symlink(target, source); err != nil {
			t.Fatalf("Symlink %s -> %s: %s", source, target, err)
		}
	}
}

func TestInstall(t *testing.T) {
	t.Run("simple install", func(t *testing.T) {
		tmp := tmpDir(t, "install", []string{
			"bash/.bashrc",
			"bash/.bin/test",
			"home/user/",
		})

		defer tmp.RemoveAll()

		p := Installer{
			State:  tmp.Join("data"),
			Target: tmp.Join("home/user"),
			Source: tmp.Join("bash"),
		}

		err := p.Install()
		require.NoError(t, err)

		assertLinks(t, tmp, Links{
			"data/source":         "bash",
			"data/target":         "home/user",
			"data/links/0":        "data/target/.bashrc",
			"home/user/.bashrc":   "data/source/.bashrc",
			"home/user/.bin/test": "data/source/.bin/test",
		})
	})
}

func TestUninstall(t *testing.T) {
	t.Run("simple uninstall", func(t *testing.T) {
		// Note that the package has been modified (it is now empty), the
		// installation state comes entirely from the symlinks in the data
		// directory
		tmp := tmpDir(t, "uninstall", []string{
			"bash/",
			"home/user/",
			"data/",
		})

		defer tmp.RemoveAll()

		createLinks(t, tmp, Links{
			"data/source":         "bash",
			"data/target":         "home/user",
			"data/links/0":        "data/target/.bashrc",
			"data/links/1":        "data/target/.bin/test",
			"data/links/2":        "data/target/.bin/another", // broken link
			"home/user/.bashrc":   "data/source/.bashrc",
			"home/user/.bin/test": "data/source/.bin/test",
		})

		p := Installer{
			State:  tmp.Join("data"),
			Target: tmp.Join("home/user"),
			Source: tmp.Join("bash"),
		}

		err := p.Uninstall()
		require.NoError(t, err)

		assertMissing(t, tmp, []string{
			"home/user/.bashrc",
			"home/user/.bin/test",
			"data/",
			"home/user/.bin",
		})
	})

	t.Run("missing links", func(t *testing.T) {
		tmp := tmpDir(t, "uninstall", []string{
			"bash/",
			"home/user/",
			"data/",
		})

		defer tmp.RemoveAll()

		createLinks(t, tmp, Links{
			"data/source": "bash",
			"data/target": "home/user",
		})

		p := Installer{
			State:  tmp.Join("data"),
			Target: tmp.Join("home/user"),
			Source: tmp.Join("bash"),
		}

		err := p.Uninstall()
		require.NoError(t, err)

		assertMissing(t, tmp, []string{
			"data/",
		})
	})
}
