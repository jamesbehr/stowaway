package pkg

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesbehr/stowaway/filesystem"
	"github.com/pelletier/go-toml/v2"
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

func writeManifest(t *testing.T, root filesystem.Path, name string, manifest *Manifest) {
	w := bytes.NewBuffer([]byte{})
	err := toml.NewEncoder(w).Encode(manifest)
	if err != nil {
		t.Fatalf("toml.Encode %s: %s", name, err)
	}

	err = root.Join(name).WriteFile(w.Bytes(), 0755)
	if err != nil {
		t.Fatalf("WriteFile %s: %s", name, err)
	}
}

func writeFile(t *testing.T, root filesystem.Path, name string, contents string, perm os.FileMode) {
	path := root.Join(name)

	if err := path.Parent().MkdirAll(0755); err != nil {
		t.Fatalf("MkdirAll %s: %s", name, err)
	}

	err := path.WriteFile([]byte(contents), perm)
	if err != nil {
		t.Fatalf("WriteFile %s: %s", name, err)
	}
}

func TestRunHook(t *testing.T) {
	tmp := tmpDir(t, "hooks", []string{"bash/", "data/"})
	defer tmp.RemoveAll()

	loader := Loader{
		State:  tmp.Join("data"),
		Target: tmp.Join("home/user"),
		Source: tmp.Join("bash"),
	}

	writeManifest(t, tmp, "bash/stowaway.toml", &Manifest{})

	p, err := loader.Load()
	require.NoError(t, err)

	script := `#!/bin/sh
touch "$1/hookran"`

	writeFile(t, tmp, "bash/hooks/broken", "not executable", 0655)
	writeFile(t, tmp, "bash/hooks/working", script, 0755)

	require.NoError(t, p.RunHookIfExists("missing"))
	require.NoError(t, p.RunHookIfExists("working"))
	require.Error(t, p.RunHookIfExists("broken"))

	require.FileExists(t, tmp.Join("data/hookran").String())
}

type InstallTestCase struct {
	Name          string
	Filesystem    []string
	Manifest      *Manifest
	ExpectedLinks Links
}

func TestInstall(t *testing.T) {
	testCases := []InstallTestCase{
		{
			Name: "simple package",
			Filesystem: []string{
				"bash/.bashrc",
				"bash/.bin/test",
				"home/user/",
			},
			ExpectedLinks: Links{
				"data/source":         "bash",
				"data/target":         "home/user",
				"data/links/0":        "data/target/.bashrc",
				"home/user/.bashrc":   "data/source/.bashrc",
				"home/user/.bin/test": "data/source/.bin/test",
			},
		},
		{
			Name: "package with default manifest",
			Filesystem: []string{
				"bash/src/.bashrc",
				"bash/src/.bin/test",
				"home/user/",
			},
			Manifest: &Manifest{},
			ExpectedLinks: Links{
				"data/source":         "bash/src",
				"data/target":         "home/user",
				"data/links/0":        "data/target/.bashrc",
				"home/user/.bashrc":   "data/source/.bashrc",
				"home/user/.bin/test": "data/source/.bin/test",
			},
		},
		{
			Name: "package with manifest with source",
			Filesystem: []string{
				"bash/files/.bashrc",
				"bash/files/.bin/test",
				"home/user/",
			},
			Manifest: &Manifest{
				Source: "files",
			},
			ExpectedLinks: Links{
				"data/source":         "bash/files",
				"data/target":         "home/user",
				"data/links/0":        "data/target/.bashrc",
				"home/user/.bashrc":   "data/source/.bashrc",
				"home/user/.bin/test": "data/source/.bin/test",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			tmp := tmpDir(t, "install", testCase.Filesystem)
			defer tmp.RemoveAll()

			if testCase.Manifest != nil {
				writeManifest(t, tmp, "bash/stowaway.toml", testCase.Manifest)
			}

			loader := Loader{
				State:  tmp.Join("data"),
				Target: tmp.Join("home/user"),
				Source: tmp.Join("bash"),
			}

			p, err := loader.Load()
			require.NoError(t, err)

			err = p.Install()
			require.NoError(t, err)

			assertLinks(t, tmp, testCase.ExpectedLinks)
		})
	}
}

