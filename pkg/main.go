package pkg

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strconv"

	"github.com/jamesbehr/stowaway/filesystem"
	"github.com/pelletier/go-toml/v2"
)

var (
	ErrPackageInstalled    = errors.New("pkg: package installed")
	ErrPackageNotInstalled = errors.New("pkg: package not installed")
)

type Package interface {
	Installed() (bool, error)
	Install() error
	Uninstall() error
	RunHookIfExists(name string) error
	Name() string
}

type Manifest struct {
	Name   string `toml:"name,omitempty"`
	Source string `toml:"source,omitempty"`
	Hooks  string `toml:"hooks,omitempty"`
}

type Loader struct {
	State, Source, Target filesystem.Path
}

func (l Loader) DefaultManifest() Manifest {
	return Manifest{
		Name:   l.Source.Basename(),
		Source: "src",
		Hooks:  "hooks",
	}
}

func (l Loader) Load() (Package, error) {
	pkg := &localPackage{
		State:       l.State,
		Source:      l.Source,
		PackageRoot: l.Source,
		Target:      l.Target,
		SourceLink:  l.State.Join("source"),
		TargetLink:  l.State.Join("target"),
		Links:       l.State.Join("links"),
	}

	manifest := pkg.Source.Join("stowaway.toml")
	exists, err := manifest.Exists()
	if err != nil {
		return nil, err
	}

	if exists {
		f, err := manifest.Open()
		if err != nil {
			return nil, err
		}

		defer f.Close()

		m := l.DefaultManifest()

		err = toml.NewDecoder(f).Decode(&m)
		if err != nil {
			return nil, err
		}

		pkg.Manifest = &m
		pkg.Source = pkg.Source.Join(m.Source)
	}

	return pkg, nil
}

type localPackage struct {
	// State is the path where all the package state will be stored
	State filesystem.Path

	// Source is the location of all the files in the package that will get
	// symlinks
	Source filesystem.Path

	// Package root is the directory containing every package file. If the
	// package has no manifest (i.e. it is a simple package), then this will be
	// the same as Source.
	PackageRoot filesystem.Path

	// Target is the the location of the directory that all symlinks will be
	// created relative to
	Target filesystem.Path

	// SourceLink is the path of the symlink in State that points to Source
	SourceLink filesystem.Path

	// TargetLink is the path of the symlink in State that points to Target
	TargetLink filesystem.Path

	// Links is the path in State that contains a number of symlinks. Each
	// symlink in this directory points to another symlink that was created in
	// the target directory.
	Links filesystem.Path

	// Manifiest is the parsed manifest for this package. If it is nil, then
	// the package had no manifiest and is thus a simple package. Simple
	// packages have no hooks and every file inside the package root will get a
	// symlink that points to it created.
	Manifest *Manifest
}

func shouldSymlink(mode fs.FileMode) bool {
	return mode.IsRegular() || mode == fs.ModeSymlink
}

func (pkg localPackage) Name() string {
	if pkg.Manifest == nil {
		return pkg.Source.Basename()
	}

	return pkg.Manifest.Name
}

func (pkg localPackage) RunHookIfExists(name string) error {
	// Simple packages cannot have hooks
	if pkg.Manifest == nil {
		return nil
	}

	executable := pkg.PackageRoot.Join(pkg.Manifest.Hooks, name)
	exists, err := executable.Exists()
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	cmd := exec.Command(executable.String(), pkg.State.String())

	cmd.Env = []string{
		fmt.Sprintf("STOWAWAY_SOURCE=%s", pkg.Source.String()),
		fmt.Sprintf("STOWAWAY_TARGET=%s", pkg.Target.String()),
		fmt.Sprintf("STOWAWAY_PACKAGE_ROOT=%s", pkg.PackageRoot.String()),
	}
	return cmd.Run()
}

func (pkg localPackage) Installed() (bool, error) {
	exists, err := pkg.State.Exists()
	if err != nil {
		return false, err
	}

	return exists, err
}

