package adb

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type Process struct {
	User string
	Pid  int
	Name string
}

// ListProcesses return list of Process
func (c *Device) ListProcesses() (ps []Process, err error) {
	reader, err := c.OpenCommand("ps")
	if err != nil {
		return
	}
	defer reader.Close()
	var fieldNames []string
	bufrd := bufio.NewReader(reader)
	for {
		line, _, err := bufrd.ReadLine()
		fields := strings.Fields(strings.TrimSpace(string(line)))
		if len(fields) == 0 {
			break
		}
		if err == io.EOF {
			break
		}
		if fieldNames == nil {
			fieldNames = fields
			continue
		}
		var process Process
		/* example output of command "ps"
		USER     PID   PPID  VSIZE  RSS     WCHAN    PC         NAME
		root      1     0     684    540   ffffffff 00000000 S /init
		root      2     0     0      0     ffffffff 00000000 S kthreadd
		*/
		if len(fields) != len(fieldNames)+1 {
			continue
		}
		for index, name := range fieldNames {
			value := fields[index]
			switch strings.ToUpper(name) {
			case "PID":
				process.Pid, _ = strconv.Atoi(value)
			case "NAME":
				process.Name = fields[len(fields)-1]
			case "USER":
				process.User = value
			}
		}
		if process.Pid == 0 {
			continue
		}
		ps = append(ps, process)
	}
	return
}

type PackageInfo struct {
	Name    string
	Path    string
	Version struct {
		Code int
		Name string
	}
}

var (
	rePkgPath = regexp.MustCompile(`codePath=([^\s]+)`)
	reVerCode = regexp.MustCompile(`versionCode=(\d+)`)
	reVerName = regexp.MustCompile(`versionName=([^\s]+)`)
)

// StatPackage returns PackageInfo
// If package not found, err will be ErrPackageNotExist
func (c *Device) StatPackage(packageName string) (pi PackageInfo, err error) {
	pi.Name = packageName
	out, err := c.RunCommand("dumpsys", "package", packageName)
	if err != nil {
		return
	}

	matches := rePkgPath.FindStringSubmatch(out)
	if len(matches) == 0 {
		err = ErrPackageNotExist
		return
	}
	pi.Path = matches[1]

	matches = reVerCode.FindStringSubmatch(out)
	if len(matches) == 0 {
		err = ErrPackageNotExist
		return
	}
	pi.Version.Code, _ = strconv.Atoi(matches[1])

	matches = reVerName.FindStringSubmatch(out)
	if len(matches) == 0 {
		err = ErrPackageNotExist
		return
	}
	pi.Version.Name = matches[1]
	return
}

// Properties extract info from $ adb shell getprop
func (c *Device) Properties() (props map[string]string, err error) {
	propOutput, err := c.RunCommand("getprop")
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`\[(.*?)\]:\s*\[(.*?)\]`)
	matches := re.FindAllStringSubmatch(propOutput, -1)
	props = make(map[string]string)
	for _, m := range matches {
		var key = m[1]
		var val = m[2]
		props[key] = val
	}
	return
}

/*
RunCommandWithExitCode use a little tricky to get exit code

The tricky is append "; echo :$?" to the command,
and parse out the exit code from output
*/
func (c *Device) RunCommandWithExitCode(cmd string, args ...string) (string, int, error) {
	exArgs := append(args, ";", "echo", ":$?")
	outStr, err := c.RunCommand(cmd, exArgs...)
	if err != nil {
		return outStr, 0, err
	}
	idx := strings.LastIndexByte(outStr, ':')
	if idx == -1 {
		return outStr, 0, fmt.Errorf("adb shell aborted, can not parse exit code")
	}
	exitCode, _ := strconv.Atoi(strings.TrimSpace(outStr[idx+1:]))
	if exitCode != 0 {
		err = ShellExitError{strings.Join(args, " "), exitCode}
	}
	outStr = strings.Replace(outStr[0:idx], "\r\n", "\n", -1) // put somewhere else
	return outStr, exitCode, err
}
