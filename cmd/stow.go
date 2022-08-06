package cmd

import (
	"crypto/md5"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"

	"github.com/jamesbehr/stowaway/filesystem"
	"github.com/jamesbehr/stowaway/pkg"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

var target string
var interactive bool
var options pkg.StowOptions

func hash(path string) string {
	h := md5.Sum([]byte(path))
	digest := hex.EncodeToString(h[:])
	return digest[:6]
}

func interactiveFilter(paths []string, packages []pkg.Package) ([]pkg.Package, error) {
	var selected []string
	prompt := &survey.MultiSelect{
		Message: "Choose packages to install",
		Options: paths,
	}

	survey.AskOne(prompt, &selected, survey.Required)

	filtered := make([]pkg.Package, len(selected))
	for i, path := range selected {
		var index int
		for i := range paths {
			if paths[i] == path {
				index = i
			}
		}

		filtered[i] = packages[index]
	}

	return filtered, nil
}

var stowCmd = &cobra.Command{
	Use:   "stow",
	Short: "Install a package",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatal("provide at least one package path")
		}

		pwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		if target == "" {
			target = pwd
		} else {
			target, err = filepath.Abs(target)
			if err != nil {
				log.Fatal(err)
			}
		}

		targetPath := filesystem.MakePath(target)

		var packages []pkg.Package
		for _, arg := range args {
			path, err := filepath.Abs(arg)
			if err != nil {
				log.Fatal(err)
			}

			loader := pkg.Loader{
				State:  targetPath.Join(".stowaway", hash(path)),
				Source: filesystem.MakePath(path),
				Target: targetPath,
			}

			pkg, err := loader.Load()
			if err != nil {
				log.Fatal(err)
			}

			packages = append(packages, pkg)
		}

		if interactive {
			packages, err = interactiveFilter(args, packages)
			if err != nil {
				log.Fatal(err)
			}
		}

		if err = pkg.Stow(options, packages...); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	stowCmd.Flags().StringVarP(&target, "target", "t", "", "installation target (default is $PWD)")
	stowCmd.Flags().BoolVarP(&interactive, "interacitve", "i", false, "start an interactive session to filter the packages passed as arguments before installing")
	stowCmd.Flags().BoolVarP(&options.Delete, "delete", "D", false, "uninstall the packages")
}
