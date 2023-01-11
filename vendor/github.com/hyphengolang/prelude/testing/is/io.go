package is

import (
	"bufio"
	"os"
	"regexp"
	"runtime"
	"strings"
)

const maxStackLen = 50

var reIsSourceFile = regexp.MustCompile(`is(-1.7)?\.go$`)

func (is *I) callerInfo() (path string, line int, ok bool) {
	var pc [maxStackLen]uintptr
	// Skip two extra frames to account for this function
	// and runtime.Callers itself.
	n := runtime.Callers(2, pc[:])
	if n == 0 {
		panic("is: zero callers found")
	}
	frames := runtime.CallersFrames(pc[:n])
	var firstFrame, frame runtime.Frame
	for more := true; more; {
		frame, more = frames.Next()
		if reIsSourceFile.MatchString(frame.File) {
			continue
		}
		if firstFrame.PC == 0 {
			firstFrame = frame
		}
		if _, ok := is.helpers[frame.Function]; ok {
			// Frame is inside a helper function.
			continue
		}
		return frame.File, frame.Line, true
	}
	// If no "non-helper" frame is found, the first non is frame is returned.
	return firstFrame.File, firstFrame.Line, true
}

// loadArguments gets the arguments from the function call
// on the specified line of the file.
func loadArguments(path string, line int) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	i := 1
	for s.Scan() {
		if i != line {
			i++
			continue
		}
		text := s.Text()
		braceI := strings.Index(text, "(")
		if braceI == -1 {
			return "", false
		}
		text = text[braceI+1:]
		cs := bufio.NewScanner(strings.NewReader(text))
		cs.Split(bufio.ScanBytes)
		j := 0
		c := 1
		for cs.Scan() {
			switch cs.Text() {
			case ")":
				c--
			case "(":
				c++
			}
			if c == 0 {
				break
			}
			j++
		}
		text = text[:j]
		return text, true
	}
	return "", false
}

// loadComment gets the Go comment from the specified line
// in the specified file.
func loadComment(path string, line int) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	i := 1
	for s.Scan() {
		if i != line {
			i++
			continue
		}

		text := s.Text()
		commentI := strings.Index(text, "// ")
		if commentI == -1 {
			return "", false // no comment
		}
		text = text[commentI+2:]
		text = strings.TrimSpace(text)
		return text, true
	}
	return "", false
}
