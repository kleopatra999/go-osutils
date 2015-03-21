[![API Documentation](http://img.shields.io/badge/api-Godoc-blue.svg?style=flat-square)](https://godoc.org/github.com/peter-edge/go-osutils)
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](https://github.com/peter-edge/go-osutils/blob/master/LICENSE)

OS utilities for Go.

## Installation
```bash
go get -u github.com/peter-edge/go-osutils
```

## Import
```go
import (
    "github.com/peter-edge/go-osutils"
)
```

## Usage

```go
var (
	ErrNotAbsolutePath     = errors.New("osutils: not absolute path")
	ErrNil                 = errors.New("osutils: nil")
	ErrEmpty               = errors.New("osutils: empty")
	ErrNotMultipleCommands = errors.New("osutils: not multiple commands")
)
```

#### func  Execute

```go
func Execute(cmd *Cmd) (func() error, error)
```

#### func  ExecutePiped

```go
func ExecutePiped(pipeCmdList *PipeCmdList) (func() error, error)
```

#### func  IsFileExists

```go
func IsFileExists(absolutePath string) (bool, error)
```

#### func  ListRegularFiles

```go
func ListRegularFiles(absolutePath string) ([]string, error)
```

#### func  NewSubDir

```go
func NewSubDir(absoluteDirPath string) (string, error)
```

#### func  NewTempDir

```go
func NewTempDir() (string, error)
```

#### type Cmd

```go
type Cmd struct {
	Args        []string
	AbsoluteDir string
	Env         []string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
}
```


#### type PipeCmd

```go
type PipeCmd struct {
	Args []string
	Env  []string
}
```


#### type PipeCmdList

```go
type PipeCmdList struct {
	PipeCmds    []*PipeCmd
	AbsoluteDir string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
}
```
