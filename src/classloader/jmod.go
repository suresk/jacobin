/*
 * Jacobin VM - A Java virtual machine
 * Copyright (c) 2021-2 by Andrew Binstock. All rights reserved.
 * Licensed under Mozilla Public License 2.0 (MPL 2.0)
 */

package classloader

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"jacobin/log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type WalkEntryFunc func(bytes []byte, filename string) error

// MagicNumber JMOD Magic Number
const MagicNumber = 0x4A4D

// Jmod Holds the file referring to a Java Module (JMOD)
// Allows walking a Java Module (JMOD). The `Walk` method will walk the module and invoke the `walk` parameter for all
// classes found. If there is a classlist file in lib\classlist (in the module), it will filter out any classes not
// contained in the classlist file; otherwise, all classes found in classes/ in the module.
type Jmod struct {
	FileName      string
	entryListOnce sync.Once
	entries       map[string]string
}

func InitJmod(fileName string) *Jmod {
	return &Jmod{
		FileName:      fileName,
		entryListOnce: sync.Once{},
		entries:       make(map[string]string),
	}
}

func getZipReader(fileName string) (*zip.Reader, error) {
	b, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	fileMagic := binary.BigEndian.Uint16(b[:2])

	if fileMagic != MagicNumber {
		err := errors.New(fmt.Sprintf("An IOException occurred reading %s: the magic number is invalid. Expected: %x, Got: %x", fileName, MagicNumber, fileMagic))
		return nil, err
	}

	// Skip over the JMOD header so that it is recognized as a ZIP file
	offsetReader := bytes.NewReader(b[4:])

	return zip.NewReader(offsetReader, int64(len(b)-4))
}

func (j *Jmod) LoadByName(name string) ([]byte, error) {
	reader, err := getZipReader(j.FileName)
	if err != nil {
		return nil, err
	}

	j.entryListOnce.Do(func() {
		for _, f := range reader.File {
			if !strings.HasPrefix(f.Name, "classes") {
				continue
			}

			classFileName := strings.Replace(f.Name, "classes/", "", 1)
			j.entries[classFileName] = f.Name
		}
	})

	class, exists := j.entries[name]

	if exists {
		f, err := reader.Open(class)

		if err != nil {
			return nil, err
		}

		return io.ReadAll(f)
	}

	return nil, nil
}

// Walk Walks a JMOD file and invokes `walk` for all classes found in the classlist
func (j *Jmod) Walk(walk WalkEntryFunc) error {
	r, err := getZipReader(j.FileName)
	if err != nil {
		return err
	}

	classSet := getClasslist(r)
	useClassSet := len(classSet) > 0

	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "classes") {
			continue
		}

		classFileName := strings.Replace(f.Name, "classes/", "", 1)
		j.entries[classFileName] = f.Name

		if useClassSet {
			_, ok := classSet[classFileName]
			if !ok {
				continue
			}
		} else {
			if !strings.HasSuffix(f.Name, ".class") {
				continue
			}
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		b, err := io.ReadAll(rc)
		if err != nil {
			return err
		}

		_ = walk(b, j.FileName+"+"+f.Name)

		_ = rc.Close()
	}

	return nil
}

// Returns lib/classlist from the JMOD file, returning an empty map if the classlist cannot be found or read
func getClasslist(reader *zip.Reader) map[string]struct{} {
	classSet := make(map[string]struct{})

	classlist, err := reader.Open("lib/classlist")
	if err != nil {
		_ = log.Log(err.Error(), log.CLASS)
		_ = log.Log("Unable to read lib/classlist from jmod file. Loading all classes in jmod file.", log.CLASS)
		return classSet
	}

	classlistContent, err := io.ReadAll(classlist)
	if err != nil {
		_ = log.Log(err.Error(), log.CLASS)
		_ = log.Log("Unable to read lib/classlist from jmod file. Loading all classes in jmod file.", log.CLASS)
		return classSet
	}

	classes := strings.Split(string(classlistContent), "\n")

	var empty struct{}

	for _, c := range classes {
		if strings.HasSuffix(c, "\r") || strings.HasSuffix(c, "\n") {
			c = strings.TrimRight(c, "\r\n")
		}
		classSet[c+".class"] = empty
	}

	log.Log("jmod manifest Classlist: "+string(classlistContent), log.TRACE_INST)

	return classSet
}

type JmodManager struct {
	jmodList map[string]*Jmod
	base     *Jmod
}

func InitJmodManager(javaHome string, baseName string) (*JmodManager, error) {
	baseDir := javaHome + string(os.PathSeparator) + "jmods"

	jmodList := make(map[string]*Jmod)
	var base *Jmod

	filepath.Walk(baseDir, func(path string, info os.FileInfo, e error) error {
		if !strings.HasSuffix(path, ".jmod") {
			return nil
		}

		jmodEntry := InitJmod(path)
		jmodList[filepath.Base(path)] = jmodEntry

		if filepath.Base(path) == baseName {
			base = jmodEntry
		}

		return nil
	})

	if base == nil {
		return nil, errors.New(fmt.Sprintf("Base JMOD with name %s not found in %s", baseName, baseDir))
	}

	return &JmodManager{
		jmodList: jmodList,
		base:     base,
	}, nil
}

func (manager *JmodManager) WalkBaseClasses(walk WalkEntryFunc) error {
	return manager.base.Walk(walk)
}

func (manager *JmodManager) LoadClassByName(name string) ([]byte, error) {
	for _, value := range manager.jmodList {
		res, err := value.LoadByName(name)

		if err != nil {
			return nil, err
		}

		if res != nil {
			return res, err
		}
	}
	return nil, nil
}
