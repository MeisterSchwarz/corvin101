//go:build windows

package system

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"golang.org/x/sys/windows"
)

func TestSingleInstanceAcrossProcesses(t *testing.T) {
	name := fmt.Sprintf(`Local\corvin101_instance_test_%d`, os.Getpid())
	handle, err := acquireInstanceMutex(name)
	if err != nil || handle == 0 {
		t.Fatalf("first instance: %v / %v", handle, err)
	}
	defer func() {
		if handle != 0 {
			windows.CloseHandle(handle)
		}
	}()
	child := func(expected string) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestInstanceChild$")
		cmd.Env = append(os.Environ(), "CORVIN_TEST_MUTEX="+name, "CORVIN_TEST_EXPECT="+expected)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("child (%s): %v\n%s", expected, err, output)
		}
	}
	child("blocked")
	// A refused start must close its own duplicate handle, or this stays locked.
	if err := windows.CloseHandle(handle); err != nil {
		t.Fatal(err)
	}
	handle = 0
	child("acquired")
}

func TestInstanceChild(t *testing.T) {
	name := os.Getenv("CORVIN_TEST_MUTEX")
	if name == "" {
		t.Skip("subprocess helper")
	}
	handle, err := acquireInstanceMutex(name)
	if handle != 0 {
		defer windows.CloseHandle(handle)
	}
	if err != nil {
		t.Fatal(err)
	}
	acquired := handle != 0
	if acquired != (os.Getenv("CORVIN_TEST_EXPECT") == "acquired") {
		t.Fatalf("unexpected acquired=%v", acquired)
	}
}
