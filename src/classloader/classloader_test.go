/*
 * Jacobin VM - A Java virtual machine
 * Copyright (c) 2021 by Andrew Binstock. All rights reserved.
 * Licensed under Mozilla Public License 2.0 (MPL 2.0)
 */

package classloader

import (
	"bytes"
	"errors"
	"io"
	"jacobin/globals"
	"jacobin/log"
	"os"
	"strings"
	"testing"
)

// Most of the functionality in classloader package is tested in other files, such as
// * cpParser_test.go (constant pool parser)
// * formatCheck_test.go (the format checking)
// * parser_test.go (the class parsing)
// etc.
// This files tests remaining routines.

func TestInitOfClassloaders(t *testing.T) {
	_ = Init()

	// check that the classloader hierarchy is set up correctly
	if BootstrapCL.Parent != "" {
		t.Errorf("Expecting parent of Boostrap classloader to be empty, got: %s",
			BootstrapCL.Parent)
	}

	if ExtensionCL.Parent != "bootstrap" {
		t.Errorf("Expecting parent of Extension classloader to be Boostrap, got: %s",
			ExtensionCL.Parent)
	}

	if AppCL.Parent != "extension" {
		t.Errorf("Expecting parent of Application classloader to be Extension, got: %s",
			AppCL.Parent)
	}

	// check that the classloaders have empty tables ready
	if len(BootstrapCL.Classes) != 0 {
		t.Errorf("Expected size of boostrap CL's table to be 0, got: %d", len(BootstrapCL.Classes))
	}

	if len(ExtensionCL.Classes) != 0 {
		t.Errorf("Expected size of extension CL's table to be 0, got: %d", len(ExtensionCL.Classes))
	}

	if len(AppCL.Classes) != 0 {
		t.Errorf("Expected size of application CL's table to be 0, got: %d", len(AppCL.Classes))
	}
}

func TestWalkWithError(t *testing.T) {
	e := errors.New("test error")
	err := walk("", nil, e)
	if err != e {
		t.Errorf("Expected an error = to 'test error', got %s",
			err.Error())
	}
}

// when walk() encounters an invalid file, it is simply skipped
// with no error generated as it's not clear that entry in jmod
// will be necessary. If it is, when it's invoked, it will be loaded
// then and any errors in finding the file will be returned then.
func TestJmodWalkWithInvalidDirAndFile(t *testing.T) {
	err := os.Mkdir("subdir", 0755)
	defer os.RemoveAll("subdir")
	_ = os.WriteFile("subdir/file1", []byte(""), 0644)

	dirEntry, err := os.ReadDir("subdir")
	err = walk("gherkin", dirEntry[0], nil)
	if err != nil {
		t.Errorf("Expected no error on invalid file in walk(), but got %s",
			err.Error())
	}
}

func TestLoadBaseClassesInvalid(t *testing.T) {
	globals.InitGlobals("test")
	log.Init()
	_ = log.SetLogLevel(log.CLASS)

	// redirect stderr & stdout to capture results from stderr
	normalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	normalStdout := os.Stdout
	_, wout, _ := os.Pipe()
	os.Stdout = wout

	g := globals.GetGlobalRef()
	g.JavaHome = "gherkin"

	errC := make(chan string)

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		errC <- buf.String()
	}()

	// Loading the base classes from an invalid JAVA_HOME should
	// result in an error message being logged.
	_ = Init()
	LoadBaseClasses(globals.GetGlobalRef())

	_ = w.Close()
	os.Stderr = normalStderr

	_ = wout.Close()
	os.Stdout = normalStdout

	errMsg := <-errC

	if !strings.Contains(errMsg, "JAVA_HOME (gherkin) does not exist") {
		t.Errorf("Expecting err msg 'JAVA_HOME (gherkin) does not exist', but got %s",
			errMsg)
	}
}

