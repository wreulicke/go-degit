package main

import (
	"errors"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path"
	"strings"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "degit",
		Short: "degit is scafollding tool using git repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("repository is not specified")
			}
			u, err := toURL(args[0])
			if err != nil {
				return err
			}
			prefix := ""
			dest := path.Base(strings.TrimSuffix(u.Path, ".git"))
			if len(args) == 2 {
				prefix = args[1]
				if !strings.Contains(prefix, "/") {
					dest = prefix
				}
			} else if len(args) == 3 {
				prefix = args[1]
				dest = args[2]
			}
			return Clone(u, prefix, dest)
		},
	}
	return cmd
}

func main() {
	if err := NewCommand().Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
