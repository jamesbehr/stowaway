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
)

var target string
var options pkg.StowOptions

func hash(path string) string {
	h := md5.Sum([]byte(path))
	digest := hex.EncodeToString(h[:])
	return digest[:6]
}

var stowCmd = &cobra.Command{
	Use:   "stow",
	Short: "Install a package",
	Run: func(cmd *cobra.Command, args []string) {
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

		if err = pkg.Stow(options, packages...); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	stowCmd.Flags().StringVarP(&target, "target", "t", "", "installation target (default is $PWD)")
	stowCmd.Flags().BoolVarP(&options.Delete, "delete", "D", false, "uninstall the packages")
}
