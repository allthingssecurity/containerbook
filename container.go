package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
        "time"
        "net"
)


func main() {

switch os.Args[1] {
	case "parent":
		parent()
	case "child":
		child()
	default:
		panic("help")
	}
}

func enableCgroup() {
	cgroups := "/root/mygrp"
	pids := filepath.Join(cgroups, "child")
	//os.Mkdir(filepath.Join(pids, "smj"), 0755)
	must(ioutil.WriteFile(filepath.Join(pids, "memory.max"), []byte("2M"), 0700))
	// Removes the new cgroup in place after the container exits
	//must(ioutil.WriteFile(filepath.Join(pids, "notify_on_release"), []byte("1"), 0700))
	must(ioutil.WriteFile(filepath.Join(pids, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}

func waitForNetwork() error {
	maxWait := time.Second * 3
	checkInterval := time.Second
	timeStarted := time.Now()

	for {
		interfaces, err := net.Interfaces()
		if err != nil {
			return err
		}

		// pretty basic check ...
		// > 1 as a lo device will already exist
		if len(interfaces) > 1 {
			return nil
		}

		if time.Since(timeStarted) > maxWait {
			return fmt.Errorf("Timeout after %s waiting for network", maxWait)
		}

		time.Sleep(checkInterval)
	}
}


func mountProc(newroot string) error {
	source := "proc"
	target := filepath.Join(newroot, "/proc")
	fstype := "proc"
	flags := 0
	data := ""

	os.MkdirAll(target, 0755)
	if err := syscall.Mount(
		source,
		target,
		fstype,
		uintptr(flags),
		data,
	); err != nil {
		return err
	}

	return nil
}

func pivotRoot(newroot string) error {
	putold := filepath.Join(newroot, "/.pivot_root")

	// bind mount newroot to itself - this is a slight hack needed to satisfy the
	// pivot_root requirement that newroot and putold must not be on the same
	// filesystem as the current root
	if err := syscall.Mount(newroot, newroot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return err
	}

	// create putold directory
	if err := os.MkdirAll(putold, 0700); err != nil {
		return err
	}

	// call pivot_root
	if err := syscall.PivotRoot(newroot, putold); err != nil {
		return err
	}

	// ensure current working directory is set to new root
	if err := os.Chdir("/"); err != nil {
		return err
	}

	// umount putold, which now lives at /.pivot_root
	putold = "/.pivot_root"
	if err := syscall.Unmount(putold, syscall.MNT_DETACH); err != nil {
		return err
	}

	// remove putold
	if err := os.RemoveAll(putold); err != nil {
		return err
	}

	return nil
}


func parent() {

	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = []string{"name=shashank"}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWUSER,
	UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}

	must(cmd.Start())


//pid := fmt.Sprintf("%d", cmd.Process.Pid)
//netsetgoCmd := exec.Command("/usr/local/bin/netsetgo", "-pid", pid)
//if err := netsetgoCmd.Run(); err != nil {
  //      fmt.Printf("Error running netsetgo - %s\n", err)
    //    os.Exit(1)
//}

if err := cmd.Wait(); err != nil {
        fmt.Printf("Error waiting for reexec.Command - %s\n", err)
        os.Exit(1)
}

}

func  child () {
enableCgroup()
cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

must(mountProc("/root/book_prep/rootfs"))
//must(syscall.Mount("proc", "proc", "proc", 0, ""))

must(syscall.Sethostname([]byte("myhost")))


	if err := pivotRoot("/root/book_prep/rootfs"); err != nil {
		fmt.Printf("Error running pivot_root - %s\n", err)
		os.Exit(1)
	}

//must(syscall.Mount("proc", "proc", "proc", 0, ""))
//if err := waitForNetwork(); err != nil {
//		fmt.Printf("Error waiting for network - %s\n", err)
//		os.Exit(1)
//	}



if err := cmd.Run(); err != nil {
	fmt.Printf("Error starting the reexec.Command - %s\n", err)
	os.Exit(1)
}






//must(cmd.Run())
}
func must(err error) {
	if err != nil {
		fmt.Printf("Error  - %s\n", err)
	}
}
