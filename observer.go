package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/radovskyb/watcher"
)

type Observer struct {
	w       *watcher.Watcher
	basedir string
	ignore  *regexp.Regexp
}

func NewObserver(dir, ignored string) (*Observer, error) {
	basedir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(ignored)
	if err != nil {
		return nil, err
	}

	w := watcher.New()
	w.SetMaxEvents(1)

	o := &Observer{
		basedir: basedir,
		ignore:  re,
		w:       w,
	}

	return o, nil
}

func (o *Observer) Run() error {
	go func() {
		for {
			select {
			case event := <-o.w.Event:
				for path, f := range o.GetFileList() {
					fmt.Printf("%s: %s\n", path, f.Name())
				}
				fmt.Println(event)
			case err := <-o.w.Error:
				log.Fatalln(err)
			case <-o.w.Closed:
				return
			}
		}
	}()

	if err := o.w.AddRecursive(o.basedir); err != nil {
		return err
	}

	if err := o.w.Start(time.Millisecond * 100); err != nil {
		return err
	}

	return nil
}

func (o *Observer) GetFileList() map[string]os.FileInfo {
	watched := o.w.WatchedFiles()

	files := map[string]os.FileInfo{}
	for path, info := range watched {
		fmt.Println(path)
		path = strings.TrimPrefix(path, o.basedir)
		path = strings.TrimPrefix(path, "/")
		path = strings.TrimPrefix(path, "\\")

		if info.IsDir() || o.isIgnored(path) {
			return nil
		}
		files[path] = info
	}

	return files
}

func (o Observer) isIgnored(path string) bool {
	return o.ignore.MatchString(path)
}
