/*
Package conf provides config loader able to load configs using different strategies
*/
package conf

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/kardianos/osext"
)

//Configurable config loader :)
type Loader struct {
	//RootPath is a location where loader search for config files.
	//By default it is set to current working directory.
	RootPath string

	//Number of arguments that should not be considered as config paths.
	//It is only used with UseArgumentPaths flag.
	//Defaults to UseFlag.
	PreservedArgs int

	lookupPaths  []string
	loadedPaths  []string
	skippedPaths []string

	loaderFlags int
}

const (
	//Uses test.json mixin path if executable ends with .test
	UseTest int = 1 << iota

	//Reads user name from .user file if it exists.
	//File lookup is within RootPath.
	UseDotUser int = 1 << iota

	//Reads config paths from arguments passed to executable
	//If number of arguments is not greater than PreservedArgs
	//or if PreservedArgs is set to UseFlag (default) and len(flag.Args()) == 0
	//it fallbacks to default behaviour.
	UseArgumentPaths int = 1 << iota

	//Use the folder where executable is located as RootPath
	UseExecutablePath int = 1 << iota

	//Populates SkippedPaths instead of returning error on missing config files
	IgnoreMissingFiles int = 1 << iota

	//Populates SkippedPaths instead of returning error on invalid JSON files
	IgnoreInvalidFiles int = 1 << iota
)

const (
	//Use flag.Args as config paths passed by arguments
	UseFlag int = -1
)

//Creates new loader.
//NewLoader can return error if it fail to identify executable folder
//and UseExecutablePath flag is set.
func NewLoader(flags int) (*Loader, error) {
	loader := &Loader{
		loaderFlags:   flags,
		PreservedArgs: UseFlag,
	}

	if loader.Implements(UseExecutablePath) {
		executableFolder, err := osext.ExecutableFolder()
		if err != nil {
			return nil, err
		}
		loader.RootPath = executableFolder
	} else {
		loader.RootPath = "."
	}

	return loader, nil
}

//Loads config into variable passed.
//It may return error if config file is missing or invalid and loader
//has no IgnoreXXX flags set.
func (l *Loader) Load(config interface{}) error {
	l.createLookupPaths()

	l.loadedPaths = []string{}
	l.skippedPaths = []string{}

	for _, configPath := range l.lookupPaths {
		configData, err := ioutil.ReadFile(configPath)
		if err != nil {
			if !l.Implements(IgnoreMissingFiles) {
				return err
			}
			l.skippedPaths = append(l.skippedPaths, configPath)
			continue
		}

		err = json.Unmarshal(configData, config)
		if err != nil {
			if !l.Implements(IgnoreInvalidFiles) {
				return err
			}
			l.skippedPaths = append(l.skippedPaths, configPath)
			continue
		}

		l.loadedPaths = append(l.loadedPaths, configPath)
	}

	return nil
}

//Checks if loader has flag set.
func (l *Loader) Implements(behaviour int) bool {
	return l.loaderFlags&behaviour > 0
}

//Returns config files successfuly loaded in previous Load call.
func (l *Loader) LoadedPaths() []string {
	return l.loadedPaths
}

//Returns config files skipped in previous Load call.
func (l *Loader) SkippedPaths() []string {
	return l.skippedPaths
}

func (l *Loader) createLookupPaths() {
	if l.Implements(UseArgumentPaths) {
		if l.PreservedArgs == UseFlag {
			flagArgs := flag.Args()
			if len(flagArgs) > 0 {
				l.lookupPaths = flagArgs
			}
		} else {
			splitSize := l.PreservedArgs + 1
			if len(os.Args) > splitSize {
				l.lookupPaths = os.Args[splitSize:]
				return
			}
		}
	}

	l.lookupPaths = []string{
		filepath.Join(l.RootPath, "config.json"),
	}

	if l.Implements(UseTest) && l.isTest() {
		l.lookupPaths = append(l.lookupPaths, filepath.Join(l.RootPath, "config", "mixins", "test.json"))
	} else {
		user := l.user()
		if len(user) > 0 {
			l.lookupPaths = append(l.lookupPaths, filepath.Join(l.RootPath, "config", "mixins", fmt.Sprintf("%s.json", user)))
		}
	}
}

func (l *Loader) user() string {
	if l.Implements(UseDotUser) {
		fileContents, err := ioutil.ReadFile(filepath.Join(l.RootPath, ".user"))
		if err == nil {
			return strings.TrimSpace(string(fileContents))
		}
	}

	user, err := user.Current()
	if err != nil {
		return ""
	}

	return user.Username
}

func (l *Loader) isTest() bool {
	runfile := os.Args[0]
	return runfile[len(runfile)-5:] == ".test"
}
