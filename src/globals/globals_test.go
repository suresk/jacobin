/*
 * Jacobin VM - A Java virtual machine
 * Copyright (c) 2021 by the Jacobin authors. All rights reserved.
 * Licensed under Mozilla Public License 2.0 (MPL 2.0)
 */

package globals

import (
	"os"
	"testing"
)

func TestGlobalsInit(t *testing.T) {
	g := InitGlobals("testInit")

	if g.JacobinName != "testInit" {
		t.Errorf("Expecting globals init to set Jacobin name to 'testInit', got: %s", g.JacobinName)
	}

	if g.VmModel != "server" {
		t.Errorf("Expected globals init to set VmModel to 'server', got: %s", g.VmModel)
	}
}

// make sure the JAVA_HOME environment variable is extracted and the embedded slashes
// are reformatted correctly
func TestJavaHomeFormat(t *testing.T) {
	origJavaHome := os.Getenv("JAVA_HOME")
	_ = os.Setenv("JAVA_HOME", "foo/bar")
	InitJavaHome()
	ret := JavaHome()
	expectedPath := "foo" + string(os.PathSeparator) + "bar"
	if ret != expectedPath {
		t.Errorf("Expecting a JAVA_HOME of '%s', got: %s", expectedPath, ret)
	}
	_ = os.Setenv("JAVA_HOME", origJavaHome)
}

// verify that a trailing slash in JAVA_HOME is removed
func TestJavaHomeRemovalOfTrailingSlash(t *testing.T) {
	origJavaHome := os.Getenv("JAVA_HOME")
	_ = os.Setenv("JAVA_HOME", "foo/bar/")
	InitJavaHome()
	ret := JavaHome()
	expectedPath := "foo" + string(os.PathSeparator) + "bar"
	if ret != expectedPath {
		t.Errorf("Expecting a JAVA_HOME of '%s', got: %s", expectedPath, ret)
	}
	_ = os.Setenv("JAVA_HOME", origJavaHome)
}

// make sure the JACOBIN_HOME environment variable is extracted and reformatted correctly
// Per JACOBIN-184, the trailing slash is removed.
func TestJacobinHomeFormat(t *testing.T) {
	origJavaHome := os.Getenv("JACOBIN_HOME")
	_ = os.Setenv("JACOBIN_HOME", "foo/bar")
	InitJacobinHome()
	ret := JacobinHome()
	expectedPath := "foo" + string(os.PathSeparator) + "bar"
	if ret != expectedPath {
		t.Errorf("Expecting a JACOBIN_HOME of '%s', got: %s", expectedPath, ret)
	}
	_ = os.Setenv("JACOBIN_HOME", origJavaHome)
}

// verify that a trailing slash in JAVA_HOME is removed
func TestJacobinHomeRemovalOfTrailingSlash(t *testing.T) {
	origJavaHome := os.Getenv("JACOBIN_HOME")
	_ = os.Setenv("JACOBIN_HOME", "foo/bar/")
	InitJacobinHome()
	ret := JacobinHome()
	expectedPath := "foo" + string(os.PathSeparator) + "bar"
	if ret != expectedPath {
		t.Errorf("Expecting a JACOBIN_HOME of '%s', got: %s", expectedPath, ret)
	}
	_ = os.Setenv("JACOBIN_HOME", origJavaHome)
}

func TestVariousInitialDefaultValues(t *testing.T) {
	InitGlobals("testInit")
	gl := GetGlobalRef()
	if gl.StrictJDK != false ||
		gl.ExitNow != false ||
		!(gl.MaxJavaVersion >= 11) {
		t.Errorf("Some global variables intialized to unexpected values.")
	}
}