type UninstallTestCase struct {
	Name            string
	Filesystem      []string
	Manifest        *Manifest
	Links           Links
	ExpectedMissing []string
}

func TestUninstall(t *testing.T) {
	testCases := []UninstallTestCase{
		{
			Name: "simple package",

			// Note that the package has been modified (it is now empty), the
			// installation state comes entirely from the symlinks in the data
			// directory
			Filesystem: []string{
				"bash/",
				"home/user/",
				"data/",
			},
			Links: Links{
				"data/source":         "bash",
				"data/target":         "home/user",
				"data/links/0":        "data/target/.bashrc",
				"data/links/1":        "data/target/.bin/test",
				"data/links/2":        "data/target/.bin/another", // broken link
				"home/user/.bashrc":   "data/source/.bashrc",
				"home/user/.bin/test": "data/source/.bin/test",
			},
			ExpectedMissing: []string{
				"home/user/.bashrc",
				"home/user/.bin/test",
				"data/",
				"home/user/.bin",
			},
		},
		{
			Name: "package with missing links",
			Filesystem: []string{
				"bash/",
				"home/user/",
				"data/",
			},
			Links: Links{
				"data/source": "bash",
				"data/target": "home/user",
			},
			ExpectedMissing: []string{
				"data/",
			},
		},
		{
			Name: "package with default manifest",
			Filesystem: []string{
				"bash/",
				"home/user/",
				"data/",
			},
			Manifest: &Manifest{},
			Links: Links{
				"data/source":         "bash/src",
				"data/target":         "home/user",
				"data/links/0":        "data/target/.bashrc",
				"data/links/1":        "data/target/.bin/test",
				"data/links/2":        "data/target/.bin/another", // broken link
				"home/user/.bashrc":   "data/source/.bashrc",
				"home/user/.bin/test": "data/source/.bin/test",
			},
			ExpectedMissing: []string{
				"home/user/.bashrc",
				"home/user/.bin/test",
				"data/",
				"home/user/.bin",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			tmp := tmpDir(t, "uninstall", testCase.Filesystem)
			defer tmp.RemoveAll()

			if testCase.Manifest != nil {
				writeManifest(t, tmp, "bash/stowaway.toml", testCase.Manifest)
			}

			createLinks(t, tmp, testCase.Links)

			loader := Loader{
				State:  tmp.Join("data"),
				Target: tmp.Join("home/user"),
				Source: tmp.Join("bash"),
			}

			p, err := loader.Load()
			require.NoError(t, err)

			err = p.Uninstall()
			require.NoError(t, err)

			assertMissing(t, tmp, testCase.ExpectedMissing)
		})
	}
}

type MockPackage struct {
	IsInstalled   bool
	InstallCalled func(string, bool)
	HookCalled    func(string, string)
	Name          string
}

func (m *MockPackage) Install() error {
	if m.InstallCalled != nil {
		m.InstallCalled(m.Name, false)
	}

	return nil
}

func (m *MockPackage) Uninstall() error {
	if m.InstallCalled != nil {
		m.InstallCalled(m.Name, true)
	}

	return nil
}

func (m *MockPackage) Installed() (bool, error) {
	return m.IsInstalled, nil
}

func (m *MockPackage) RunHookIfExists(name string) error {
	if m.HookCalled != nil {
		m.HookCalled(m.Name, name)
	}

	return nil
}

