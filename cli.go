package main

import (
	"fmt"
	"regexp"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	deduceAlterverseFrom   string
	deduceAlterverseTo     string
	deduceAlterverseIgnore string
	deduceAlterverseDryRun bool
	deduceAlterverseSilent bool

	findContextsIn string
)

const defaultIgnore = `^.*[\\/]\..*|^\..*`

func init() {
	rootCmd.AddCommand(versionCmd)

	deduceAlterverseCmd.Flags().StringVarP(&deduceAlterverseFrom, "from", "f", "", "source alterverse path")
	deduceAlterverseCmd.MarkFlagRequired("from")
	deduceAlterverseCmd.Flags().StringVarP(&deduceAlterverseTo, "to", "t", "", "destination alterverse path")
	deduceAlterverseCmd.MarkFlagRequired("to")
	deduceAlterverseCmd.Flags().StringVar(&deduceAlterverseIgnore, "ignore", defaultIgnore, "if a filename matches this regexp it is ignored - by default all hidden files and directories (starting with a '.') are ignored")
	deduceAlterverseCmd.Flags().BoolVar(&deduceAlterverseDryRun, "dry-run", false, "only in-memory, no write to filesystem")
	deduceAlterverseCmd.Flags().BoolVar(&deduceAlterverseSilent, "silent", false, "mimimum output, no diff")
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
		from, errs := NewAlterverse(deduceAlterverseFrom, deduceAlterverseIgnore)
		exitOnErr(errs...)
		fromFiles, err := from.Files()
		exitOnErr(err)

		to, errs := NewAlterverse(deduceAlterverseTo, deduceAlterverseIgnore)
		exitOnErr(errs...)
		toFilesCurrent, err := to.Files()
		exitOnErr(err)

		interverse, err := NewInterverse(from.Manifest, to.Manifest)
		exitOnErr(err)
		toFilesNew := interverse.Deduce(fromFiles)

		if !deduceAlterverseSilent {
			diffs, toDelete, toCreate := DiffFiles(toFilesCurrent, toFilesNew)

			for filename, diff := range diffs {
				if diff == "" {
					fmt.Println(color.YellowString("--- file '%s' is unchanged.", filename))
				} else {
					fmt.Printf(color.MagentaString("--- file '%s' has changes:\n", filename)+"%s", diff)
				}
			}

			for filename := range toDelete {
				fmt.Println(color.RedString("--- file '%s' will be deleted in destination.", filename))
			}

			for filename := range toCreate {
				fmt.Println(color.GreenString("--- file '%s' will be created in destination.", filename))
			}
		}

		if !deduceAlterverseDryRun {
			fmt.Println("--- writing files")
			err = to.WriteFiles(toFilesNew)
			exitOnErr(err)
		} else {
			fmt.Println("--- dry-run NO files will be written")
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
		in, errs := NewAlterverse(findContextsIn, deduceAlterverseIgnore)
		exitOnErr(errs...)
		inData, err := in.Files()
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
