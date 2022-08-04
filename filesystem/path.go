package filesystem

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type Path string

func MakePath(names ...string) Path {
	p := filepath.Join(names...)

	if !filepath.IsAbs(p) {
		panic("MakePath requires absolute path")
	}

	return Path(p)
}

func (p Path) Join(names ...string) Path {
	args := []string{string(p)}
	args = append(args, names...)
	return MakePath(args...)
}

func (p Path) Parent() Path {
	return Path(filepath.Dir(string(p)))
}

func (p Path) Parents() []Path {
	parents := []Path{}

	for {
		parent := p.Parent()
		if string(parent) == string(p) {
			break
		}

		parents = append(parents, parent)
		p = parent
	}

	return parents
}

func (p Path) MkdirAll(perm os.FileMode) error {
	return os.MkdirAll(string(p), perm)
}

func (p Path) RemoveAll() error {
	return os.RemoveAll(string(p))
}

func (p Path) Remove() error {
	return os.Remove(string(p))
}

func (p Path) WriteFile(data []byte, perm os.FileMode) error {
	return os.WriteFile(string(p), data, perm)
}

func (p Path) Open() (*os.File, error) {
	return os.Open(string(p))
}

func (p Path) Readlink() (Path, error) {
	target, err := os.Readlink(string(p))
	if err != nil {
		return Path(""), err
	}

	return Path(target), nil
}

func (p Path) Symlink(target Path) error {
	return os.Symlink(string(target), string(p))
}

func (p Path) Walk(f filepath.WalkFunc) error {
	fsys := os.DirFS(string(p))
	return fs.WalkDir(fsys, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return f(path, nil, err)
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		return f(path, info, err)
	})
}

func (p Path) Exists() (bool, error) {
	_, err := os.Lstat(string(p))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (p Path) Empty() (bool, error) {
	file, err := os.Open(string(p))
	if err != nil {
		return false, err
	}

	defer file.Close()

	_, err = file.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}

	return false, err
}

func (p Path) String() string {
	return string(p)
}