func TestLoadClassFromFileInvalidName(t *testing.T) {
	// redirect stderr & stdout to capture results from stderr
	normalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	normalStdout := os.Stdout
	_, wout, _ := os.Pipe()
	os.Stdout = wout

	name, err := LoadClassFromFile(Classloader{}, "noSuchFile")

	if name != "" {
		t.Errorf("Expected empty filename due to error, got: %s", err.Error())
	}
	if err == nil {
		t.Errorf("Expected an error message for invalid file name, but got none")
	}

	_ = w.Close()
	_, _ = io.ReadAll(r)
	os.Stderr = normalStderr

	_ = wout.Close()
	os.Stdout = normalStdout
}

// remove leading [L and delete trailing;, eliminate all other entries with [prefix
func TestNormalizingClassReference(t *testing.T) {
	s := normalizeClassReference("[Ljava/test/java.String;")
	if s != "java/test/java.String" {
		t.Error("Unexpected normalized class reference: " + s)
	}

	s = normalizeClassReference("[B")
	if s != "" {
		t.Error("Unexpected normalized class reference: " + s)
	}

	s = normalizeClassReference("java/lang/Object")
	if s != "java/lang/Object" {
		t.Error("Unexpected normalized class reference: " + s)
	}
}

func TestConvertToPostableClassStringRefs(t *testing.T) {
	// Testing the changes made as a result of JACOBIN-103
	globals.InitGlobals("test")
	log.Init()
	_ = log.SetLogLevel(log.CLASS)

	// set up a class with a constant pool containing the one
	// StringConst we want to make sure is converted to a UTF8
	klass := ParsedClass{}
	klass.cpIndex = append(klass.cpIndex, cpEntry{})
	klass.cpIndex = append(klass.cpIndex, cpEntry{StringConst, 0})
	klass.cpIndex = append(klass.cpIndex, cpEntry{UTF8, 0})

	klass.stringRefs = append(klass.stringRefs, stringConstantEntry{index: 0})
	klass.utf8Refs = append(klass.utf8Refs, utf8Entry{content: "Hello string"})

	klass.cpCount = 3

	postableClass := convertToPostableClass(&klass)
	if len(postableClass.CP.Utf8Refs) != 1 {
		t.Errorf("Expecting a UTF8 slice of length 1, got %d",
			len(postableClass.CP.Utf8Refs))
	}

	// cpIndex[1] is a StringConst above, should now be a UTF8
	utf8 := postableClass.CP.CpIndex[1]
	if utf8.Type != UTF8 {
		t.Errorf("Expecting StringConst entry to have become UTF8 entry,"+
			"but instead is of type: %d", utf8.Type)
	}
}

func TestGetInvalidJar(t *testing.T) {
	globals.InitGlobals("test")
	log.Init()
	_ = log.SetLogLevel(log.CLASS)

	// redirect stderr & stdout to capture results from stderr
	normalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	normalStdout := os.Stdout
	_, wout, _ := os.Pipe()
	os.Stdout = wout

	_, err := getJarFile(BootstrapCL, "")
	if err == nil {
		t.Errorf("expected err msg for fetching an invalid JAR, but got none")
	}

	// restore stderr and stdout to what they were before
	_ = w.Close()
	out, _ := io.ReadAll(r)
	os.Stderr = normalStderr

	msg := string(out[:])

	_ = wout.Close()
	os.Stdout = normalStdout

	if !strings.Contains(msg, "Invalid or corrupt jarfile") {
		t.Error("Got unexpected error msg: " + msg)
	}
}

func TestGetClassFromInvalidJar(t *testing.T) {
	globals.InitGlobals("test")
	log.Init()
	_ = log.SetLogLevel(log.CLASS)

	// redirect stderr & stdout to capture results from stderr
	normalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	normalStdout := os.Stdout
	_, wout, _ := os.Pipe()
	os.Stdout = wout

	_, err := LoadClassFromJar(BootstrapCL, "pickle", "gherkin")
	if err == nil {
		t.Errorf("expected err msg for loading invalid class from invalid JAR, but got none")
	}

	// restore stderr and stdout to what they were before
	_ = w.Close()
	out, _ := io.ReadAll(r)
	os.Stderr = normalStderr

	msg := string(out[:])

	_ = wout.Close()
	os.Stdout = normalStdout

	if !strings.Contains(msg, "Invalid or corrupt jarfile") {
		t.Error("Got unexpected error msg: " + msg)
	}
}

