# Stowaway
A symlink farm manager similar to [GNU Stow]. It allows you to create symlinks
in a directory such as your home directory that all point to files under a
single directory, such as `~/dotfiles`. You can then check your `~/dotfiles`
into version control, without fear of polluting the repository with other files
from your home directory.

[GNU Stow]: https://www.gnu.org/software/stow/

## Differences to GNU Stow
- Zero dependencies, just download and run the binary
- Keeps track of which symlinks have been created
- Scriptable hooks to run before or after a package is installed

## Installation
You can install it with `go install`.

    go install github.com/jamesbehr/stowaway

If you just want to download the binary, you can also do that. This is useful
for bootstrapping scripts that will setup your dotfiles on a clean install. For
example, you can download the 64-bit x86 Linux binaries like this. See the
releases page for more information about which platforms are supported.

    curl -LO  https://github.com/jamesbehr/stowaway/releases/latest/download/stowaway-linux-amd64.tar.gz
    curl -LO  https://github.com/jamesbehr/stowaway/releases/latest/download/stowaway-linux-amd64.tar.gz.sha256sum
    sha256sum -c stowaway-linux-amd64.tar.gz.sha256sum
    tar -xzf stowaway-linux-amd64.tar.gz
    rm stowaway-linux-amd64.tar.gz*

## Usage
Like GNU Stow, Stowaway operates on *packages*. Packages are GNU Stow
compatible, in that they are just directories containing a number of files.
Each file inside the package will get a symlink in the target directory.

Assuming you've cloned the repository into a directory called `stowaway`,

```console
$ find stowaway/examples/bash
stowaway/examples/bash
stowaway/examples/bash/.bashrc
```

The package will be installed into a *target directory*. Stowaway will create a
symlink to each file in the package in the target directory and create any
missing directories along the way. The symlinks path in the target directory
corresponds to its path in the package directory.

You can install a package by running the `stow` command. You can specify
multiple packages paths to install into a target directory. If you do not
specify a target directory with the `--target` flag, then the current working
directory will be used as the target. If the package is already installed it
will be uninstalled before being reinstalled.

```console
$ pwd
/home/me
$ rm .bashrc
$ cp -avr stowaway/examples ~/dotfiles
$ stowaway stow ~/dotfiles/bash
```

After installing there will be a file called `/home/me/.bashrc` pointing to the
file in the package `~/dotfiles/bash/.bashrc`.

```console
$ ls -a .bash*
.bash_logout
.bashrc

$ readlink -f ~/.bashrc
/home/me/dotfiles/bash/.bashrc
```

If you want to uninstall a package you can provide the `--delete` flag. This
will clear up symlinks even when the original package has been modified.

```console
$ rm ~/dotfiles/bash/.bashrc
$ stowaway stow --delete ~/dotfiles/bash
$ ls -a .bash*
.bash_logout
```

### Interactive mode
You can also pass the `--interactive` flag to the `stow` command, which will
prompt the user to select which packages they want to install or uninstall from
the list of packages you provide as arguments. This allows you to do things
like passing in all the available packages as arguments and have the user
select which ones they want to install.

## Advanced features
Stowaway also supports some advanced features, such as installation hooks.
Hooks are scripts that run at certain points in the package's life cycle. For
example, you might have a package that has code written in a compiled language.
You could use a hook that runs after the package is installed to run `make` and
compile the package.

To use these advanced features, you'll need to use a different package
structure to the normal, GNU Stow-compatible, packages. An package that wants
to use these features might look like this:

```console
$ find stowaway/examples/bash-advanced
stowaway/examples/bash-advanced
stowaway/examples/bash-advanced/src
stowaway/examples/bash-advanced/src/.bashrc
stowaway/examples/bash-advanced/stowaway.toml
stowaway/examples/bash-advanced/hooks
stowaway/examples/bash-advanced/hooks/after_install
```

Notice the `stowaway.toml` in the root of the package. This is the package
manifest. The presence of the package manifest enables the advanced features.

All the files that will get symlinks created are now located under the `src`
directory in the package. You can change this package by setting the `source`
configuration option in the package manifest. All symlink names are derived
from the name of the file relative to this directory, that is to say the name
of the symlink pointing to `src/.bashrc` will be `$TARGET/.bashrc`, not
`$TARGET/src/.bashrc` (where `$TARGET` is the installation target directory).

The package can also specify hooks, which work similarly to Git hooks. A hook
is just a file with the executable flag set. This file will be executed at
certain points in the package life cycle. All hooks currently get the path to
the packages installation state directory passed as their only argument. See
the [section on package state](#package-state). The name of the hook specifies
the life cycle event that will cause it to run.

The following hoooks are currently available, in the order they are run:

- `before_uninstall_all`: Run for each selected package in a `stow --delete`
operation.
- `before_uninstall` : Run for a package right before it is uninstalled. Only run
if the package is installed.
- `after_uninstall`: Run after uninstalling the package.
- `after_uninstall_all` Like `before_uninstall_all`, but run after every package
was uninstalled.
- `before_install_all`: Run for each selected package in a `stow` operation.
- `before_install` : Run for a package right before it is installed.
- `after_install`: Run after installing the package.
- `after_install_all` Like `before_install_all`, but run after every package
was installed.

## Package State
Stowaway keeps track of each package installed in the `.stowaway` directory
inside the target directory. Inside this directory are a number of
subdirectories, each containing the state of an installed Stowaway package.

```console
$ stowaway stow stowaway/examples/bash
$ find /home/me/.stowaway
/home/me/.stowaway
/home/me/.stowaway/37bc12
/home/me/.stowaway/37bc12/links
/home/me/.stowaway/37bc12/links/0
/home/me/.stowaway/37bc12/source
/home/me/.stowaway/37bc12/target
```

In the example above, `/home/me/.stowaway/37bc12` is the package installation
state directory.

For each symlink that Stowaway creates, it creates another symlink pointing to
that symlink inside the `links` directory. This enables Stowaway to keep track
of which symlinks it has created, even when the contents of the package have
been modified.

The `target` and `source` directories are symlinks to the installation target
and package source directories respectively. For packages with a manifest, this
defaults to the `src` directory in the package root, and is the same as the
package root for packages without a manifest.

```console
$ readlink /home/me/.stowaway/37bc12/links/0
/home/me/.stowaway/37bc12/target/.bashrc

$ readlink /home/me/.stowaway/37bc12/target/.bashrc
/home/me/.stowaway/37bc12/source/.bashrc

$ readlink -f /home/me/.stowaway/37bc12/source/.bashrc
/home/me/stowaway/examples/bash/.bashrc
```

## Tests
You can run the unit tests by running `make test`.

You can also verify that the examples in the README are correct by `make
doctest`. This requires Docker to be installed. The doctests validate that
every code fence in this markdown document that has `console` selected as its
language actually outputs what is written down.
