package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

/*
  cat /var/log/pacman.log | ./date-filter -after="2019-09-16 17:20" -before="2019-09-16 17:22" -regexp="[^\[].+[^\]]" -delimiter=" "
*/

func main() {
	var (
		err     error
		indexes []int
		re      *regexp.Regexp

		before time.Time
		after  time.Time

		cmdDelimiter string
		cmdRegExp    string
		cmdElements  string
		cmdBefore    string
		cmdAfter     string

		cmdVerbose bool
	)

	// set custom logging flags
	log.SetFlags(0)

	// input arguments
	flag.StringVar(&cmdDelimiter, "delimiter", " ", "default delimiter for line elements")
	flag.StringVar(&cmdElements, "elements", "1,2", "list of elements for date, comma separated")

	// flag.StringVar(&cmdRegExp, "regexp", `[^\[].+[^\]]`, "default regular expression for date extraction")
	flag.StringVar(&cmdRegExp, "regexp", `.+`, "default regular expression for date extraction")

	flag.StringVar(&cmdBefore, "before", time.Now().Format("2006-01-02 15:04:05"), "list of elements for date, comma separated")
	flag.StringVar(&cmdAfter, "after", time.Unix(0, 1).Format("2006-01-02 15:04:05"), "filter lines that have date after specified")

	flag.BoolVar(&cmdVerbose, "verbose", false, "print parse errors to stderr")

	flag.Parse()

	// element indexes
	indexes, err = ParseListOfElements(cmdElements)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	// parse date string to object
	before, err = ParseDate(cmdBefore)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	// parse date string to object
	after, err = ParseDate(cmdAfter)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	// date regex
	re, err = CompileRegExp(cmdRegExp)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	// scan stdin
	scanner := bufio.NewScanner(os.Stdin)

	// used to store lines count
	var n int

	for scanner.Scan() {
		// increment lines count
		n++

		// store original input line content
		b := scanner.Bytes()

		// extract date from input line content
		bb := bytes.Split(b, []byte(cmdDelimiter))
		bb = CleanMultipleSequentialSeparators(bb)
		bb = ExtractElements(bb, indexes)
		date := string(ExtractDateUsingRegExp(bytes.Join(bb, []byte(cmdDelimiter)), re))

		// parse date string to object
		t, err := ParseDate(date)
		if err != nil {
			if cmdVerbose {
				log.Printf("#%d: %s\n", n, err.Error())
			}

			continue
		}

		if t.After(after) {
			if t.Before(before) {
				fmt.Println(scanner.Text())
			}
		}
	}

	if err := scanner.Err(); err != nil {
		os.Exit(0)
	}
}

// ParseDate convinience function to parse supplied date
func ParseDate(date string) (time.Time, error) {
	t, err := dateparse.ParseLocal(date)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse date '%s'", date)
	}

	return t, nil
}

// CompileRegExp convinience function to compile supplied regular expression
func CompileRegExp(expr string) (*regexp.Regexp, error) {
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression: %v", err)
	}

	return re, nil
}

// ParseListOfElements returns parsed list of element indexes
func ParseListOfElements(elements string) ([]int, error) {
	out := make([]int, 0)

	for _, el := range strings.FieldsFunc(elements, func(c rune) bool { return c == ',' }) {
		i, err := strconv.Atoi(el)
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s to number", el)
		}

		out = append(out, i)
	}

	return out, nil
}

// CleanMultipleSequentialSeparators returns new slice of slices of bytes without multiple sequential separators
func CleanMultipleSequentialSeparators(bb [][]byte) [][]byte {
	// slice of slices of bytes splited by separator with multiple sequential separator condensed to single separator
	bbf := make([][]byte, 0, len(bb))

	// loop-over original bytes slice
	for i := 0; i < len(bb); i++ {
		// add bytes to new slice then current bytes are not separator
		if !bytes.Equal(bb[i], []byte{}) {
			bbf = append(bbf, bb[i])

			continue
		}

		// current bytes are separator, skip then next bytes also separtors
		if i+1 < len(bb) {
			if bytes.Equal(bb[i], bb[i+1]) {
				continue
			}
		}
	}

	return bbf
}

// ExtractElements returns new slice of slices of bytes with only requested elements
func ExtractElements(bb [][]byte, elements []int) [][]byte {
	// slice of slices of bytes splited by separator that should contain elements specified by index-1
	bbf := make([][]byte, 0, len(bb))

	// loop-over desired elements and filter slice of slices of bytes
	for _, n := range elements {
		// use n-1 to start element enumeration from 1
		if n-1 < 0 {
			continue // skip negative index
		}
		if n-1 > len(bb) || len(bb) == 0 {
			continue // index out of range
		}

		bbf = append(bbf, bb[n-1])
	}

	return bbf
}

// ExtractDateUsingRegExp additional RegExp filter for bytes
func ExtractDateUsingRegExp(b []byte, re *regexp.Regexp) []byte {
	return re.Find(b)
}
