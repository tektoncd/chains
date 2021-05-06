package rpm

import (
	"os"
	"time"
)

// File flags make up some attributes of files depending on how they were
// specified in the rpmspec
const (
	FileFlagNone      = 0
	FileFlagConfig    = (1 << 0)  // %%config
	FileFlagDoc       = (1 << 1)  // %%doc
	FileFlagIcon      = (1 << 2)  // %%donotuse
	FileFlagMissingOk = (1 << 3)  // %%config(missingok)
	FileFlagNoReplace = (1 << 4)  // %%config(noreplace)
	FileFlagGhost     = (1 << 6)  // %%ghost
	FileFlagLicense   = (1 << 7)  // %%license
	FileFlagReadme    = (1 << 8)  // %%readme
	FileFlagPubkey    = (1 << 11) // %%pubkey
	FileFlagArtifact  = (1 << 12) // %%artifact
)

// A FileInfo describes a file in a RPM package and is returned by
// packagefile.Files.
//
// FileInfo implements the os.FileInfo interface.
type FileInfo struct {
	name     string
	size     int64
	mode     os.FileMode
	modTime  time.Time
	isDir    bool
	flags    int64
	owner    string
	group    string
	digest   string
	linkname string
}

// compile-time check that rpm.FileInfo implements os.FileInfo interface
var _ os.FileInfo = new(FileInfo)

func (f *FileInfo) String() string {
	return f.Name()
}

// Name is the full path of a file in a RPM package
func (f *FileInfo) Name() string {
	return f.name
}

// Size is the size in bytes of a file in a RPM package
func (f *FileInfo) Size() int64 {
	return f.size
}

// Mode is the file mode in bits of a file in a RPM package
func (f *FileInfo) Mode() os.FileMode {
	return f.mode
}

// ModTime is the modification time of a file in a RPM package
func (f *FileInfo) ModTime() time.Time {
	return f.modTime
}

// IsDir returns true if a file is a directory in a RPM package
func (f *FileInfo) IsDir() bool {
	return f.mode.IsDir()
}

func (f *FileInfo) Flags() int64 {
	return f.flags
}

// Owner is the name of the owner of a file in a RPM package
func (f *FileInfo) Owner() string {
	return f.owner
}

// Group is the name of the owner group of a file in a RPM package
func (f *FileInfo) Group() string {
	return f.group
}

// Digest is the md5sum of a file in a RPM package
func (f *FileInfo) Digest() string {
	return f.digest
}

// Linkname is the link target of a link file in a RPM package
func (f *FileInfo) Linkname() string {
	return f.linkname
}

// Sys is required to implement os.FileInfo and always returns nil
func (f *FileInfo) Sys() interface{} {
	// underlying data source is a bunch of rpm header indices
	return nil
}
