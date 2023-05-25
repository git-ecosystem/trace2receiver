//go:build darwin
// +build darwin

package trace2receiver

import (
	"net"
	"os/user"
	"strconv"

	"golang.org/x/sys/unix"
)

// Get the username of the process on the other end of
// the unix domain socket connection.  (It is not sufficient
// to just call `user.Current()` because the telemetry
// service will probably be running as root or some other
// pseudo-user.)
func getPeerUsername(conn *net.UnixConn) (string, error) {
	raw, err := conn.SyscallConn()
	if err != nil {
		return "", err
	}

	var cred *unix.Xucred
	var crederr error

	err = raw.Control(
		func(fd uintptr) {
			cred, crederr = unix.GetsockoptXucred(int(fd),
				unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
			err = crederr
		})

	if err != nil {
		return "", err
	}

	uidString := strconv.FormatUint(uint64(cred.Uid), 10)

	u, err := user.LookupId(uidString)
	if err != nil {
		return "", err
	}

	return u.Username, nil
}
