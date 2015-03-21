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
	Args []string
	Env  []string
}

type PipeCmdList struct {
	PipeCmds    []*PipeCmd
	AbsoluteDir string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
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

func IsFileExists(absolutePath string) (bool, error) {
	return isFileExists(absolutePath)
}

func NewTempDir() (string, error) {
	return newTempDir()
}

func NewSubDir(absoluteDirPath string) (string, error) {
	return newSubDir(absoluteDirPath)
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
	if pipeCmdList.AbsoluteDir != "" && !isAbsolutePath(pipeCmdList.AbsoluteDir) {
		return nil, ErrNotAbsolutePath
	}
	for _, pipeCmd := range pipeCmdList.PipeCmds {
		if pipeCmd.Args == nil {
			return nil, ErrNil
		}
		if len(pipeCmd.Args) == 0 {
			return nil, ErrEmpty
		}
	}
	execCmds := make([]*exec.Cmd, numCmds)
	for i, pipeCmd := range pipeCmdList.PipeCmds {
		execCmd, err := execPipeCmd(pipeCmdList.AbsoluteDir, pipeCmd)
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

func isFileExists(absolutePath string) (bool, error) {
	if !isAbsolutePath(absolutePath) {
		return false, ErrNotAbsolutePath
	}
	_, err := os.Stat(absolutePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func newTempDir() (string, error) {
	tempDir, err := ioutil.TempDir("", tempDirPrefix)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(filepath.Clean(tempDir))
}

func newSubDir(absoluteDirPath string) (string, error) {
	if !isAbsolutePath(absoluteDirPath) {
		return "", ErrNotAbsolutePath
	}
	subDir := filepath.Join(absoluteDirPath, uuid.NewUUID().String())
	subDir, err := filepath.EvalSymlinks(filepath.Clean(subDir))
	if err != nil {
		return "", err
	}
	if err := os.Mkdir(subDir, 0755); err != nil {
		return "", err
	}
	return subDir, nil
}

func isAbsolutePath(path string) bool {
	return filepath.IsAbs(path)
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

func execPipeCmd(absoluteDir string, pipeCmd *PipeCmd) (*exec.Cmd, error) {
	var execCmd *exec.Cmd
	if len(pipeCmd.Args) == 1 {
		execCmd = exec.Command(pipeCmd.Args[0])
	} else {
		execCmd = exec.Command(pipeCmd.Args[0], pipeCmd.Args[1:]...)
	}
	execCmd.Dir = absoluteDir
	execCmd.Env = pipeCmd.Env
	return execCmd, nil
}
