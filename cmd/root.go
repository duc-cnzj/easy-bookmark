package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "easy-bookmark",
	Short: "google 书签全文搜索工具",
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
