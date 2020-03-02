package main

import (
	"fmt"
	"regexp"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

const defaultIgnore = `^.*[\\/]\..*|^\..*`

type App struct {
	// config
	cfg struct {
		deduceFrom    string
		deduceTo      string
		deduceIgnore  string
		deduceDryRun  bool
		deduceSilent  bool
		contextsIn    string
		observeIn     string
		observeIgnore string
	}

	// entry point
	Execute func() error
}

func NewApp() *App {
	a := &App{}

	// root
	rootCmd := &cobra.Command{
		Use:   "omniverse",
		Short: "Create a copy of a directory with deviations",
	}
	a.Execute = rootCmd.Execute

	// deduce
	deduceCmd := &cobra.Command{
		Use:   "deduce",
		Short: "Deduce an alterverse",
		Run:   a.deduceCmd,
	}
	deduceCmd.Flags().StringVarP(&a.cfg.deduceFrom, "from", "f", "", "source alterverse path")
	deduceCmd.MarkFlagRequired("from")
	deduceCmd.Flags().StringVarP(&a.cfg.deduceTo, "to", "t", "", "destination alterverse path")
	deduceCmd.MarkFlagRequired("to")
	deduceCmd.Flags().StringVar(&a.cfg.deduceIgnore, "ignore", defaultIgnore, "if a filename matches this regexp it is ignored - by default all hidden files and directories (starting with a '.') are ignored")
	deduceCmd.Flags().BoolVar(&a.cfg.deduceDryRun, "dry-run", false, "only in-memory, no write to filesystem")
	deduceCmd.Flags().BoolVar(&a.cfg.deduceSilent, "silent", false, "mimimum output, no diff")
	rootCmd.AddCommand(deduceCmd)

	// contexts
	contextsCmd := &cobra.Command{
		Use:   "contexts",
		Short: "Finds each value given in the manifest file and gets the contexts in which those occure",
		Long: `To ensure the quality of your manifest find-contexts lets you find in which contexts (eg. 'words')
the strings defined in the manifest as values appear. This if a string is present an many diffierent
contexts you perhaps want to consider to have more manifest definitions which are more percise`,
		Hidden: true,
		Run:    a.contextsCmd,
	}
	contextsCmd.Flags().StringVar(&a.cfg.contextsIn, "in", ".", "alterverse path to check")
	rootCmd.AddCommand(contextsCmd)

	// observe
	observeCmd := &cobra.Command{
		Use:    "observe",
		Short:  "Observes changes and prints information to guard consintency",
		Long:   ``,
		Hidden: true,
		Run:    a.observeCmd,
	}
	observeCmd.Flags().StringVar(&a.cfg.observeIn, "in", ".", "alterverse path to observe")
	observeCmd.Flags().StringVar(&a.cfg.observeIgnore, "ignore", defaultIgnore, "if a filename matches this regexp it is ignored - by default all hidden files and directories (starting with a '.') are ignored")
	rootCmd.AddCommand(observeCmd)

	// version
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version info",
		Run:   a.versionCmd,
	}
	rootCmd.AddCommand(versionCmd)

	return a
}

func (a *App) deduceCmd(cmd *cobra.Command, args []string) {
	from, errs := NewAlterverse(a.cfg.deduceFrom, a.cfg.deduceIgnore)
	exitOnErr(errs...)
	fromFiles, err := from.Files()
	exitOnErr(err)

	to, errs := NewAlterverse(a.cfg.deduceTo, a.cfg.deduceIgnore)
	exitOnErr(errs...)
	toFilesCurrent, err := to.Files()
	exitOnErr(err)

	interverse, err := NewInterverse(from.Manifest, to.Manifest)
	exitOnErr(err)
	toFilesNew, errs := interverse.DeduceStrict(fromFiles)
	exitOnErr(errs...)

	if !a.cfg.deduceSilent {
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

	if !a.cfg.deduceDryRun {
		fmt.Println("--- writing files")
		err = to.WriteFiles(toFilesNew)
		exitOnErr(err)
	} else {
		fmt.Println("--- dry-run NO files will be written")
	}
}

func (a *App) contextsCmd(cmd *cobra.Command, args []string) {
	in, errs := NewAlterverse(a.cfg.contextsIn, a.cfg.deduceIgnore)
	exitOnErr(errs...)
	inData, err := in.Files()
	exitOnErr(err)

	contexts := map[string][]string{}
	regexStart, regexEnd := `(\b[\w-_\.]*`, `[\w-_\.]*\b*)`
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
	fmt.Printf("--- \n%s\n\n", string(d))
}

func (a *App) observeCmd(cmd *cobra.Command, args []string) {
	o, err := NewObserver(a.cfg.observeIn, a.cfg.observeIgnore)
	exitOnErr(err)

	err = o.Run()
	exitOnErr(err)
}

func (a *App) versionCmd(cmd *cobra.Command, args []string) {
	fmt.Println(versionInfo())
}
