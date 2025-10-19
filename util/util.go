package util

import (
	"fmt"
	"os"
	"bufio"
	"strings"
	"strconv"
	"errors"
	"path/filepath"
	"os/exec"
)

func Error(format string, v ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", v...)
}
func Warn(format string, v ...any) {
	fmt.Fprintf(os.Stderr, "WARN: "+format+"\n", v...)
}
func Fatal(format string, v ...any) {
	Error(format, v...)
	os.Exit(1)
}

func AskString(msg string, defalt string) string {
	if len(defalt) == 0 {
		defalt = "-"
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s [%s]: ", msg, defalt)
	answer, _ := reader.ReadString('\n')

	answer = strings.TrimSpace(answer)
	if len(answer) == 0 {
		answer = defalt
	}

	if answer == "-" {
		return ""
	} else {
		return answer
	}
}
func AskInt(msg string, defalt int) int {
	for {
		def := "-"
		if defalt > 0 {
			def = strconv.Itoa(defalt)
		}
		str := AskString(msg, def)
		if len(str) == 0 {
			return defalt
		}

		num, err := strconv.Atoi(str)
		if err != nil {
			Error("Ivalid number, try again")
			continue
		}

		return num
	}
}
func AskBool(msg string) bool {
	return AskString(msg, "yes/no") == "yes"
}

// Replace all "___" with "/"
func FixTitle(title string) string {
	title = strings.TrimSpace(title)
	title = strings.Replace(title, "___", "/", -1)
	return title
}

// Get song number and title
func SongNumberAndTitle(basename string) (int, string) {
	number := -1
	title := ""

	chunks := strings.SplitN(basename, " ", 2)
	if len(chunks) == 1 {
		number = -1
		title = chunks[0]
	} else if len(chunks) == 2 {
		n, err := strconv.Atoi(chunks[0])
		if err != nil {
			number = -1
		} else {
			number = n
		}

		title = chunks[1]
	} else {
		number = -1
		title = basename
	}

	title = strings.TrimSpace(title)
	dotIdx := strings.LastIndexByte(title, '.')
	if dotIdx > 0 {
		title = title[:dotIdx]
	}
	title = FixTitle(title)

	return number, title
}

// Get parenting dir of the path
func ParentDir(path string) (string, os.FileInfo, error) {
	abspath, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}

	parentpath := filepath.Dir(abspath)
	parentinfo, err := os.Stat(parentpath)
	if err != nil {
		return "", nil, err
	}
	if (!parentinfo.IsDir()) {
		return "", nil, errors.New("Not a directory")
	}

	return parentpath, parentinfo, nil
}

// Set file path extention
func SetExt(path string, ext string) string {
	if len(filepath.Ext(path)) == 0 {
		return path
	}

	idx := strings.LastIndexByte(path, '.')
	if idx <= 0 {
		return path
	}
	return fmt.Sprintf("%s.%s", path[:idx], ext)
}

func RunCmdWithOutput(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		Fatal("Unable to execute '%s' command: %s: %s", name, err, string(output))
	}
	return string(output)
}
func RunCmd(name string, args ...string) bool {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		Fatal("Unable to execute '%s' command: %s", name, err)
		return false
	}
	return true
}
