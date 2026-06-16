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

	tea "github.com/charmbracelet/bubbletea"
)

type SelectedServer struct {
	Name string
	IP   string
	Port string
	User string
	Pass string
}

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

func openconnect(p *tea.Program, stopChan <-chan struct{}, doneChan chan<- struct{}, sv SelectedServer, flags []FlagRow) {
	defer close(doneChan)
	var err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	var TARGET_SERVER string = fmt.Sprintf("%s:%s", sv.IP, sv.Port)
	var OC_USER string = sv.User
	var OC_PASS string = sv.Pass
	var SELECTED_FLAGS []string
	for _, flag := range flags {
		if flag.Selected == "1" {
			if strings.HasSuffix(flag.Flag, "=") {
				SELECTED_FLAGS = append(SELECTED_FLAGS, "--"+flag.Flag+flag.Value)
			} else {
				SELECTED_FLAGS = append(SELECTED_FLAGS, "--"+flag.Flag)
			}
		}
	}
	oc_args := []string{TARGET_SERVER, "-u", OC_USER}
	oc_args = append(oc_args, SELECTED_FLAGS...)
	cmd := exec.Command("openconnect", oc_args...)
	stdout, _ := cmd.StdoutPipe()
	stdin, _ := cmd.StdinPipe()
	cmd.Stderr = cmd.Stdout
	configureSysProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		p.Send(vpnStatusMsg(fmt.Sprintf("Error starting process: %v", err)))
		// log.Fatal("Error starting process: %v", err)
		return
	}

	p.Send(vpnStatusMsg("2"))
	go func() {
		<-stopChan
		// fmt.Println("\n[Go]: Stop signal received. Attempting graceful shutdown...")
		err := interruptProcess(cmd)
		if err != nil {
			// fmt.Printf("Error during graceful shutdown: %v\n", err)
			_ = cmd.Process.Kill()
		}
	}()

	buf := make([]byte, 1024)
	var accumulatedOutput string

	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			p.Send(vpnLogMsg(chunk))

			accumulatedOutput += chunk

			if strings.Contains(accumulatedOutput, "Server certificate verify failed: signer not found") {
				// fmt.Println("\n[Go]: Prompt detected! Sending 'yes'...")
				io.WriteString(stdin, "yes\n")
				accumulatedOutput = ""
			} else if strings.Contains(accumulatedOutput, "Password:") {
				// fmt.Println("\n[Go]: Prompt detected! Sending password...")
				var pass_prompt = OC_PASS + "\n"
				io.WriteString(stdin, pass_prompt)
				accumulatedOutput = ""
			} else if strings.Contains(accumulatedOutput, "Legacy IP route configuration done.") {
				p.Send(vpnStatusMsg("1"))
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
	p.Send(vpnStatusMsg("0"))
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