func (pkg localPackage) Install() error {
	exists, err := pkg.State.Exists()
	if err != nil {
		return err
	}

	if exists {
		return ErrPackageInstalled
	}

	if err := pkg.Links.MkdirAll(0700); err != nil {
		return err
	}

	if err := pkg.SourceLink.Symlink(pkg.Source); err != nil {
		return err
	}

	if err := pkg.TargetLink.Symlink(pkg.Target); err != nil {
		return err
	}

	linkCount := 0
	return pkg.SourceLink.Walk(func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the current directory
		if path == "." {
			return nil
		}

		if !shouldSymlink(info.Mode()) {
			return nil
		}

		target := pkg.TargetLink.Join(path)
		link := pkg.Links.Join(strconv.Itoa(linkCount))
		linkCount++

		// Every symlink that is created in the target directory gets an entry
		// in the links directory. The entry is itself a symlink pointing to
		// the target link, which allows Stowaway to keep track of all the
		// symlinks it has created.
		if err := link.Symlink(target); err != nil {
			return err
		}

		if err := target.Parent().MkdirAll(0755); err != nil {
			return err
		}

		source := pkg.SourceLink.Join(path)
		if err := target.Symlink(source); err != nil {
			return err
		}

		return nil
	})
}

func (pkg localPackage) Uninstall() error {
	exists, err := pkg.State.Exists()
	if err != nil {
		return err
	}

	if !exists {
		return ErrPackageNotInstalled
	}

	err = pkg.Links.Walk(func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == "." {
			return nil
		}

		link := pkg.Links.Join(path)
		target, err := link.Readlink()
		if err != nil {
			return err
		}

		if err := target.Remove(); err != nil {
			return err
		}

		if err := link.Remove(); err != nil {
			return err
		}

		// Remove empty parent directories
		for _, parent := range target.Parents() {
			empty, err := parent.Empty()
			if err != nil {
				break
			}

			if empty {
				if err := parent.Remove(); err != nil {
					return err
				}
			}

		}

		return nil
	})

	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	return pkg.State.RemoveAll()
}

type StowOptions struct {
	Delete bool
}

const (
	HookBeforeUninstallAll = "before_uninstall_all"
	HookAfterUninstallAll  = "after_uninstall_all"
	HookBeforeUninstall    = "before_uninstall"
	HookAfterUninstall     = "after_uninstall"
	HookBeforeInstall      = "before_install"
	HookAfterInstall       = "after_install"
	HookBeforeInstallAll   = "before_install_all"
	HookAfterInstallAll    = "after_install_all"
)

func Stow(options StowOptions, pkgs ...Package) error {
	for _, pkg := range pkgs {
		hook := HookBeforeInstallAll
		if options.Delete {
			hook = HookBeforeUninstallAll
		}

		if err := pkg.RunHookIfExists(hook); err != nil {
			return err
		}
	}

	for _, pkg := range pkgs {
		installed, err := pkg.Installed()
		if err != nil {
			return err
		}

		if installed {
			if err := pkg.RunHookIfExists(HookBeforeUninstall); err != nil {
				return err
			}

			if err := pkg.Uninstall(); err != nil {
				return err
			}

			if err := pkg.RunHookIfExists(HookAfterUninstall); err != nil {
				return err
			}
		}

		if !options.Delete {
			if err := pkg.RunHookIfExists(HookBeforeInstall); err != nil {
				return err
			}

			if err := pkg.Install(); err != nil {
				return err
			}

			if err := pkg.RunHookIfExists(HookAfterInstall); err != nil {
				return err
			}
		}
	}

	for _, pkg := range pkgs {
		hook := HookAfterInstallAll
		if options.Delete {
			hook = HookAfterUninstallAll
		}

		if err := pkg.RunHookIfExists(hook); err != nil {
			return err
		}
	}

	return nil
}
