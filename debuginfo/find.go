// Copyright 2022-2023 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package debuginfo

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"go-readelf/util"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/goburrow/cache"
	"go.uber.org/zap"
)

type realfs struct{}
type buildIDHeader struct {
	Namesz uint32
	Descsz uint32
	Type   uint32
}

func (f *realfs) Open(name string) (fs.File, error) { return os.Open(name) }

var fileSystem fs.FS = &realfs{}
var errNotFound = fmt.Errorf("not found")

// Finder finds the separate debug information files on the system.
type Finder struct {
	cache     cache.Cache
	debugDirs []string
}

// NewFinder creates a new Finder.
func NewFinder(debugDirs []string) *Finder {
	return &Finder{
		cache:     cache.New(cache.WithMaximumSize(128)),
		debugDirs: debugDirs,
	}
}

func find_debuglink(e *elf.File) (debugLink string, crc uint32) {
	sec := e.Section(".gnu_debuglink")
	if sec == nil {
		util.Logger.Debug("no .gnu_debuglink section found")
		return "", 0
	}

	buf, err := sec.Data()
	if err != nil {
		util.Logger.Debug("can't read .gnu_debuglink", zap.Error(err))
	}

	parts := bytes.Split(buf, []byte{0})
	debugLink = string(parts[0])
	checksum := parts[len(parts)-1]
	if len(checksum) != 4 {
		util.Logger.Debug("incorrect .gnu_debuglink format", zap.String("buf", string(buf)))
	}

	crc = e.FileHeader.ByteOrder.Uint32(checksum)
	if crc == 0 {
		util.Logger.Debug("invalid crc")
	}

	return
}

func find_buildid(e *elf.File) (string, error) {
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
		return "", fmt.Errorf("no .note.gnu.build-id section found")
	}

	br := buildid.Open()
	bh := new(buildIDHeader)
	if err := binary.Read(br, e.ByteOrder, bh); err != nil {
		return "", fmt.Errorf("can't read build-id header: %+v", err)
	}

	name := make([]byte, bh.Namesz)
	if err := binary.Read(br, e.ByteOrder, name); err != nil {
		return "", fmt.Errorf("can't read build-id name: %+v", err)
	}

	if strings.TrimSpace(string(name)) != "GNU\x00" {
		return "", fmt.Errorf("invalid build-id signature")
	}

	descBinary := make([]byte, bh.Descsz)
	if err := binary.Read(br, e.ByteOrder, descBinary); err != nil {
		return "", fmt.Errorf("can't read build-id desc: %+v", err)
	}

	return hex.EncodeToString(descBinary), nil
}

func (fd *Finder) FindSeperateDbgFile(root, file string) (path string, err error) {
	if file, err = filepath.Abs(file); err != nil {
		return "", fmt.Errorf("cannot locate file %s", file)
	}
	elffile, err := elf.Open(file)
	if err != nil {
		util.Logger.Error("elf.NewFile failed", zap.Error(err))
		return "", err
	}

	buildID, err := find_buildid(elffile)
	if err != nil {
		return "", err
	}

	if val, ok := fd.cache.GetIfPresent(buildID); ok {
		switch v := val.(type) {
		case string:
			return v, nil
		case error:
			return "", v
		default:
			// We didn't put you there?!
			return "", errors.New("unexpected type")
		}
	}

	ret, err := fd.Find(root, buildID, file, elffile)
	if err != nil {
		if errors.Is(err, errNotFound) {
			fd.cache.Put(buildID, err)
			return "", err
		}
	}

	fd.cache.Put(buildID, ret)
	return ret, nil
}

func (fd *Finder) Find(root, buildID, file string, elffile *elf.File) (path string, err error) {
	if len(buildID) < 2 {
		return "", fmt.Errorf("invalid buildid %s", buildID)
	}

	link, crc := find_debuglink(elffile)
	if len(link) == 0 {
		link = filepath.Base(file)
	}

	const dbgExt = ".debug"
	ext := filepath.Ext(link)
	if ext == "" {
		ext = dbgExt
	}
	dbgFilePath := filepath.Join(filepath.Dir(file), link)

	var files []string
	files = append(files, []string{dbgFilePath, filepath.Join(filepath.Dir(file), dbgExt, filepath.Base(dbgFilePath))}...)
	for _, dir := range fd.debugDirs {
		files = append(files, []string{
			filepath.Join(root, dir, dbgFilePath),
			filepath.Join(root, dir, ".build-id", buildID[:2], buildID[2:]) + dbgExt,
			filepath.Join(root, dir, buildID, "debuginfo"),
		}...)
	}
	if len(files) == 0 {
		return "", fmt.Errorf("failed to generate paths")
	}

	var found string
	for _, f := range files {
		if _, err := fs.Stat(fileSystem, f); err == nil {
			found = f
			break
		}
	}

	if found == "" {
		return "", errNotFound
	}
	// verifying CRC is only required for debuglink
	if strings.Contains(found, ".build-id") || strings.HasSuffix(found, "/debuginfo") || crc <= 0 {
		return found, nil
	}

	match, err := checkSum(found, crc)
	if err != nil {
		return "", fmt.Errorf("failed to check checksum: %w", err)
	}

	if match {
		return found, nil
	}

	return "", errNotFound
}

func checkSum(path string, crc uint32) (bool, error) {
	file, err := fileSystem.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	d, err := io.ReadAll(file)
	if err != nil {
		return false, err
	}
	return crc == crc32.ChecksumIEEE(d), nil
}
