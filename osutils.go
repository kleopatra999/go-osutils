package osutils

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"code.google.com/p/go-uuid/uuid"
)

const (
	tempDirPrefix = "osutils"
)

var (
	ErrNotAbsolutePath     = errors.New("osutils: not absolute path")
	ErrNil                 = errors.New("osutils: nil")
	ErrEmpty               = errors.New("osutils: empty")
	ErrNotMultipleCommands = errors.New("osutils: not multiple commands")
	ErrFileDoesNotExist    = errors.New("osutils: file does not exist")
	ErrNotRegularFile      = errors.New("osutils: not regular file")
	ErrNotDir              = errors.New("osutils: not dir")
)

type Cmd struct {
	Args        []string
	AbsoluteDir string
	Env         []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
}

type PipeCmd struct {
	Args        []string
	AbsoluteDir string
	Env         []string
}

type PipeCmdList struct {
	PipeCmds []*PipeCmd
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
}

func Execute(cmd *Cmd) (func() error, error) {
	return execute(cmd)
}

func ExecutePiped(pipeCmdList *PipeCmdList) (func() error, error) {
	return executePiped(pipeCmdList)
}

func ListRegularFiles(absolutePath string) ([]string, error) {
	return listRegularFiles(absolutePath)
}

func Open(absolutePath string) (*os.File, error) {
	return open(absolutePath)
}

func Create(absolutePath string) (*os.File, error) {
	return create(absolutePath)
}

func IsRegularFileExists(absolutePath string) (bool, error) {
	return isRegularFileExists(absolutePath)
}

func IsDirExists(absolutePath string) (bool, error) {
	return isDirExists(absolutePath)
}

func IsFileExists(absolutePath string) (bool, error) {
	return isFileExists(absolutePath)
}

func Mkdir(absolutePath string, perm os.FileMode) error {
	return mkdir(absolutePath, perm)
}

func MkdirAll(absolutePath string, perm os.FileMode) error {
	return mkdirAll(absolutePath, perm)
}

func RemoveAll(absolutePath string) error {
	return removeAll(absolutePath)
}

func Rename(oldpath string, newpath string) error {
	return rename(oldpath, newpath)
}

func Getwd() (string, error) {
	return getwd()
}

func NewTempDir() (string, error) {
	return newTempDir()
}

func NewTempSubDir(absoluteBaseDirPath string) (string, error) {
	return newTempSubDir(absoluteBaseDirPath)
}

func CleanPath(absolutePath string) (string, error) {
	return cleanPath(absolutePath)
}

// ***** PRIVATE *****

func execute(cmd *Cmd) (func() error, error) {
	if cmd.Args == nil {
		return nil, ErrNil
	}
	if len(cmd.Args) == 0 {
		return nil, ErrEmpty
	}
	if cmd.AbsoluteDir != "" && !isAbsolutePath(cmd.AbsoluteDir) {
		return nil, ErrNotAbsolutePath
	}
	execCmd, err := execCmd(cmd)
	if err != nil {
		return nil, err
	}
	if err := execCmd.Start(); err != nil {
		return nil, err
	}
	return func() error { return execCmd.Wait() }, nil
}

func executePiped(pipeCmdList *PipeCmdList) (func() error, error) {
	if pipeCmdList.PipeCmds == nil {
		return nil, ErrNil
	}
	numCmds := len(pipeCmdList.PipeCmds)
	if numCmds == 0 {
		return nil, ErrEmpty
	}
	if numCmds <= 1 {
		return nil, ErrNotMultipleCommands
	}
	for _, pipeCmd := range pipeCmdList.PipeCmds {
		if pipeCmd.Args == nil {
			return nil, ErrNil
		}
		if len(pipeCmd.Args) == 0 {
			return nil, ErrEmpty
		}
		if pipeCmd.AbsoluteDir != "" && !isAbsolutePath(pipeCmd.AbsoluteDir) {
			return nil, ErrNotAbsolutePath
		}
	}
	execCmds := make([]*exec.Cmd, numCmds)
	for i, pipeCmd := range pipeCmdList.PipeCmds {
		execCmd, err := execPipeCmd(pipeCmd)
		if err != nil {
			return nil, err
		}
		execCmds[i] = execCmd
	}
	readers := make([]*io.PipeReader, numCmds-1)
	writers := make([]*io.PipeWriter, numCmds-1)
	reader, writer := io.Pipe()
	readers[0] = reader
	writers[0] = writer
	execCmds[0].Stdin = pipeCmdList.Stdin
	for i := 0; i < numCmds-1; i++ {
		execCmds[i].Stdout = writer
		execCmds[i].Stderr = pipeCmdList.Stderr
		execCmds[i+1].Stdin = reader
		if i != numCmds-2 {
			reader, writer = io.Pipe()
			readers[i+1] = reader
			writers[i+1] = writer
		}
	}
	execCmds[numCmds-1].Stdout = pipeCmdList.Stdout
	execCmds[numCmds-1].Stderr = pipeCmdList.Stderr
	for _, execCmd := range execCmds {
		if err := execCmd.Start(); err != nil {
			return nil, err
		}
	}
	return func() error {
		for i := 0; i < numCmds-1; i++ {
			if err := execCmds[i].Wait(); err != nil {
				return err
			}
			if i != 0 {
				if err := readers[i-1].Close(); err != nil {
					return err
				}
			}
			if err := writers[i].Close(); err != nil {
				return err
			}
		}
		if err := execCmds[numCmds-1].Wait(); err != nil {
			return err
		}
		if err := readers[numCmds-2].Close(); err != nil {
			return err
		}
		return nil
	}, nil
}

