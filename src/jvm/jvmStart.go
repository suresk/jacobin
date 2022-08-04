/*
 * Jacobin VM - A Java virtual machine
 * Copyright (c) 2022 by the Jacobin authors. All rights reserved.
 * Licensed under Mozilla Public License 2.0 (MPL 2.0)
 */

package jvm

import (
	"context"
	"fmt"
	"jacobin/classloader"
	"jacobin/globals"
	"jacobin/log"
	"jacobin/management"
	"jacobin/shutdown"
	"os"
)

var Global globals.Globals

// JVMrun is where everything begins
// The call to shutdown.Exit() exits the program (after some clean-up and logging); the reason
// it is here returned is because in testing mode, the actual exit() call is side-stepped and
// instead an int is returned (because calling exit() during testing exits the testing run as well).
func JVMrun() int {
	management.StartMetricWriter()
	// if globals.JacobinName == "test", then we're in test mode and globals and log have been set
	// in the testing function. So, don't reset them here.
	if globals.GetGlobalRef().JacobinName != "test" {
		Global = globals.InitGlobals(os.Args[0])
		log.Init()
	} else {
		Global = *globals.GetGlobalRef()
	}

	_ = log.Log("running program: "+Global.JacobinName, log.FINE)

	// handle the command-line interface (cli) -- i.e., process the args
	LoadOptionsTable(Global)
	err := HandleCli(os.Args, &Global)
	if err != nil {
		return shutdown.Exit(shutdown.APP_EXCEPTION)
	}
	// some CLI options, like -version, show data and immediately exit. This tests for that.
	if Global.ExitNow == true {
		return shutdown.Exit(shutdown.OK)
	}

	// Init classloader and load base classes
	_ = classloader.Init()
	classloader.LoadBaseClasses(&Global)

	var mainClass string

	if Global.StartingJar != "" {
		manifestClass, err := classloader.GetMainClassFromJar(classloader.BootstrapCL, Global.StartingJar)

		if err != nil {
			_ = log.Log(err.Error(), log.INFO)
			return shutdown.Exit(shutdown.JVM_EXCEPTION)
		}

		if manifestClass == "" {
			_ = log.Log(fmt.Sprintf("no main manifest attribute, in %s", Global.StartingJar), log.INFO)
			return shutdown.Exit(shutdown.APP_EXCEPTION)
		}
		mainClass, err = classloader.LoadClassFromJar(classloader.BootstrapCL, manifestClass, Global.StartingJar)
		if err != nil { // the exceptions message will already have been shown to user
			return shutdown.Exit(shutdown.JVM_EXCEPTION)
		}
	} else if Global.StartingClass != "" {
		mainClass, err = classloader.LoadClassFromFile(classloader.BootstrapCL, Global.StartingClass)
		if err != nil { // the exceptions message will already have been shown to user
			return shutdown.Exit(shutdown.JVM_EXCEPTION)
		}
	} else {
		_ = log.Log("Error: No executable program specified. Exiting.", log.INFO)
		ShowUsage(os.Stdout)
		return shutdown.Exit(shutdown.APP_EXCEPTION)
	}

	server := management.StartServer()

	classloader.LoadReferencedClasses(mainClass)

	// begin execution
	_ = log.Log("Starting execution with: "+mainClass, log.INFO)
	if StartExec(mainClass, &Global) != nil {
		return shutdown.Exit(shutdown.APP_EXCEPTION)
	}

	var inp string

	fmt.Scanln(&inp)

	fmt.Println(inp)

	management.StopMetricWriter()
	server.Shutdown(context.TODO())

	return shutdown.Exit(shutdown.OK)
}
