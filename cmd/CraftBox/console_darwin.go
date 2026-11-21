package main

import "os"

func redirectStderr(f *os.File) {
	os.Stderr = f
}