func listRegularFiles(absolutePath string) ([]string, error) {
	if !isAbsolutePath(absolutePath) {
		return nil, ErrNotAbsolutePath
	}
	files := make([]string, 0)
	err := filepath.Walk(
		absolutePath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				files = append(files, path)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func open(absolutePath string) (*os.File, error) {
	if !isAbsolutePath(absolutePath) {
		return nil, ErrNotAbsolutePath
	}
	exists, err := isFileExists(absolutePath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrFileDoesNotExist
	}
	return os.Open(absolutePath)
}

func create(absolutePath string) (*os.File, error) {
	if !isAbsolutePath(absolutePath) {
		return nil, ErrNotAbsolutePath
	}
	return os.Create(absolutePath)
}

func isRegularFileExists(absolutePath string) (bool, error) {
	if !isAbsolutePath(absolutePath) {
		return false, ErrNotAbsolutePath
	}
	fileInfo, err := stat(absolutePath)
	if err != nil {
		return false, err
	}
	if fileInfo == nil {
		return false, nil
	}
	if !fileInfo.Mode().IsRegular() {
		return false, ErrNotRegularFile
	}
	return true, nil
}

func isDirExists(absolutePath string) (bool, error) {
	if !isAbsolutePath(absolutePath) {
		return false, ErrNotAbsolutePath
	}
	fileInfo, err := stat(absolutePath)
	if err != nil {
		return false, err
	}
	if fileInfo == nil {
		return false, nil
	}
	if !fileInfo.Mode().IsDir() {
		return false, ErrNotDir
	}
	return true, nil
}

func isFileExists(absolutePath string) (bool, error) {
	if !isAbsolutePath(absolutePath) {
		return false, ErrNotAbsolutePath
	}
	fileInfo, err := stat(absolutePath)
	return fileInfo != nil, err
}

func mkdir(absolutePath string, perm os.FileMode) error {
	if !isAbsolutePath(absolutePath) {
		return ErrNotAbsolutePath
	}
	return os.Mkdir(absolutePath, perm)
}

func mkdirAll(absolutePath string, perm os.FileMode) error {
	if !isAbsolutePath(absolutePath) {
		return ErrNotAbsolutePath
	}
	return os.MkdirAll(absolutePath, perm)
}

func removeAll(absolutePath string) error {
	if !isAbsolutePath(absolutePath) {
		return ErrNotAbsolutePath
	}
	return os.RemoveAll(absolutePath)
}

func rename(oldpath string, newpath string) error {
	if !isAbsolutePath(oldpath) {
		return ErrNotAbsolutePath
	}
	if !isAbsolutePath(newpath) {
		return ErrNotAbsolutePath
	}
	return os.Rename(oldpath, newpath)
}

func getwd() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(wd)
	if err != nil {
		return "", err
	}
	return cleanPath(abs)
}

func newTempDir() (string, error) {
	tempDir, err := ioutil.TempDir("", tempDirPrefix)
	if err != nil {
		return "", err
	}
	return cleanPath(tempDir)
}

func newTempSubDir(absoluteBaseDirPath string) (string, error) {
	if !isAbsolutePath(absoluteBaseDirPath) {
		return "", ErrNotAbsolutePath
	}
	subDir := filepath.Join(absoluteBaseDirPath, uuid.NewUUID().String())
	if err := os.Mkdir(subDir, 0755); err != nil {
		return "", err
	}
	return cleanPath(subDir)
}

func cleanPath(absolutePath string) (string, error) {
	return filepath.EvalSymlinks(filepath.Clean(absolutePath))
}

func isAbsolutePath(path string) bool {
	return filepath.IsAbs(path)
}

func stat(absolutePath string) (os.FileInfo, error) {
	fileInfo, err := os.Stat(absolutePath)
	if err == nil {
		return fileInfo, nil
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return nil, err
}

func execCmd(cmd *Cmd) (*exec.Cmd, error) {
	var execCmd *exec.Cmd
	if len(cmd.Args) == 1 {
		execCmd = exec.Command(cmd.Args[0])
	} else {
		execCmd = exec.Command(cmd.Args[0], cmd.Args[1:]...)
	}
	execCmd.Dir = cmd.AbsoluteDir
	execCmd.Env = cmd.Env
	execCmd.Stdin = cmd.Stdin
	execCmd.Stdout = cmd.Stdout
	execCmd.Stderr = cmd.Stderr
	return execCmd, nil
}

func execPipeCmd(pipeCmd *PipeCmd) (*exec.Cmd, error) {
	var execCmd *exec.Cmd
	if len(pipeCmd.Args) == 1 {
		execCmd = exec.Command(pipeCmd.Args[0])
	} else {
		execCmd = exec.Command(pipeCmd.Args[0], pipeCmd.Args[1:]...)
	}
	execCmd.Dir = pipeCmd.AbsoluteDir
	execCmd.Env = pipeCmd.Env
	return execCmd, nil
}
