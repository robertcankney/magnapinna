package pty

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

// not defined in syscall_unix - calculated using https://elixir.bootlin.com/linux/latest/source/include/uapi/asm-generic/ioctls.h#L81
// and https://elixir.bootlin.com/linux/latest/source/include/uapi/asm-generic/ioctl.h#L85
const TIOCGPTPEER = 21569

func open() (pty, tty *os.File, err error) {
	p, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	// In case of error after this point, make sure we close the ptmx fd.
	defer func() {
		if err != nil {
			_ = p.Close() // Best effort.
		}
	}()

	sname, err := ptsname(p)
	if err != nil {
		return nil, nil, err
	}

	// if err := grantpt(sname); err != nil {
	// 	return nil, nil, err
	// }

	if err := unlockpt(p); err != nil {
		return nil, nil, err
	}

	t, err := os.OpenFile(sname, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	return p, t, nil
}

func ptsname(f *os.File) (string, error) {
	var n _C_uint
	err := ioctl(f.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n)))
	if err != nil {
		return "", err
	}
	return "/dev/pts/" + strconv.Itoa(int(n)), nil
}

func unlockpt(f *os.File) error {
	var u _C_int
	// use TIOCSPTLCK with a pointer to zero to clear the lock
	return ioctl(f.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&u)))
}

func grantpt(pts string) error {
	return os.Chmod(pts, 0620)
}