func TestMainClassFromInvalidJar(t *testing.T) {
	globals.InitGlobals("test")
	log.Init()
	_ = log.SetLogLevel(log.CLASS)

	// redirect stderr & stdout to capture results from stderr
	normalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	normalStdout := os.Stdout
	_, wout, _ := os.Pipe()
	os.Stdout = wout

	_, err := GetMainClassFromJar(BootstrapCL, "gherkin")
	if err == nil {
		t.Errorf("expected err msg for loading main class from invalid JAR, but got none")
	}

	// restore stderr and stdout to what they were before
	_ = w.Close()
	out, _ := io.ReadAll(r)
	os.Stderr = normalStderr

	msg := string(out[:])

	_ = wout.Close()
	os.Stdout = normalStdout

	if !strings.Contains(msg, "Invalid or corrupt jarfile") {
		t.Error("Got unexpected error msg: " + msg)
	}
}

func TestInsertionIntoMethodArea(t *testing.T) {
	globals.InitGlobals("test")
	log.Init()
	_ = log.SetLogLevel(log.CLASS)

	// redirect stderr & stdout to capture results from stderr
	normalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	normalStdout := os.Stdout
	_, wout, _ := os.Pipe()
	os.Stdout = wout

	Classes = make(map[string]Klass)

	k := Klass{}
	k.Status = 'F'
	k.Loader = "application"
	clData := ClData{}
	clData.Name = "WillyWonkaClass"
	k.Data = &clData
	_ = insert("WillyWonkaClass", k)

	// restore stderr and stdout to what they were before
	_ = w.Close()
	out, _ := io.ReadAll(r)
	os.Stderr = normalStderr

	msg := string(out[:])

	_ = wout.Close()
	os.Stdout = normalStdout

	if !strings.Contains(msg, "WillyWonkaClass") || !strings.Contains(msg, "application") {
		t.Error("Got unexpected logging message for insertion of Klass into method area: " + msg)
	}

	if len(Classes) != 1 {
		t.Errorf("Expecting method area to have a size of 1, got: %d", len(Classes))
	}
}

func TestInvalidMagicNumberViaParseAndPostFunction(t *testing.T) {

	normalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	normalStdout := os.Stdout
	_, wout, _ := os.Pipe()
	os.Stdout = wout

	globals.InitGlobals("test")
	log.Init()
	_ = log.SetLogLevel(log.WARNING)
	err := Init()

	testBytes := []byte{
		0xCB, 0xFE, 0xBA, 0xBE,
	}

	_, err = ParseAndPostClass(BootstrapCL, "Hello2", testBytes)
	if err == nil {
		t.Error("Expected an error, but got none.")
	}

	_ = w.Close()
	msg, _ := io.ReadAll(r)
	os.Stderr = normalStderr

	_ = wout.Close()
	os.Stdout = normalStdout

	if !strings.Contains(string(msg), "invalid magic number") {
		t.Errorf("Expected error message to contain in part 'invalid magic number', got: %s", string(msg))
	}
}

