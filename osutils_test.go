package osutils

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
	tempDir string
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (this *Suite) SetupSuite() {
}

func (this *Suite) SetupTest() {
	tempDir, err := NewTempDir()
	require.NoError(this.T(), err)
	this.tempDir = tempDir
}

func (this *Suite) TearDownTest() {
	require.NoError(this.T(), os.RemoveAll(this.tempDir))
}

func (this *Suite) TearDownSuite() {
}

func (this *Suite) TestPwd() {
	stdout, _ := this.execute([]string{"pwd", "-P"}, nil)
	require.Equal(this.T(), this.tempDir, stdout)
}

func (this *Suite) TestEnv() {
	writeFile, err := os.Create(filepath.Join(this.tempDir, "echo_foo.sh"))
	require.NoError(this.T(), err)
	fromFile, err := os.Open("_testdata/echo_foo.sh")
	require.NoError(this.T(), err)
	defer fromFile.Close()
	data, err := ioutil.ReadAll(fromFile)
	require.NoError(this.T(), err)
	_, err = writeFile.Write(data)
	require.NoError(this.T(), err)
	err = writeFile.Chmod(0777)
	require.NoError(this.T(), err)
	writeFile.Close()
	stdout, _ := this.execute([]string{"bash", filepath.Join(this.tempDir, "echo_foo.sh")}, []string{"FOO=foo"})
	require.Equal(this.T(), "foo", stdout)
}

func (this *Suite) TestPipe() {
	var input bytes.Buffer
	input.WriteString("hello\n")
	input.WriteString("hello\n")
	input.WriteString("woot\n")
	input.WriteString("hello\n")
	input.WriteString("foo\n")
	input.WriteString("woot\n")
	input.WriteString("foo\n")
	input.WriteString("woot\n")
	input.WriteString("hello\n")
	input.WriteString("foo\n")
	input.WriteString("foo\n")
	input.WriteString("foo\n")
	var output bytes.Buffer
	wait, err := ExecutePiped(
		&PipeCmdList{
			PipeCmds: []*PipeCmd{
				&PipeCmd{
					Args: []string{"sort"},
				},
				&PipeCmd{
					Args: []string{"uniq"},
				},
				&PipeCmd{
					Args: []string{"wc", "-l"},
				},
			},
			AbsoluteDir: this.tempDir,
			Stdin:       &input,
			Stdout:      &output,
		},
	)
	require.NoError(this.T(), err)
	require.NoError(this.T(), wait())
	require.True(this.T(), strings.Contains(output.String(), "3"))
}

func (this *Suite) TestListFileInfosShallow() {
	err := os.MkdirAll(filepath.Join(this.tempDir, "dirOne"), 0755)
	require.NoError(this.T(), err)
	err = os.MkdirAll(filepath.Join(this.tempDir, "dirTwo"), 0755)
	require.NoError(this.T(), err)
	err = os.MkdirAll(filepath.Join(this.tempDir, "dirOne/dirOneOne"), 0755)
	require.NoError(this.T(), err)
	err = os.MkdirAll(filepath.Join(this.tempDir, "dirTwo/dirTwoOne"), 0755)
	require.NoError(this.T(), err)
	file, err := os.Create(filepath.Join(this.tempDir, "one"))
	require.NoError(this.T(), err)
	file.Close()
	file, err = os.Create(filepath.Join(this.tempDir, "two"))
	require.NoError(this.T(), err)
	file.Close()
	file, err = os.Create(filepath.Join(this.tempDir, "dirOne/oneOne"))
	require.NoError(this.T(), err)
	file.Close()

	fileNameToDir := map[string]bool{
		"dirOne": true,
		"dirTwo": true,
		"one":    false,
		"two":    false,
	}
	dir, err := os.Open(this.tempDir)
	require.NoError(this.T(), err)
	fileInfos, err := dir.Readdir(-1)
	require.NoError(this.T(), err)
	dir.Close()
	require.Equal(this.T(), 4, len(fileInfos))
	for _, fileInfo := range fileInfos {
		dir, ok := fileNameToDir[fileInfo.Name()]
		require.True(this.T(), ok)
		require.Equal(this.T(), dir, fileInfo.IsDir())
		require.Equal(this.T(), !dir, fileInfo.Mode().IsRegular())
	}
}

func (this *Suite) execute(args []string, env []string) (stdout string, stderr string) {
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	wait, err := Execute(
		&Cmd{
			Args:        args,
			AbsoluteDir: this.tempDir,
			Env:         env,
			Stdout:      &stdoutBuffer,
			Stderr:      &stderrBuffer,
		},
	)
	require.NoError(this.T(), err)
	require.NoError(this.T(), wait())
	stdout = strings.TrimSpace(stdoutBuffer.String())
	stderr = strings.TrimSpace(stderrBuffer.String())
	return
}

func (this *Suite) checkFileExists(path string) {
	_, err := os.Stat(path)
	require.NoError(this.T(), err)
}

func (this *Suite) checkFileDoesNotExist(path string) {
	_, err := os.Stat(path)
	require.True(this.T(), os.IsNotExist(err))
}
