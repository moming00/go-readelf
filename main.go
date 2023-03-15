package main

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"encoding/hex"
	"go-readelf/util"
	"os"
	"strings"

	"go.uber.org/zap"
)

type buildIDHeader struct {
	Namesz uint32
	Descsz uint32
	Type   uint32
}

func find_debuglink(e *elf.File) (debugLink string, crc uint32) {
	sec := e.Section(".gnu_debuglink")
	if sec == nil {
		util.Logger.Info("no .gnu_debuglink section found")
		return "", 0
	}

	buf, err := sec.Data()
	if err != nil {
		util.Logger.Warn("can't read .gnu_debuglink", zap.Error(err))
	}

	parts := bytes.Split(buf, []byte{0})
	debugLink = string(parts[0])
	checksum := parts[len(parts)-1]
	if len(checksum) != 4 {
		util.Logger.Warn("incorrect .gnu_debuglink format", zap.String("buf", string(buf)))
	}

	crc = e.FileHeader.ByteOrder.Uint32(checksum)
	if crc == 0 {
		util.Logger.Warn("invalid crc")
	}

	return
}

func find_buildid(e *elf.File) string {
	// findBuildID := func(notes []elfreader.ElfNote) ([]byte, error) {
	// 	var buildID []byte
	// 	for _, note := range notes {
	// 		if note.Name == "GNU" && note.Type == elfreader.NoteTypeGNUBuildID {
	// 			if buildID == nil {
	// 				buildID = note.Desc
	// 			} else {
	// 				return nil, fmt.Errorf("multiple build ids found, don't know which to use")
	// 			}
	// 		}
	// 	}
	// 	return buildID, nil
	// }

	// s := e.Section(".note.gnu.build-id")
	// if s == nil {
	// 	util.Logger.Info("no .note.gnu.build-id section found")
	// 	return ""
	// }

	// notes, err := elfreader.ParseNotes(s.Open(), int(s.Addralign), e.ByteOrder)
	// if err != nil {
	// 	util.Logger.Info("failed to parse .note.gnu.build-id section", zap.Error(err))
	// }
	// if b, err := findBuildID(notes); b != nil {
	// 	return hex.EncodeToString(b)
	// } else {
	// 	util.Logger.Info("failed to find buildid", zap.Error(err))
	// }
	// return ""

	buildid := e.Section(".note.gnu.build-id")
	if buildid == nil {
		util.Logger.Info("no .note.gnu.build-id section found")
		return ""
	}

	br := buildid.Open()
	bh := new(buildIDHeader)
	if err := binary.Read(br, e.ByteOrder, bh); err != nil {
		util.Logger.Warn("can't read build-id header", zap.Error(err))
		return ""
	}

	name := make([]byte, bh.Namesz)
	if err := binary.Read(br, e.ByteOrder, name); err != nil {
		util.Logger.Warn("can't read build-id name", zap.Error(err))
		return ""
	}

	if strings.TrimSpace(string(name)) != "GNU\x00" {
		util.Logger.Warn("invalid build-id signature")
		return ""
	}

	descBinary := make([]byte, bh.Descsz)
	if err := binary.Read(br, e.ByteOrder, descBinary); err != nil {
		util.Logger.Warn("can't read build-id desc", zap.Error(err))
		return ""
	}

	return hex.EncodeToString(descBinary)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	util.InitLogger([]string{"go-readelf.log", "stdout"})
	util.SetLogLevel("info")

	file, err := elf.Open(os.Args[1])
	if err != nil {
		util.Logger.Fatal("elf.NewFile failed", zap.Error(err))
	}

	id := find_buildid(file)
	if id != "" {
		util.Logger.Info("find_buildid result", zap.String("buildid", id))
	}
	link, csc := find_debuglink(file)
	if link != "" {
		util.Logger.Info("find_debuglink result", zap.String("debuglink", link), zap.Uint32("CSC", csc))
	}
}

func usage() {
	// fmt.Printf("Usage: %s [-hrsS] <target-binary>\n", os.Args[0])
	// fmt.Println("\t-h: View Elf header")
	// fmt.Println("\t-r: View relocation entries")
	// fmt.Println("\t-s: View symbols")
	// fmt.Println("\t-S: View Sections")
	// fmt.Println("\t-l: View program headers")
}
