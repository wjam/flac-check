//go:build !unix

package main

import "os"

var shutdownSignals = []os.Signal{os.Interrupt}
