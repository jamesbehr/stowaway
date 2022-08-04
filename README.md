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

    $ find examples/bash
    examples/bash
    examples/bash/.bashrc

The package will be installed into a *target directory*. Stowaway will create a
symlink to each file in the package in the target directory and create any
missing directories along the way. The symlinks path in the target directory
corresponds to its path in the package directory.

You can install a package by running the `stow` command. You can specify
multiple packages paths to install into a target directory. If you do not
specify a target directory with the `--target` flag, then the current working
directory will be used as the target. If the package is already installed it
will be uninstalled before being reinstalled.

    $ cp -ar examples ~/dotfiles
    $ stowaway stow --target /home/me ~/dotfiles/bash

After installing there will be a file called `/home/me/.bashrc` pointing to the
file in the package `~/dotfiles/bash/.bashrc`.

    $ readlink -f ~/.bashrc
    /home/me/dotfiles/bash/.bashrc

If you want to uninstall a package you can provide the `--delete` flag.

    $ stowaway stow --delete --target /home/me ~/dotfiles/bash

## How it works
Stowaway keeps track of each package installed in the `.stowaway` directory
inside the target directory. Inside this directory are a number of
subdirectories, each containing the state of an installed Stowaway package.

For each symlink that Stowaway creates, it creates another symlink pointing to
that symlink inside the package state directory. This allows Stoaway to track
which symlinks it has created, even when the contents of the package have been
modified.
