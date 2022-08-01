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

var target = ""

func hash(path string) string {
	h := md5.Sum([]byte(path))
	digest := hex.EncodeToString(h[:])
	return digest[:6]
}

var installCmd = &cobra.Command{
	Use:   "install",
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

		for _, arg := range args {
			path, err := filepath.Abs(arg)
			if err != nil {
				log.Fatal(err)
			}

			installer := pkg.Installer{
				State:  targetPath.Join(".stowaway", hash(path)),
				Source: filesystem.MakePath(path),
				Target: targetPath,
			}

			installed, err := installer.Installed()
			if err != nil {
				log.Fatal(err)
			}

			if installed {
				if err := installer.Uninstall(); err != nil {
					log.Fatal(err)
				}
			}

			if err := installer.Install(); err != nil {
				log.Fatal(err)
			}
		}
	},
}

func init() {
	installCmd.Flags().StringVar(&target, "target", "", "installation target (default is $PWD)")
}
