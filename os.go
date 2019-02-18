package main

import (
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

// OS functions that may be overridden for tests or dry-runs

var (
	osSymlink  = os.Symlink
	osReadlink = os.Readlink
	osLstat    = os.Lstat
	osMkdir    = os.Mkdir
	osMkdirAll = os.MkdirAll
	runCmd     = liveExec
)

func setDryRun() {
	osSymlink = drySymlink
	osMkdirAll = dryMkdirAll
	osMkdir = dryMkdir
	runCmd = dryExec
}

func drySymlink(oldname, newname string) error {
	drylogf("symlink %q -> %q", oldname, newname)
	return nil
}

func dryMkdirAll(path string, perm os.FileMode) error {
	drylogf("mkdir -p %q [%#o]", path, uint32(perm))
	return nil
}

func dryMkdir(path string, perm os.FileMode) error {
	drylogf("mkdir %q [%#o]", path, uint32(perm))
	return nil
}

func stringArgs(args []interface{}) []string {
	strs := make([]string, 0, len(args))
	for _, v := range args {
		if sub, ok := v.([]interface{}); ok {
			strs = append(strs, stringArgs(sub)...)
		} else {
			strs = append(strs, fmt.Sprint(v))
		}
	}
	return strs
}

func liveExec(name string, args ...interface{}) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("unable to locate %q: %v", err)
	}
	argv := stringArgs(args)
	vlogf("RUN %s %q", path, argv)
	cmd := exec.Command(name, argv...)
	return cmd.Run()
}

func dryExec(name string, args ...interface{}) error {
	path, err := exec.LookPath(name)
	if err != nil {
		path = name
	}
	argv := stringArgs(args)
	drylogf("RUN %s %q", path, argv)
	return nil
}

// setChroot chroots to the given path only if the path is non-empty.
func setChroot(path string) error {
	if path == "" {
		return nil
	}
	return unix.Chroot(path)
}

// symlink creates a symlink at name pointing to target.
// If the location given by name is already a symlink to the target, no action is taken.
// If force is true, it will unlink any file already located at the location given by name.
func symlink(target, name string, force bool) error {
	switch stat, err := osLstat(name); {
	case os.IsNotExist(err):
		goto symlink
	case err == nil && stat.Mode()&os.ModeSymlink != 0:
		vlogf("Symlink already exists: %s", name)
		// It's a symlink, so check the link
	case err != nil:
		return err
	}

	switch linked, err := os.Readlink(name); {
	case os.IsNotExist(err):
	case err == nil && linked == target:
		return nil
	case err == nil:
	case err == nil && force:
		vlogf("Removing file to create symlink: %s", name)
		if err = os.Remove(name); err != nil {
			return err
		}
	case err != nil:
		return err
	}
symlink:
	vlogf("Creating symlink %s -> %s", name, target)
	return osSymlink(target, name)
}
