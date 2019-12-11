package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	rootConfigPath      string
	rootQuiet           bool
	rootSingularityPath string

	createAlterversTarget      string
	createAlterversDestination string
	createAlterversDryRun      bool

	deduceSingularityAlterverse string
	deduceSingularitySource     string
	deduceSingularityDryRun     bool
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&rootConfigPath, "config", "c", "omniverse.yaml", "configuration file for omniverse")

	rootCmd.PersistentFlags().BoolVar(&rootQuiet, "quiet", false, "omit log output")
	rootCmd.PersistentFlags().StringVar(&rootSingularityPath, "singularity", "singularity", "path of the singularity")

	createAlterverseCmd.Flags().StringVarP(&createAlterversTarget, "alterverse", "a", "", "name of the target alterverse (required)")
	createAlterverseCmd.MarkFlagRequired("alterverse")
	createAlterverseCmd.Flags().StringVarP(&createAlterversDestination, "destination", "d", "", "destination folder of the alterverse")
	createAlterverseCmd.MarkFlagRequired("destination")
	createAlterverseCmd.Flags().BoolVar(&createAlterversDryRun, "dry-run", false, "only in-memory, no write to filesystem")
	rootCmd.AddCommand(createAlterverseCmd)

	rootCmd.AddCommand(printConfigCmd)

	deduceSingularityCmd.Flags().StringVarP(&deduceSingularityAlterverse, "alterverse", "a", "", "name of the source alterverse (required)")
	deduceSingularityCmd.MarkFlagRequired("alterverse")
	deduceSingularityCmd.Flags().StringVarP(&deduceSingularitySource, "source", "s", "", "path of the source alterverse (required)")
	deduceSingularityCmd.MarkFlagRequired("source")
	deduceSingularityCmd.Flags().BoolVar(&deduceSingularityDryRun, "dry-run", false, "only in-memory, no write to filesystem")
	rootCmd.AddCommand(deduceSingularityCmd)

	rootCmd.AddCommand(listSingularityKeysCmd)
}

var rootCmd = &cobra.Command{
	Use:   "omniverse",
	Short: "Create a copy of a directory with deviations",
}

var createAlterverseCmd = &cobra.Command{
	Use:   "create-alterverse",
	Short: "Create alterverse from singularity",
	Run: func(cmd *cobra.Command, args []string) {
		l := NewLogger()

		cfg, err := NewConfig(rootConfigPath)
		exitOnErr(err)

		s := &cfg.Singularity
		err = s.Read(rootSingularityPath, l.Input)
		exitOnErr(err)

		a, err := cfg.Alterverses.GetAlterverse(createAlterversTarget)
		exitOnErr(err)

		errs := s.CheckIfKeysDefined(a.Definitions())
		for _, err = range errs {
			fmt.Fprintln(os.Stderr, err)
		}
		if len(errs) > 0 {
			os.Exit(-1)
		}

		errs = s.CheckIfDefinedIsKey(a.Definitions())
		for _, err = range errs {
			if !rootQuiet {
				fmt.Fprintf(os.Stderr, "WARNING: %s\n", err.Error())
			}
		}

		rendered, err := s.Generate(rootSingularityPath, a.Definitions(), l.Input)
		exitOnErr(err)

		if !createAlterversDryRun {
			err = a.Write(rendered, createAlterversDestination, l.Input)
			exitOnErr(err)
		}

		l.Input <- "Done"
		l.Quit <- true
	},
}

var listSingularityKeysCmd = &cobra.Command{
	Use:   "list-singularity-keys",
	Short: "Discover and list keys which are defined in singularity",
	Run: func(cmd *cobra.Command, args []string) {
		l := NewLogger()

		cfg, err := NewConfig(rootConfigPath)
		exitOnErr(err)

		s := &cfg.Singularity
		err = s.Read(rootSingularityPath, l.Input)
		exitOnErr(err)

		b, err := yaml.Marshal(s.GetKeys())
		exitOnErr(err)

		fmt.Println(string(b))
	},
}

var printConfigCmd = &cobra.Command{
	Use:   "print-config",
	Short: "Print the configuration as parsed by omniverse",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := NewConfig(rootConfigPath)
		exitOnErr(err)

		b, err := yaml.Marshal(cfg)
		exitOnErr(err)
		fmt.Println(string(b))
	},
}

var deduceSingularityCmd = &cobra.Command{
	Use:   "deduce-singularity",
	Short: "Deduce singularity from alterverse",
	Run: func(cmd *cobra.Command, args []string) {
		l := NewLogger()

		cfg, err := NewConfig(rootConfigPath)
		exitOnErr(err)

		s := &cfg.Singularity

		a, err := cfg.Alterverses.GetAlterverse(deduceSingularityAlterverse)
		exitOnErr(err)

		err = a.Read(deduceSingularitySource, l.Input)
		exitOnErr(err)

		rendered, err := a.SubstituteDefinitions(s.ExpressionTemplate, l.Input)
		exitOnErr(err)

		if !deduceSingularityDryRun {
			err = s.Write(rendered, rootSingularityPath, l.Input)
			exitOnErr(err)
		}

		l.Input <- "Done"
		l.Quit <- true
	},
}

func main() {
	err := rootCmd.Execute()
	exitOnErr(err)
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}
