//go:build freebsd
// +build freebsd

package filesystem

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/canonical/lxd/shared/logger"
)

// Filesystem magic numbers.
const (
	FilesystemSuperMagicZfs = 0x2fc12fc1
)

// StatVFS retrieves Virtual File System (VFS) info about a path.
func StatVFS(path string) (*unix.Statfs_t, error) {
	var st unix.Statfs_t

	err := unix.Statfs(path, &st)
	if err != nil {
		return nil, err
	}

	return &st, nil
}

// Detect returns the filesystem on which the passed-in path sits.
func Detect(path string) (string, error) {
	fs, err := StatVFS(path)
	if err != nil {
		return "", err
	}

	return FSTypeToName(int32(fs.Type))
}

// FSTypeToName returns the name of the given fs type.
// The fsType is from the Type field of unix.Statfs_t. We use int32 so that this function behaves the same on both
// 32bit and 64bit platforms by requiring any 64bit FS types to be overflowed before being passed in. They will
// then be compared with equally overflowed FS type constant values.
func FSTypeToName(fsType int32) (string, error) {
	// This function is needed to allow FS type constants that overflow an int32 to be overflowed without a
	// compile error on 32bit platforms. This allows us to use any 64bit constants from the unix package on
	// both 64bit and 32bit platforms without having to define the constant in its rolled over form on 32bit.
	_ = func(fsType int64) int32 {
		return int32(fsType)
	}

	switch fsType {
	case FilesystemSuperMagicZfs:
		return "zfs", nil
	}

	logger.Debugf("Unknown backing filesystem type: 0x%x", fsType)
	return fmt.Sprintf("0x%x", fsType), nil
}

func hasMountEntry(name string) int {
	// In case someone uses symlinks we need to look for the actual
	// mountpoint.
	actualPath, err := filepath.EvalSymlinks(name)
	if err != nil {
		return -1
	}

	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return -1
	}

	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Fields(line)
		if len(tokens) < 5 {
			return -1
		}

		cleanPath := filepath.Clean(tokens[4])
		if cleanPath == actualPath {
			return 1
		}
	}

	return 0
}

// IsMountPoint returns true if path is a mount point.
func IsMountPoint(path string) bool {
	// If we find a mount entry, it is obviously a mount point.
	ret := hasMountEntry(path)
	if ret == 1 {
		return true
	}

	// Get the stat details.
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}

	rootStat, err := os.Lstat(path + "/..")
	if err != nil {
		return false
	}

	// If the directory has the same device as parent, then it's not a mountpoint.
	if stat.Sys().(*syscall.Stat_t).Dev == rootStat.Sys().(*syscall.Stat_t).Dev {
		return false
	}

	// Btrfs annoyingly uses a different Dev id for different subvolumes on the same mount.
	// So for btrfs, we require a matching mount entry in mountinfo.
	fs, err := Detect(path)
	if err == nil && fs == "btrfs" {
		return false
	}

	return true
}

// SyncFS will force a filesystem sync for the filesystem backing the provided path.
func SyncFS(path string) error {
	// Get us a file descriptor.
	fsFile, err := os.Open(path)
	if err != nil {
		return err
	}

	defer func() { _ = fsFile.Close() }()

	// Call SyncFS.
	return nil
}

// PathNameEncode encodes a path string to be used as part of a file name.
// The encoding scheme replaces "-" with "--" and then "/" with "-".
func PathNameEncode(text string) string {
	return strings.Replace(strings.Replace(text, "-", "--", -1), "/", "-", -1)
}

// PathNameDecode decodes a string containing an encoded path back to its original form.
// The decoding scheme converts "-" back to "/" and "--" back to "-".
func PathNameDecode(text string) string {
	// This converts "--" to the null character "\0" first, to allow remaining "-" chars to be
	// converted back to "/" before making a final pass to convert "\0" back to original "-".
	return strings.Replace(strings.Replace(strings.Replace(text, "--", "\000", -1), "-", "/", -1), "\000", "-", -1)
}

// mountOption represents an individual mount option.
type mountOption struct {
	capture bool
	flag    uintptr
}

// mountFlagTypes represents a list of possible mount flags.
var mountFlagTypes = map[string]mountOption{
	"defaults": {true, 0},
}

// ResolveMountOptions resolves the provided mount options.
func ResolveMountOptions(options []string) (uintptr, string) {
	mountFlags := uintptr(0)
	var mountOptions []string

	for i := range options {
		do, ok := mountFlagTypes[options[i]]
		if !ok {
			mountOptions = append(mountOptions, options[i])
			continue
		}

		if do.capture {
			mountFlags |= do.flag
		} else {
			mountFlags &= ^do.flag
		}
	}

	return mountFlags, strings.Join(mountOptions, ",")
}

// GetMountinfo tracks down the mount entry for the path and returns all MountInfo fields.
func GetMountinfo(path string) ([]string, error) {
	return nil, fmt.Errorf("No mountinfo entry found")
}
