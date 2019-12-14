package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	version string
	commit  string
	date    string

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

	rootCmd.AddCommand(versionCmd)
}

var rootCmd = &cobra.Command{
	Use:   "omniverse",
	Short: "Create a copy of a directory with deviations",
}

var createAlterverseCmd = &cobra.Command{
	Use:   "create-alterverse",
	Short: "Create alterverse from singularity",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, valErrs, err := NewConfig(rootConfigPath)
		exitOnErr(append(valErrs, err)...)

		ss, err := NewSyncer(rootSingularityPath, []string{})
		exitOnErr(err)

		sd, err := ss.ReadFiles()
		exitOnErr(err)

		s, err := NewSingularity(cfg.Singularity, sd)
		exitOnErr(err)

		a, err := cfg.Alterverses.GetAlterverse(createAlterversTarget, map[string][]byte{})
		exitOnErr(err)

		checker := NewChecker()
		errs := checker.ValidateSingularityIfKeysDefined(*s, a.Definitions())
		exitOnErr(errs...)

		errs = checker.ValidateDefinitionIfDefinitionsAreObsolete(a.Definitions(), *s)
		for _, err = range errs {
			log.Warn(err.Error())
		}

		rendered, err := s.Generate(rootSingularityPath, a.Definitions())
		exitOnErr(err)

		if !createAlterversDryRun {
			ignore := []string{}
			as, err := NewSyncer(createAlterversDestination, ignore)
			exitOnErr(err)

			deleteObsolete := true
			err = as.WriteFiles(rendered, deleteObsolete)
			exitOnErr(err)
		}

		log.Info("Done")
	},
}

var listSingularityKeysCmd = &cobra.Command{
	Use:   "list-singularity-keys",
	Short: "Discover and list keys which are defined in singularity",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, valErrs, err := NewConfig(rootConfigPath)
		exitOnErr(append(valErrs, err)...)

		ss, err := NewSyncer(rootSingularityPath, []string{})
		exitOnErr(err)

		sd, err := ss.ReadFiles()
		exitOnErr(err)

		s, err := NewSingularity(cfg.Singularity, sd)
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
		cfg, valErrs, err := NewConfig(rootConfigPath)
		exitOnErr(append(valErrs, err)...)

		b, err := yaml.Marshal(cfg)
		exitOnErr(err)

		fmt.Println(string(b))
	},
}

var deduceSingularityCmd = &cobra.Command{
	Use:   "deduce-singularity",
	Short: "Deduce singularity from alterverse",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, valErrs, err := NewConfig(rootConfigPath)
		exitOnErr(append(valErrs, err)...)

		checker := NewChecker()

		s, err := NewSingularity(cfg.Singularity, map[string][]byte{})
		exitOnErr(err)

		ignore := []string{}
		as, err := NewSyncer(deduceSingularitySource, ignore)
		exitOnErr(err)

		af, err := as.ReadFiles()
		exitOnErr(err)

		a, err := cfg.Alterverses.GetAlterverse(deduceSingularityAlterverse, af)
		exitOnErr(err)

		errs := checker.ValidateEqualDefinitonValues(a.Definitions())
		exitOnErr(errs...)

		rendered, err := a.SubstituteDefinitions(s.ExpressionTemplate)
		exitOnErr(err)

		if !deduceSingularityDryRun {
			ignore := []string{}
			ss, err := NewSyncer(rootSingularityPath, ignore)
			exitOnErr(err)

			deleteObsolete := true
			err = ss.WriteFiles(rendered, deleteObsolete)
			exitOnErr(err)
		}

		log.Info("Done")
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version info",
	Run: func(cmd *cobra.Command, args []string) {
		if version == "" {
			version = "dirty"
		}
		if commit == "" {
			commit = "dirty"
		}
		if date == "" {
			date = "unknown"
		}
		fmt.Printf("Version:    %s\nCommit:     %s\nBuild Date: %s\n", version, commit, date)
	},
}
