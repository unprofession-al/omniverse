package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	deduceAlterverseFrom   string
	deduceAlterverseTo     string
	deduceAlterverseIgnore []string
	deduceAlterverseDryRun bool
)

func init() {
	rootCmd.AddCommand(versionCmd)

	deduceAlterverseCmd.Flags().StringVarP(&deduceAlterverseFrom, "from", "f", "", "source alterverse path")
	deduceAlterverseCmd.MarkFlagRequired("from")
	deduceAlterverseCmd.Flags().StringVarP(&deduceAlterverseTo, "to", "t", "", "destination alterverse path")
	deduceAlterverseCmd.MarkFlagRequired("to")
	deduceAlterverseCmd.Flags().StringSliceVarP(&deduceAlterverseIgnore, "ignore", "i", []string{`/.`, `\.`, `.git`, alterverseFile}, "patterns to ignore")
	deduceAlterverseCmd.Flags().BoolVar(&deduceAlterverseDryRun, "dry-run", false, "only in-memory, no write to filesystem")
	rootCmd.AddCommand(deduceAlterverseCmd)
}

var rootCmd = &cobra.Command{
	Use:   "omniverse",
	Short: "Create a copy of a directory with deviations",
}

var deduceAlterverseCmd = &cobra.Command{
	Use:   "deduce-alterverse",
	Short: "Deduce an alterverse",
	Run: func(cmd *cobra.Command, args []string) {
		from, errs := NewAlterverse(deduceAlterverseFrom)
		exitOnErr(errs...)
		fromSyncer, err := NewSyncer(deduceAlterverseFrom, deduceAlterverseIgnore)
		exitOnErr(err)
		fromData, err := fromSyncer.ReadFiles()
		exitOnErr(err)

		to, errs := NewAlterverse(deduceAlterverseTo)
		exitOnErr(errs...)
		toSyncer, err := NewSyncer(deduceAlterverseTo, deduceAlterverseIgnore)
		exitOnErr(err)

		interverse, err := NewInterverse(from.Manifest, to.Manifest)
		exitOnErr(err)
		toData := interverse.Do(fromData)

		if !deduceAlterverseDryRun {
			deleteObselete := true
			err = toSyncer.WriteFiles(toData, deleteObselete)
			exitOnErr(err)
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(versionInfo())
	},
}
