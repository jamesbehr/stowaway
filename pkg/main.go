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

type Installer struct {
	State, Source, Target filesystem.Path
}

func (i Installer) SourceLink() filesystem.Path {
	return i.State.Join("source")
}

func (i Installer) TargetLink() filesystem.Path {
	return i.State.Join("target")
}

func (i Installer) Links() filesystem.Path {
	return i.State.Join("links")
}

func shouldSymlink(mode fs.FileMode) bool {
	return mode.IsRegular() || mode == fs.ModeSymlink
}

func (i Installer) Installed() (bool, error) {
	exists, err := i.State.Exists()
	if err != nil {
		return false, err
	}

	return exists, err
}

func (i Installer) Install() error {
	exists, err := i.State.Exists()
	if err != nil {
		return err
	}

	if exists {
		return ErrPackageInstalled
	}

	if err := i.Links().MkdirAll(0700); err != nil {
		return err
	}

	if err := i.SourceLink().Symlink(i.Source); err != nil {
		return err
	}

	if err := i.TargetLink().Symlink(i.Target); err != nil {
		return err
	}

	linkCount := 0
	return i.SourceLink().Walk(func(path string, info os.FileInfo, err error) error {
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

		target := i.TargetLink().Join(path)
		link := i.Links().Join(strconv.Itoa(linkCount))
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

		source := i.SourceLink().Join(path)
		if err := target.Symlink(source); err != nil {
			return err
		}

		return nil
	})
}

func (i Installer) Uninstall() error {
	exists, err := i.State.Exists()
	if err != nil {
		return err
	}

	if !exists {
		return ErrPackageNotInstalled
	}

	err = i.Links().Walk(func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == "." {
			return nil
		}

		link := i.Links().Join(path)
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

	return i.State.RemoveAll()
}
