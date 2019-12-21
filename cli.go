package main

import (
	"fmt"
	"regexp"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	deduceAlterverseFrom   string
	deduceAlterverseTo     string
	deduceAlterverseIgnore string
	deduceAlterverseDryRun bool

	findContextsIn string
)

const defaultIgrore = `^.*[\\/]\..*|^\..*`

func init() {
	rootCmd.AddCommand(versionCmd)

	deduceAlterverseCmd.Flags().StringVarP(&deduceAlterverseFrom, "from", "f", "", "source alterverse path")
	deduceAlterverseCmd.MarkFlagRequired("from")
	deduceAlterverseCmd.Flags().StringVarP(&deduceAlterverseTo, "to", "t", "", "destination alterverse path")
	deduceAlterverseCmd.MarkFlagRequired("to")
	deduceAlterverseCmd.Flags().StringVarP(&deduceAlterverseIgnore, "ignore", "i", defaultIgrore, "if a filename matches this regexp it is ignored")
	deduceAlterverseCmd.Flags().BoolVar(&deduceAlterverseDryRun, "dry-run", false, "only in-memory, no write to filesystem")
	rootCmd.AddCommand(deduceAlterverseCmd)

	findContextsCmd.Flags().StringVar(&findContextsIn, "in", "", "alterverse path to check")
	rootCmd.AddCommand(findContextsCmd)
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

var findContextsCmd = &cobra.Command{
	Use:   "find-contexts",
	Short: "Finds each value given in the manifest file and gets the contexts in which those occure",
	Long: `To ensure the quality of your manifest find-contexts lets you find in which contexts (eg. 'words')
the strings defined in the manifest as values appear. This if a string is present an many diffierent
contexts you perhaps want to consider to have more manifest definitions which are more percise`,
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		in, errs := NewAlterverse(findContextsIn)
		exitOnErr(errs...)
		inSyncer, err := NewSyncer(findContextsIn, deduceAlterverseIgnore)
		exitOnErr(err)
		inData, err := inSyncer.ReadFiles()
		exitOnErr(err)

		contexts := map[string][]string{}
		regexStart, regexEnd := `(\b*`, `\b*)`
		for _, data := range inData {
			for key, value := range in.Manifest {
				re, err := regexp.Compile(fmt.Sprintf("%s%s%s", regexStart, regexp.QuoteMeta(value), regexEnd))
				exitOnErr(err)
				matches := re.FindAll(data, -1)
				for _, m := range matches {
					context := string(m)
					index := fmt.Sprintf("%s (%s)", key, value)
					if _, ok := contexts[index]; !ok {
						contexts[index] = []string{context}
					} else {
						found := false
						for _, v := range contexts[index] {
							if context == v {
								found = true
								break
							}
						}
						if !found {
							contexts[index] = append(contexts[index], context)
						}
					}
				}

			}
		}
		d, err := yaml.Marshal(&contexts)
		exitOnErr(err)
		fmt.Printf("---g \n%s\n\n", string(d))
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(versionInfo())
	},
}
