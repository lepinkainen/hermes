package cmd

import (
	"github.com/spf13/cobra"

	"github.com/lepinkainen/hermes/cmd/goodreads"
	"github.com/lepinkainen/hermes/cmd/imdb"
	"github.com/lepinkainen/hermes/cmd/steam"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import data from various sources",
	Long:  `Import data from various sources into the system.`,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.AddCommand(imdb.GetCommand())
	importCmd.AddCommand(steam.GetCommand())
	importCmd.AddCommand(goodreads.GetCommand())
}
