package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type SelectedServer struct {
	Name string
	IP   string
	Port string
	User string
	Pass string
}

func openconnect(p *tea.Program, stopChan <-chan struct{}, doneChan chan<- struct{}, sv Profile, flags []FlagRow) {
	defer close(doneChan)

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
			p.Send(vpnLogMsg(accumulatedOutput))

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
