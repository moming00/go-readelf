package main

import (
	"flag"
	"go-readelf/debuginfo"
	"go-readelf/util"
	"os"

	"go.uber.org/zap"
)

type CmdOptions struct {
	RootDir   string
	BinaryDir string
}

var cmdOps CmdOptions

func main() {
	if cmdOps.BinaryDir == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if _, err := os.Stat(cmdOps.BinaryDir); err != nil {
		util.Logger.Fatal("file not present", zap.String("binary directory", cmdOps.BinaryDir), zap.Error(err))
	}
	if cmdOps.RootDir != "" {
		if _, err := os.Stat(cmdOps.RootDir); err != nil {
			util.Logger.Fatal("directory not present", zap.String("binary directory", cmdOps.RootDir), zap.Error(err))
		}
	}

	f := debuginfo.NewFinder([]string{"/usr/lib/debug", "/usr/src/debug"})
	if debugfile, err := f.FindSeperateDbgFile(cmdOps.RootDir, cmdOps.BinaryDir); err != nil {
		util.Logger.Fatal("seperate debug info not found")
	} else {
		util.Logger.Info("seperate debug info file located", zap.String("path", debugfile))
	}
}

func init() {
	cmdOps = CmdOptions{}
	flag.StringVar(&cmdOps.RootDir, "r", cmdOps.RootDir, "root directory")
	flag.StringVar(&cmdOps.BinaryDir, "b", cmdOps.BinaryDir, "target binary directory")
	flag.Parse()

	util.InitLogger([]string{"go-readelf.log", "stdout"})
}