func TestStow(t *testing.T) {

	testCases := []struct {
		IsInstalled                    bool
		Delete                         bool
		ExpectInstall, ExpectUninstall bool
	}{
		{IsInstalled: true, Delete: true, ExpectInstall: false, ExpectUninstall: true},
		{IsInstalled: false, Delete: true, ExpectInstall: false, ExpectUninstall: false},
		{IsInstalled: true, Delete: false, ExpectInstall: true, ExpectUninstall: true},
		{IsInstalled: false, Delete: false, ExpectInstall: true, ExpectUninstall: false},
	}

	for _, testCase := range testCases {
		name := fmt.Sprintf("installed=%t delete=%t", testCase.IsInstalled, testCase.Delete)

		t.Run(name, func(t *testing.T) {
			installCalled := false
			uninstallCalled := false

			mock := MockPackage{
				IsInstalled: testCase.IsInstalled,
				InstallCalled: func(name string, uninstall bool) {
					if uninstall {
						uninstallCalled = true
					} else {
						installCalled = true
					}
				},
			}

			options := StowOptions{
				Delete: testCase.Delete,
			}

			err := Stow(options, &mock)
			require.NoError(t, err)

			require.Equal(t, testCase.ExpectUninstall, uninstallCalled)
			require.Equal(t, testCase.ExpectInstall, installCalled)
		})
	}

	t.Run("uninstall hooks", func(t *testing.T) {
		actions := []string{}
		ins := func(pkgName string, uninstall bool) {
			action := "install"
			if uninstall {
				action = "uninstall"
			}

			actions = append(actions, fmt.Sprintf("%s:%s", pkgName, action))
		}

		hook := func(pkgName, name string) {
			actions = append(actions, fmt.Sprintf("%s:%s", pkgName, name))
		}

		pkgs := []Package{
			&MockPackage{Name: "a", IsInstalled: true, HookCalled: hook, InstallCalled: ins},
			&MockPackage{Name: "b", IsInstalled: false, HookCalled: hook, InstallCalled: ins},
			&MockPackage{Name: "c", IsInstalled: true, HookCalled: hook, InstallCalled: ins},
		}

		options := StowOptions{
			Delete: true,
		}

		err := Stow(options, pkgs...)
		require.NoError(t, err)

		require.Equal(t, []string{
			"a:before_uninstall_all",
			"b:before_uninstall_all",
			"c:before_uninstall_all",
			"a:before_uninstall",
			"a:uninstall",
			"a:after_uninstall",
			"c:before_uninstall",
			"c:uninstall",
			"c:after_uninstall",
			"a:after_uninstall_all",
			"b:after_uninstall_all",
			"c:after_uninstall_all",
		}, actions)
	})

	t.Run("install hooks", func(t *testing.T) {
		actions := []string{}
		ins := func(pkgName string, uninstall bool) {
			action := "install"
			if uninstall {
				action = "uninstall"
			}

			actions = append(actions, fmt.Sprintf("%s:%s", pkgName, action))
		}

		hook := func(pkgName, name string) {
			actions = append(actions, fmt.Sprintf("%s:%s", pkgName, name))
		}

		pkgs := []Package{
			&MockPackage{Name: "a", IsInstalled: false, HookCalled: hook, InstallCalled: ins},
			&MockPackage{Name: "b", IsInstalled: true, HookCalled: hook, InstallCalled: ins},
			&MockPackage{Name: "c", IsInstalled: false, HookCalled: hook, InstallCalled: ins},
		}

		options := StowOptions{
			Delete: false,
		}

		err := Stow(options, pkgs...)
		require.NoError(t, err)

		require.Equal(t, []string{
			"a:before_install_all",
			"b:before_install_all",
			"c:before_install_all",

			"a:before_install",
			"a:install",
			"a:after_install",

			"b:before_uninstall",
			"b:uninstall",
			"b:after_uninstall",
			"b:before_install",
			"b:install",
			"b:after_install",

			"c:before_install",
			"c:install",
			"c:after_install",

			"a:after_install_all",
			"b:after_install_all",
			"c:after_install_all",
		}, actions)
	})
}
