/*
 * Jacobin VM - A Java virtual machine
 * Copyright (c) 2021-2 by the Jacobin authors. All rights reserved.
 * Licensed under Mozilla Public License 2.0 (MPL 2.0)
 */
package classloader

import (
	"jacobin/globals"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJmodFile(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Error("Unable to get cwd")
		return
	}

	jmodFileName := filepath.Join(pwd, "..", "..", "testdata", "jmod", "jacobin.jmod")

	jmod := InitJmod(jmodFileName)

	filesFound := make(map[string]any, 10)

	var empty struct{}

	jmod.Walk(func(bytes []byte, filename string) error {
		fname := strings.Split(filename, "+")[1]
		filesFound[fname] = empty
		return nil
	})

	if _, ok := filesFound["classes/org/jacobin/test/Hello.class"]; !ok {
		t.Error("Expected org.jacobin.test.Hello, but it wasn't there.")
	}

	if _, ok := filesFound["classes/module-info.class"]; ok {
		t.Error("Didn't expect module-info, but it was there.")
	}
}

func TestJmodFileNoClasslist(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Error("Unable to get cwd")
		return
	}

	jmodFileName := filepath.Join(pwd, "..", "..", "testdata", "jmod", "jacobinfull.jmod")

	jmod := InitJmod(jmodFileName)

	filesFound := make(map[string]any, 10)

	var empty struct{}

	err = jmod.Walk(func(bytes []byte, filename string) error {
		fname := strings.Split(filename, "+")[1]
		filesFound[fname] = empty
		return nil
	})

	if err != nil {
		t.Error(err)
	}

	if _, ok := filesFound["classes/org/jacobin/test/Hello.class"]; !ok {
		t.Error("Expected org.jacobin.test.Hello, but it wasn't there.")
	}

	if _, ok := filesFound["classes/module-info.class"]; !ok {
		t.Error("Expected module-info, but it wasn't there.")
	}
}

func TestNotJmodFile(t *testing.T) {
	// informs shutdown.Exit() that we're in test mode so not to exit on exception
	g := globals.GetGlobalRef()
	globals.InitGlobals("test")
	g.JacobinName = "test"

	pwd, err := os.Getwd()
	if err != nil {
		t.Error("Unable to get cwd")
		return
	}

	jmodFileName := filepath.Join(pwd, "..", "..", "testdata", "jmod", "README.md")

	jmod := InitJmod(jmodFileName)

	err = jmod.Walk(func(bytes []byte, filename string) error {
		return nil
	})

	if err == nil {
		t.Error("Should have gotten error that README.md isn't a JMOD file, but didn't.")
	}
}
