package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

// These flags define which text to prefix to each log entry generated by the Logger.
const (
	// Bits or'ed together to control what's printed.
	// There is no control over the order they appear (the order listed
	// here) or the format they present (as described in the comments).
	// The prefix is followed by a colon only when Llongfile or Lshortfile
	// is specified.
	// For example, flags Ldate | Ltime (or LstdFlags) produce,
	//	2009/01/23 01:23:23 message
	// while flags Ldate | Ltime | Lmicroseconds | Llongfile produce,
	//	2009/01/23 01:23:23.123123 /a/b/c/d.go:23: message
	Ldate         = 1 << iota                  // the date in the local time zone: 2009/01/23
	Ltime                                      // the time in the local time zone: 01:23:23
	Lmicroseconds                              // microsecond resolution: 01:23:23.123123.  assumes Ltime.
	Llongfile                                  // full file name and line number: /a/b/c/d.go:23
	Lshortfile                                 // final file name element and line number: d.go:23. overrides Llongfile
	LUTC                                       // if Ldate or Ltime is set, use UTC rather than the local time zone
	LstdFlags     = Ldate | Ltime | Lshortfile // initial values for the standard logger
)

var DEBUG bool
var FLAG int = LstdFlags
var OUT io.Writer = os.Stderr // destination for output
var mu sync.Mutex             // ensures atomic writes; protects the following fields
var buf []byte                // for accumulating text to write

// Cheap integer to fixed-width decimal ASCII.  Give a negative width to avoid zero-padding.
func itoa(i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	buf = append(buf, b[bp:]...)
}

func formatHeader(t time.Time, level string, file string, line int) {
	if FLAG&LUTC != 0 {
		t = t.UTC()
	}
	if FLAG&(Ldate|Ltime|Lmicroseconds) != 0 {
		if FLAG&Ldate != 0 {
			year, month, day := t.Date()
			itoa(year, 4)
			buf = append(buf, '/')
			itoa(int(month), 2)
			buf = append(buf, '/')
			itoa(day, 2)
			buf = append(buf, ' ')
		}
		if FLAG&(Ltime|Lmicroseconds) != 0 {
			hour, min, sec := t.Clock()
			itoa(hour, 2)
			buf = append(buf, ':')
			itoa(min, 2)
			buf = append(buf, ':')
			itoa(sec, 2)
			if FLAG&Lmicroseconds != 0 {
				buf = append(buf, '.')
				itoa(t.Nanosecond()/1e3, 6)
			}
			buf = append(buf, ' ')
		}
	}

	buf = append(buf, level...)
	buf = append(buf, ' ')

	if FLAG&(Lshortfile|Llongfile) != 0 {
		if FLAG&Lshortfile != 0 {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		buf = append(buf, file...)
		buf = append(buf, ':')
		itoa(line, -1)
		buf = append(buf, ": "...)
	}
}

// Output outputs the string of with level to the writer.
func Output(level string, s string) error {
	now := time.Now() // get this early.
	var file string
	var line int
	mu.Lock()
	defer mu.Unlock()
	if FLAG&(Lshortfile|Llongfile) != 0 {
		// release lock while getting caller info - it's expensive.
		mu.Unlock()
		var ok bool
		_, file, line, ok = runtime.Caller(2)
		if !ok {
			file = "???"
			line = 0
		}
		mu.Lock()
	}
	buf = buf[:0] // clear buffer
	formatHeader(now, level, file, line)
	buf = append(buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf = append(buf, '\n')
	}
	_, err := OUT.Write(buf)
	return err
}

// Debug output the debug info if DEBUG is set to true.
func Debug(a ...interface{}) {
	if !DEBUG {
		return
	}
	Output("[DEBG]", fmt.Sprint(a...))
}

// Debugf output the formated debug info if DEBUG is set to true.
func Debugf(format string, a ...interface{}) {
	if !DEBUG {
		return
	}
	Output("[DEBG]", fmt.Sprintf(format, a...))
}

// Info output the info.
func Info(a ...interface{}) {
	Output("[INFO]", fmt.Sprint(a...))
}

// Infof output the formated info.
func Infof(format string, a ...interface{}) {
	Output("[INFO]", fmt.Sprintf(format, a...))
}