var Hello2Bytes = []byte{
	0xCA, 0xFE, 0xBA, 0xBE, 0x00, 0x00, 0x00, 0x37, 0x00, 0x2B, 0x07, 0x00, 0x02, 0x01, 0x00, 0x06,
	0x48, 0x65, 0x6C, 0x6C, 0x6F, 0x32, 0x07, 0x00, 0x04, 0x01, 0x00, 0x10, 0x6A, 0x61, 0x76, 0x61,
	0x2F, 0x6C, 0x61, 0x6E, 0x67, 0x2F, 0x4F, 0x62, 0x6A, 0x65, 0x63, 0x74, 0x01, 0x00, 0x06, 0x3C,
	0x69, 0x6E, 0x69, 0x74, 0x3E, 0x01, 0x00, 0x03, 0x28, 0x29, 0x56, 0x01, 0x00, 0x04, 0x43, 0x6F,
	0x64, 0x65, 0x0A, 0x00, 0x03, 0x00, 0x09, 0x0C, 0x00, 0x05, 0x00, 0x06, 0x01, 0x00, 0x0F, 0x4C,
	0x69, 0x6E, 0x65, 0x4E, 0x75, 0x6D, 0x62, 0x65, 0x72, 0x54, 0x61, 0x62, 0x6C, 0x65, 0x01, 0x00,
	0x12, 0x4C, 0x6F, 0x63, 0x61, 0x6C, 0x56, 0x61, 0x72, 0x69, 0x61, 0x62, 0x6C, 0x65, 0x54, 0x61,
	0x62, 0x6C, 0x65, 0x01, 0x00, 0x04, 0x74, 0x68, 0x69, 0x73, 0x01, 0x00, 0x08, 0x4C, 0x48, 0x65,
	0x6C, 0x6C, 0x6F, 0x32, 0x3B, 0x01, 0x00, 0x04, 0x6D, 0x61, 0x69, 0x6E, 0x01, 0x00, 0x16, 0x28,
	0x5B, 0x4C, 0x6A, 0x61, 0x76, 0x61, 0x2F, 0x6C, 0x61, 0x6E, 0x67, 0x2F, 0x53, 0x74, 0x72, 0x69,
	0x6E, 0x67, 0x3B, 0x29, 0x56, 0x0A, 0x00, 0x01, 0x00, 0x11, 0x0C, 0x00, 0x12, 0x00, 0x13, 0x01,
	0x00, 0x06, 0x61, 0x64, 0x64, 0x54, 0x77, 0x6F, 0x01, 0x00, 0x05, 0x28, 0x49, 0x49, 0x29, 0x49,
	0x09, 0x00, 0x15, 0x00, 0x17, 0x07, 0x00, 0x16, 0x01, 0x00, 0x10, 0x6A, 0x61, 0x76, 0x61, 0x2F,
	0x6C, 0x61, 0x6E, 0x67, 0x2F, 0x53, 0x79, 0x73, 0x74, 0x65, 0x6D, 0x0C, 0x00, 0x18, 0x00, 0x19,
	0x01, 0x00, 0x03, 0x6F, 0x75, 0x74, 0x01, 0x00, 0x15, 0x4C, 0x6A, 0x61, 0x76, 0x61, 0x2F, 0x69,
	0x6F, 0x2F, 0x50, 0x72, 0x69, 0x6E, 0x74, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6D, 0x3B, 0x0A, 0x00,
	0x1B, 0x00, 0x1D, 0x07, 0x00, 0x1C, 0x01, 0x00, 0x13, 0x6A, 0x61, 0x76, 0x61, 0x2F, 0x69, 0x6F,
	0x2F, 0x50, 0x72, 0x69, 0x6E, 0x74, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6D, 0x0C, 0x00, 0x1E, 0x00,
	0x1F, 0x01, 0x00, 0x07, 0x70, 0x72, 0x69, 0x6E, 0x74, 0x6C, 0x6E, 0x01, 0x00, 0x04, 0x28, 0x49,
	0x29, 0x56, 0x01, 0x00, 0x04, 0x61, 0x72, 0x67, 0x73, 0x01, 0x00, 0x13, 0x5B, 0x4C, 0x6A, 0x61,
	0x76, 0x61, 0x2F, 0x6C, 0x61, 0x6E, 0x67, 0x2F, 0x53, 0x74, 0x72, 0x69, 0x6E, 0x67, 0x3B, 0x01,
	0x00, 0x01, 0x78, 0x01, 0x00, 0x01, 0x49, 0x01, 0x00, 0x01, 0x69, 0x01, 0x00, 0x0D, 0x53, 0x74,
	0x61, 0x63, 0x6B, 0x4D, 0x61, 0x70, 0x54, 0x61, 0x62, 0x6C, 0x65, 0x07, 0x00, 0x21, 0x01, 0x00,
	0x01, 0x6A, 0x01, 0x00, 0x01, 0x6B, 0x01, 0x00, 0x0A, 0x53, 0x6F, 0x75, 0x72, 0x63, 0x65, 0x46,
	0x69, 0x6C, 0x65, 0x01, 0x00, 0x0B, 0x48, 0x65, 0x6C, 0x6C, 0x6F, 0x32, 0x2E, 0x6A, 0x61, 0x76,
	0x61, 0x00, 0x20, 0x00, 0x01, 0x00, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
	0x05, 0x00, 0x06, 0x00, 0x01, 0x00, 0x07, 0x00, 0x00, 0x00, 0x2F, 0x00, 0x01, 0x00, 0x01, 0x00,
	0x00, 0x00, 0x05, 0x2A, 0xB7, 0x00, 0x08, 0xB1, 0x00, 0x00, 0x00, 0x02, 0x00, 0x0A, 0x00, 0x00,
	0x00, 0x06, 0x00, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x0B, 0x00, 0x00, 0x00, 0x0C, 0x00, 0x01,
	0x00, 0x00, 0x00, 0x05, 0x00, 0x0C, 0x00, 0x0D, 0x00, 0x00, 0x00, 0x09, 0x00, 0x0E, 0x00, 0x0F,
	0x00, 0x01, 0x00, 0x07, 0x00, 0x00, 0x00, 0x81, 0x00, 0x03, 0x00, 0x03, 0x00, 0x00, 0x00, 0x1E,
	0x03, 0x3D, 0xA7, 0x00, 0x15, 0x1C, 0x1C, 0x04, 0x64, 0xB8, 0x00, 0x10, 0x3C, 0xB2, 0x00, 0x14,
	0x1B, 0xB6, 0x00, 0x1A, 0x84, 0x02, 0x01, 0x1C, 0x10, 0x0A, 0xA1, 0xFF, 0xEB, 0xB1, 0x00, 0x00,
	0x00, 0x03, 0x00, 0x0A, 0x00, 0x00, 0x00, 0x16, 0x00, 0x05, 0x00, 0x00, 0x00, 0x06, 0x00, 0x05,
	0x00, 0x07, 0x00, 0x0D, 0x00, 0x08, 0x00, 0x14, 0x00, 0x06, 0x00, 0x1D, 0x00, 0x0A, 0x00, 0x0B,
	0x00, 0x00, 0x00, 0x20, 0x00, 0x03, 0x00, 0x00, 0x00, 0x1E, 0x00, 0x20, 0x00, 0x21, 0x00, 0x00,
	0x00, 0x0D, 0x00, 0x0A, 0x00, 0x22, 0x00, 0x23, 0x00, 0x01, 0x00, 0x02, 0x00, 0x1B, 0x00, 0x24,
	0x00, 0x23, 0x00, 0x02, 0x00, 0x25, 0x00, 0x00, 0x00, 0x0F, 0x00, 0x02, 0xFF, 0x00, 0x05, 0x00,
	0x03, 0x07, 0x00, 0x26, 0x00, 0x01, 0x00, 0x00, 0x11, 0x00, 0x08, 0x00, 0x12, 0x00, 0x13, 0x00,
	0x01, 0x00, 0x07, 0x00, 0x00, 0x00, 0x38, 0x00, 0x02, 0x00, 0x02, 0x00, 0x00, 0x00, 0x04, 0x1A,
	0x1B, 0x60, 0xAC, 0x00, 0x00, 0x00, 0x02, 0x00, 0x0A, 0x00, 0x00, 0x00, 0x06, 0x00, 0x01, 0x00,
	0x00, 0x00, 0x0D, 0x00, 0x0B, 0x00, 0x00, 0x00, 0x16, 0x00, 0x02, 0x00, 0x00, 0x00, 0x04, 0x00,
	0x27, 0x00, 0x23, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00, 0x28, 0x00, 0x23, 0x00, 0x01, 0x00,
	0x01, 0x00, 0x29, 0x00, 0x00, 0x00, 0x02, 0x00, 0x2A,
}

func TestLoadFullyParsedClass(t *testing.T) {
	globals.InitGlobals("test")
	log.Init()
	_ = log.SetLogLevel(log.CLASS)

	fullyParsedClass, err := parse(Hello2Bytes)
	if err != nil {
		t.Errorf("Got unexpected error from parse of Hello2.class: %s", err.Error())
	}
	classToPost := convertToPostableClass(&fullyParsedClass)
	if len(classToPost.Methods) < 1 {
		t.Errorf("Invalid number of methods in Hello2.class: %d", len(classToPost.Methods))
	}
}
