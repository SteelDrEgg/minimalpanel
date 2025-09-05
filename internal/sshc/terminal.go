package sshc

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
)

func setupTerminal(session *ssh.Session, height int, width int) (stdin io.WriteCloser, stdout io.Reader, err error) {
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.ECHOCTL:       0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	err = session.RequestPty("xterm-256color", height, width, modes)
	if err != nil {
		return nil, nil, fmt.Errorf("request pseudo terminal failed: %v", err)
	}
	stdin, err = session.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to setup stdin for session: %v", err)
	}
	stdout, err = session.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to setup stdout for session: %v", err)
	}
	return
}
