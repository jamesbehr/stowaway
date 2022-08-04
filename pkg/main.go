package pkg

import (
	"errors"
	"io/fs"
	"os"
	"strconv"

	"github.com/jamesbehr/stowaway/filesystem"
)

var (
	ErrPackageInstalled    = errors.New("pkg: package installed")
	ErrPackageNotInstalled = errors.New("pkg: package not installed")
)

type Package interface {
	Installed() (bool, error)
	Install() error
	Uninstall() error
}

type Loader struct {
	State, Source, Target filesystem.Path
}

func (l Loader) Load() (Package, error) {
	pkg := &localPackage{
		State:      l.State,
		Source:     l.Source,
		Target:     l.Target,
		SourceLink: l.State.Join("source"),
		TargetLink: l.State.Join("target"),
		Links:      l.State.Join("links"),
	}

	return pkg, nil
}

type localPackage struct {
	State, Source, Target, SourceLink, TargetLink, Links filesystem.Path
}

func shouldSymlink(mode fs.FileMode) bool {
	return mode.IsRegular() || mode == fs.ModeSymlink
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
