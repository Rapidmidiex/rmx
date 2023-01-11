// NOTE the original always breaks for
// me so have taken one part
// of this, hopefully that is ok
// source: https://github.com/matryer/is

package is

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type T interface {
	Fail()
	FailNow()
}

// I is the test helper harness.
type I struct {
	t       T
	out     io.Writer
	fail    func()
	helpers map[string]struct{}
}

func New(t T) *I {
	return &I{t, os.Stdout, t.FailNow, map[string]struct{}{}}
}

func (is *I) NoErr(err error) {
	if err != nil {
		is.logf("err: %s", err.Error())
	}
}

func (is *I) True(expression bool) {
	if !expression {
		is.log("not true: $ARGS")
	}
}

func (is *I) Equal(a, b any) {
	if areEqual(a, b) {
		return
	}
	// NOTE source: https://github.com/matryer/is/issues/46
	if isNil(a) || isNil(b) || reflect.ValueOf(a).Type() != reflect.ValueOf(b).Type() {
		is.logf("%s != %s", is.valWithType(a), is.valWithType(b))
	} else {
		is.logf("%v != %v", a, b)
	}
}

func (is *I) logf(format string, args ...interface{}) {
	is.log(fmt.Sprintf(format, args...))
}

func (is *I) log(args ...interface{}) {
	s := is.decorate(fmt.Sprint(args...))
	fmt.Fprint(os.Stdout, s)
	is.fail()
}

func (is *I) valWithType(v interface{}) string {
	if isNil(v) {
		return "<nil>"
	}
	return fmt.Sprintf("%[1]T(%[1]v)", v)
}

// isNil gets whether the object is nil or not.
func isNil(object interface{}) bool {
	if object == nil {
		return true
	}
	value := reflect.ValueOf(object)
	kind := value.Kind()
	if kind >= reflect.Chan && kind <= reflect.Slice && value.IsNil() {
		return true
	}
	return false
}

// areEqual gets whether a equals b or not.
func areEqual(a, b interface{}) bool {
	// NOTE source: https://github.com/matryer/is/issues/49
	if isNil(a) || isNil(b) {
		return isNil(a) && isNil(b)
	}

	if reflect.DeepEqual(a, b) {
		return true
	}
	aValue := reflect.ValueOf(a)
	bValue := reflect.ValueOf(b)
	return aValue == bValue
}

// TODO add line and position number - https://github.com/matryer/is/blob/master/is.go
//
// decorate prefixes the string with the file and line of the call site
// and inserts the final newline if needed and indentation tabs for formatting.
// this function was copied from the testing framework and modified.
func (is *I) decorate(s string) string {
	path, lineNumber, ok := is.callerInfo() // decorate + log + public function.
	file := filepath.Base(path)
	if ok {
		// Truncate file name at last file name separator.
		if index := strings.LastIndex(file, "/"); index >= 0 {
			file = file[index+1:]
		} else if index = strings.LastIndex(file, "\\"); index >= 0 {
			file = file[index+1:]
		}
	} else {
		file = "???"
		lineNumber = 1
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s:%d: ", file, lineNumber) // avoids needing to use strconv
	s = escapeFormatString(s)

	lines := strings.Split(s, "\n")
	if l := len(lines); l > 1 && lines[l-1] == "" {
		lines = lines[:l-1]
	}

	for i, line := range lines {
		if i > 0 {
			// Second and subsequent lines are indented an extra tab.
			sb.WriteString("\n\t\t")
		}
		// expand arguments (if $ARGS is present)
		if strings.Contains(line, "$ARGS") {
			args, _ := loadArguments(path, lineNumber)
			line = strings.Replace(line, "$ARGS", args, -1)
		}
		sb.WriteString(line)
	}
	comment, ok := loadComment(path, lineNumber)
	if ok {
		sb.WriteString(" // ")
		comment = escapeFormatString(comment)
		sb.WriteString(comment)
	}
	sb.WriteRune('\n')
	return sb.String()
}

// escapeFormatString escapes strings for use in formatted functions like Sprintf.
func escapeFormatString(fmt string) string {
	return strings.Replace(fmt, "%", "%%", -1)
}
