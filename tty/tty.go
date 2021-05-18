package tty

import (
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

// TODO: add filtering for repeated bash prompts

// Terminal is the type responsible for creating and managing ptys, as well
// as managing and buffering IO for the process running in the pty.
type Terminal struct {
	stdout   chan []byte
	stdin    chan []byte
	ctx      context.Context
	shell    *exec.Cmd
	shellout poller
	shellin  poller
	errors   chan error
}

// poller wraps an open file descriptor, as well as a buffer for IO with the pty.
type poller struct {
	handle *os.File
	buffer []byte
	data   int64
}

// ErrPtsWrite is returned when the write to stdin of the pts' process fails.
var ErrPtsWrite = errors.New("failed to write to pts for stdin")

const bufferSize = 1024 * 32

// NewTerminal creates a new pair of ptys, and runs command using the create ptys for IO.
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
		return nil, fmt.Errorf("could not open pty %s: %w", name, err)
	}
	_, err = term.MakeRaw(int(ptmx.Fd()))
	if err != nil {
		return nil, fmt.Errorf("could not set pty %s to raw mode: %w", name, err)
	}

	return &Terminal{
		shell: cmd,
		ctx:   ctx,
		shellout: poller{
			handle: ptmx,
			buffer: make([]byte, bufferSize),
		},
		shellin: poller{
			handle: pts,
			buffer: make([]byte, bufferSize),
		},
		stdout: make(chan []byte),
		stdin:  make(chan []byte, 1),
	}, nil
}

// Errors returns a read-only channel to receive errors from the Terminal. Note that
// errors from the actual process using the ptys will be returned on the Out channel -
// these will be errors from the Terminal itself.
func (t *Terminal) Errors() <-chan error {
	return t.errors
}

// Out returns a read-only channel to receive output from the Terminal.
func (t *Terminal) Out() <-chan []byte {
	return t.stdout
}

// Start starts the process passed in to NewTerminal, and begins buffering IO for the process.
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

// TODO combine both polls to avoid hot read/EOF loop below
// outpoll and inpoll read and write data from the appropriate channels to the
// stdin and stdout of the pty.
func (t *Terminal) outpoll() {
	for {
		select {
		case <-t.ctx.Done():
			err := t.shellout.handle.Close()
			if err != nil {
				t.errors <- err
			}
		default:
			n := 0
			var err error
			for err == nil {
				n, err = t.shellout.handle.Read(t.shellout.buffer)
				t.shellout.data += int64(n)
				t.stdout <- t.shellout.buffer[:n]
			}
			if err == io.EOF {
				continue
			}
			t.errors <- err
		}
	}
}

func (t *Terminal) inpoll() {
	for {
		n := 0
		var err error
		for err == nil {
			b := <-t.stdin
			for _, c := range b {
				err = tiocsti(t.shellin.handle, c)
				if err != nil {
					t.errors <- fmt.Errorf("%w: %s", ErrPtsWrite, err.Error())
				}
				n++
			}
			t.shellout.data += int64(n)
		}
		if err == io.EOF {
			continue
		}
		t.errors <- err
	}
}

// RunCommand hands the byte slice off to the process running in Terminal. Note that this is an unbuffered channel,
// but that a return from this method does not mean that the command has been executed, just that it has been written to the pty's
// input queue.
func (t *Terminal) RunCommand(cmd []byte) {
	t.stdin <- cmd
}

// wraps the TIOCSTI syscall for writing to the input queue of a pts
// https://man7.org/linux/man-pages/man2/ioctl_tty.2.html
func tiocsti(f *os.File, b byte) error {
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCSTI, uintptr(unsafe.Pointer(&b)))
	if err != 0 {
		return err
	}
	return nil
}

// wraps the TIOCGTPN syscall to get the /dev/pts for an FD returned from opening /dev/ptmx
// https://man7.org/linux/man-pages/man3/ptsname.3.html
func ptsname(f *os.File) (string, error) {
	var n int
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n)))
	if err != 0 {
		return "", err
	}
	return "/dev/pts/" + strconv.Itoa(int(n)), nil
}
