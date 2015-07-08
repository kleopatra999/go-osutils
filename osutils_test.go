package osutils

import (
	"bytes"
	"io"
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

func (s *Suite) SetupSuite() {
}

func (s *Suite) SetupTest() {
	tempDir, err := NewTempDir()
	require.NoError(s.T(), err)
	s.tempDir = tempDir
}

func (s *Suite) TearDownTest() {
	require.NoError(s.T(), os.RemoveAll(s.tempDir))
}

func (s *Suite) TearDownSuite() {
}

func (s *Suite) TestPwd() {
	stdout, _ := s.execute([]string{"pwd", "-P"}, nil)
	require.Equal(s.T(), s.tempDir, stdout)
}

func (s *Suite) TestEnv() {
	writeFile, err := os.Create(filepath.Join(s.tempDir, "echo_foo.sh"))
	require.NoError(s.T(), err)
	fromFile, err := os.Open("_testdata/echo_foo.sh")
	require.NoError(s.T(), err)
	defer s.checkClose(fromFile)
	data, err := ioutil.ReadAll(fromFile)
	require.NoError(s.T(), err)
	_, err = writeFile.Write(data)
	require.NoError(s.T(), err)
	err = writeFile.Chmod(0777)
	require.NoError(s.T(), err)
	s.checkClose(writeFile)
	stdout, _ := s.execute([]string{"bash", filepath.Join(s.tempDir, "echo_foo.sh")}, []string{"FOO=foo"})
	require.Equal(s.T(), "foo", stdout)
}

func (s *Suite) TestPipe() {
	var input bytes.Buffer
	_, _ = input.WriteString("hello\n")
	_, _ = input.WriteString("hello\n")
	_, _ = input.WriteString("woot\n")
	_, _ = input.WriteString("hello\n")
	_, _ = input.WriteString("foo\n")
	_, _ = input.WriteString("woot\n")
	_, _ = input.WriteString("foo\n")
	_, _ = input.WriteString("woot\n")
	_, _ = input.WriteString("hello\n")
	_, _ = input.WriteString("foo\n")
	_, _ = input.WriteString("foo\n")
	_, _ = input.WriteString("foo\n")
	var output bytes.Buffer
	wait, err := ExecutePiped(
		&PipeCmdList{
			PipeCmds: []*PipeCmd{
				&PipeCmd{
					Args:        []string{"sort"},
					AbsoluteDir: s.tempDir,
				},
				&PipeCmd{
					Args:        []string{"uniq"},
					AbsoluteDir: s.tempDir,
				},
				&PipeCmd{
					Args:        []string{"wc", "-l"},
					AbsoluteDir: s.tempDir,
				},
			},
			Stdin:  &input,
			Stdout: &output,
		},
	)
	require.NoError(s.T(), err)
	require.NoError(s.T(), wait())
	require.True(s.T(), strings.Contains(output.String(), "3"))
}

func (s *Suite) TestListFileInfosShallow() {
	err := os.MkdirAll(filepath.Join(s.tempDir, "dirOne"), 0755)
	require.NoError(s.T(), err)
	err = os.MkdirAll(filepath.Join(s.tempDir, "dirTwo"), 0755)
	require.NoError(s.T(), err)
	err = os.MkdirAll(filepath.Join(s.tempDir, "dirOne/dirOneOne"), 0755)
	require.NoError(s.T(), err)
	err = os.MkdirAll(filepath.Join(s.tempDir, "dirTwo/dirTwoOne"), 0755)
	require.NoError(s.T(), err)
	file, err := os.Create(filepath.Join(s.tempDir, "one"))
	require.NoError(s.T(), err)
	s.checkClose(file)
	file, err = os.Create(filepath.Join(s.tempDir, "two"))
	require.NoError(s.T(), err)
	s.checkClose(file)
	file, err = os.Create(filepath.Join(s.tempDir, "dirOne/oneOne"))
	require.NoError(s.T(), err)
	s.checkClose(file)

	fileNameToDir := map[string]bool{
		"dirOne": true,
		"dirTwo": true,
		"one":    false,
		"two":    false,
	}
	dir, err := os.Open(s.tempDir)
	require.NoError(s.T(), err)
	fileInfos, err := dir.Readdir(-1)
	require.NoError(s.T(), err)
	s.checkClose(dir)
	require.Equal(s.T(), 4, len(fileInfos))
	for _, fileInfo := range fileInfos {
		dir, ok := fileNameToDir[fileInfo.Name()]
		require.True(s.T(), ok)
		require.Equal(s.T(), dir, fileInfo.IsDir())
		require.Equal(s.T(), !dir, fileInfo.Mode().IsRegular())
	}
}

func (s *Suite) execute(args []string, env []string) (stdout string, stderr string) {
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	wait, err := Execute(
		&Cmd{
			Args:        args,
			AbsoluteDir: s.tempDir,
			Env:         env,
			Stdout:      &stdoutBuffer,
			Stderr:      &stderrBuffer,
		},
	)
	require.NoError(s.T(), err)
	require.NoError(s.T(), wait())
	stdout = strings.TrimSpace(stdoutBuffer.String())
	stderr = strings.TrimSpace(stderrBuffer.String())
	return
}

func (s *Suite) checkFileExists(path string) {
	_, err := os.Stat(path)
	require.NoError(s.T(), err)
}

func (s *Suite) checkFileDoesNotExist(path string) {
	_, err := os.Stat(path)
	require.True(s.T(), os.IsNotExist(err))
}

func (s *Suite) checkClose(closer io.Closer) {
	err := closer.Close()
	require.NoError(s.T(), err)
}
