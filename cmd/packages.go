package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jamesbehr/stowaway/filesystem"
	"github.com/spf13/cobra"
)

var prefix string

var packagesCmd = &cobra.Command{
	Use:   "packages",
	Short: "List installed packages",
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

		state := targetPath.Join(".stowaway")

		files, err := state.ReadDir()
		if err != nil {
			log.Fatal(err)
		}

		if prefix != "" {
			prefix, err = filepath.Abs(prefix)
			if err != nil {
				log.Fatal(err)
			}
		}

		for _, file := range files {
			if !file.IsDir() {
				continue
			}

			source, err := state.Join(file.Name(), "source").Readlink()
			if err != nil {
				log.Fatal(err)
			}

			if prefix == "" || strings.HasPrefix(source.String(), prefix) {
				fmt.Println(source)
			}
		}
	},
}

func init() {
	packagesCmd.Flags().StringVarP(&target, "target", "t", "", "directory to list installed packages for (default is $PWD)")
	packagesCmd.Flags().StringVarP(&prefix, "prefix", "p", "", "only list packages that start with this")
}
