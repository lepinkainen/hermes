/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/lepinkainen/hermes/cmd/imdb"
	"github.com/lepinkainen/hermes/cmd/steam"
	"github.com/spf13/cobra"
)

// importCmd represents the import command
// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import data from various sources",
	Long:  `Import data from various sources into the system.`,
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.AddCommand(imdb.GetCommand())
	importCmd.AddCommand(steam.GetCommand())
}
