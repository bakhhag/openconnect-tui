package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"golang.org/x/sys/windows"
)

func amIAdmin() bool {
	var sid *windows.SID

	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

func runAsAdmin() {
	verb := "runas"
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(args)

	var showCmd int32 = 1

	err = windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	if err != nil {
		log.Fatalf("Failed to elevate process: %v", err)
	}
	os.Exit(0)
}

func openconnect(stopChan <-chan struct{}) {
	var err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	OC_USER := os.Getenv("OC_USER")
	OC_PASS := os.Getenv("OC_PASS")
	cmd := exec.Command("openconnect", "fr.securetunnels.net:22", "-u", OC_USER)
	stdout, _ := cmd.StdoutPipe()
	stdin, _ := cmd.StdinPipe()
	cmd.Stderr = cmd.Stdout
	configureSysProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start: %v", err)
	}

	go func() {
		<-stopChan
		fmt.Println("\n[Go]: Stop signal received. Attempting graceful shutdown...")
		err := interruptProcess(cmd)
		if err != nil {
			fmt.Printf("Error during graceful shutdown: %v\n", err)
			_ = cmd.Process.Kill()
		}
	}()

	buf := make([]byte, 1024)
	var accumulatedOutput string

	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			fmt.Print(chunk)

			accumulatedOutput += chunk

			if strings.Contains(accumulatedOutput, "Server certificate verify failed: signer not found") {
				fmt.Println("\n[Go]: Prompt detected! Sending 'yes'...")
				io.WriteString(stdin, "yes\n")
				accumulatedOutput = ""
			} else if strings.Contains(accumulatedOutput, "Password:") {
				fmt.Println("\n[Go]: Prompt detected! Sending password...")
				var pass_prompt = OC_PASS + "\n"
				io.WriteString(stdin, pass_prompt)
				accumulatedOutput = ""
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("Read error: %v\n", err)
			break
		}
	}
	cmd.Wait()
}

func configureSysProcAttr(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		}
	}
}

func interruptProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return fmt.Errorf("process not started")
	}

	if runtime.GOOS == "windows" {
		dll, err := syscall.LoadDLL("kernel32.dll")
		if err != nil {
			return err
		}
		defer dll.Release()

		proc, err := dll.FindProc("GenerateConsoleCtrlEvent")
		if err != nil {
			return err
		}

		r, _, errResult := proc.Call(syscall.CTRL_BREAK_EVENT, uintptr(cmd.Process.Pid))
		if r == 0 {
			return errResult
		}
		return nil
	}

	return cmd.Process.Signal(os.Interrupt)
}
