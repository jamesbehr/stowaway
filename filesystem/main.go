package filesystem

// Linker is an interface for creating symlinks inside of a filesystem. It
// implements a similar interface to FS in io/fs. The Linker has a root path
// that all other paths are relative to and implementors of this interface must
// not support absolute paths unless it is specifically indicated that they are
// allowed.
type Linker interface {
	// CreateLink creates a symlink to the path target at path linkName. It
	// will also create any missing directories if the underlying filesystem
	// supports it. The target may be an absolute path.
	CreateLink(target, linkName string) error

	// ReadLink fetches path that the link located at name points to.
	ReadLink(name string) (string, error)

	// RemoveLink removes the link at the specified path. Any parent
	// directories that are left empty must be removed by the implementation.
	RemoveLink(name string) error

	// Sub returns a Linker rooted at the subdirectory dir inside of the
	// current Linker's root. If dir is ".", the linker remains unchanged.
	Sub(dir string) (Linker, error)

	Root() string
}
