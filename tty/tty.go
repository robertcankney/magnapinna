package tty

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"golang.org/x/term"
)

type Terminal struct {
	Stdin    io.Writer
	stdout   chan []byte
	stdin    chan []byte
	shell    *exec.Cmd
	shellout poller
	shellin  poller
	errors   chan error
}

type poller struct {
	out    *os.File
	in     *os.File
	buffer []byte
}

type readwriter struct {
	*bytes.Buffer
}

var ErrIn = errors.New("failed to write to stdin")

const bufferSize = 1024 * 256

// func (r *readwriter) Write(b []byte) (n int, err error) {
// 	return r.Buffer.Write(b)
// }

// func (r *readwriter) Read(b []byte) (n int, err error) {
// 	return r.Buffer.Read(b)
// }

func NewTerminal(ctx context.Context, command string, args ...string) (*Terminal, error) {
	cmd := exec.Command(command, args...)
	winsz := &pty.Winsize{
		Rows: 64,
		Cols: 128,
		X:    720,
		Y:    360,
	}
	attrs := &syscall.SysProcAttr{
		Setsid: true,
	}
	ptmx, err := pty.StartWithAttrs(cmd, winsz, attrs)
	if err != nil {
		return nil, fmt.Errorf("could not open ptmx: %w", err)
	}
	name, err := ptsname(ptmx)
	if err != nil {
		return nil, fmt.Errorf("could not get ptsname: %w", err)
	}
	pts, err := os.OpenFile(name, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %w", name, err)
	}

	_, err = term.MakeRaw(int(ptmx.Fd()))
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %w", name, err)
	}
	return &Terminal{
		shell: cmd,
		shellout: poller{
			out:    ptmx,
			buffer: make([]byte, bufferSize),
		},
		shellin: poller{
			in:     pts,
			buffer: make([]byte, bufferSize),
		},
		stdout: make(chan []byte),
		stdin:  make(chan []byte),
	}, nil
}

func (t *Terminal) Errors() <-chan error {
	return t.errors
}

func (t *Terminal) Out() <-chan []byte {
	return t.stdout
}

func (t *Terminal) Start() error {
	go t.wait()
	go t.outpoll()
	go t.inpoll()
	return nil
}

func (t *Terminal) wait() {
	err := t.shell.Wait()
	t.errors <- err
}

// TODO better error handling and context handling for both polls
func (t *Terminal) outpoll() {
	total := 0
	tmp := make([]byte, len(t.shellout.buffer))
	for {
		n := 0
		var err error
		for err == nil {
			n, err = t.shellout.out.Read(t.shellout.buffer)
			total += n
			copy(tmp, t.shellout.buffer)
			t.stdout <- t.shellout.buffer[:n]
			t.shellout.buffer = t.shellout.buffer[:]
			tmp = tmp[:]
		}
		if err == io.EOF {
			continue
		}
		t.errors <- err
	}
}

func (t *Terminal) inpoll() {
	total := 0
	for {
		n := 0
		var err error
		for err == nil {
			b := <-t.stdin
			for _, c := range b {
				err = tiocsti(t.shellin.in, c)
				if err != nil {
					fmt.Println(err.Error())
				}
				n++
			}
			total += n
		}
		if err == io.EOF {
			continue
		}
		t.errors <- err
	}
}

func (t *Terminal) RunCommand(cmd []byte) {
	t.stdin <- cmd
}

func tiocsti(f *os.File, b byte) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCSTI, uintptr(unsafe.Pointer(&b)))
	if err != 0 {
		return err
	}
	return nil
}

func ptsname(f *os.File) (string, error) {
	var n int
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n)))
	if err != 0 {
		return "", err
	}
	return "/dev/pts/" + strconv.Itoa(int(n)), nil
}
