package cmdtest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// GrepTimeout defines timeout grep is waiting for substring
var GrepTimeout = 30 * time.Second

const extraDebug = false

// GrepTrue reads from reader until finds substring with timeout or fails,
// printing read lines
func (s *IntegrationSuite) GrepTrue(r io.Reader, substr string) {
	s.GrepAll(r, []string{substr})
}

// GrepAll is like GrepTrue but for an array of strings. It waits util the last
// line is found, then looks for all the lines in the read text
func (s *IntegrationSuite) GrepAll(r io.Reader, strs []string) {
	s.GrepAndNotAll(r, strs, nil)
}

// GrepAndNot reads from reader until finds substring with timeout and checks noSubstr was read
// or fails printing read lines
func (s *IntegrationSuite) GrepAndNot(r io.Reader, substr, noSubstr string) {
	s.GrepAndNotAll(r, []string{substr}, []string{noSubstr})
}

// GrepAndNotAll is like GrepAndNot but for arrays of strings. It waits util
// the last line is found, then looks for all the lines in the read text
func (s *IntegrationSuite) GrepAndNotAll(r io.Reader, strs []string, noStrs []string) {
	// If the stream from stdin is read sequentially with Grep(), there was
	// an erratic behaviour where some lines where not processed.

	// Wait until the last substr is found
	_, buf := s.Grep(r, strs[len(strs)-1])
	read := buf.String()

	// Look for the previous messages in the lines read up to that last substr
	for _, st := range strs {
		if !strings.Contains(read, st) {
			fmt.Printf("'%s' is not found in output:\n", st)
			fmt.Println(read)
			fmt.Printf("\nThe complete command output:\n%s", s.logBuf.String())
			s.Stop()
			s.Suite.T().FailNow()
		}
	}

	for _, st := range noStrs {
		if strings.Contains(read, st) {
			fmt.Printf("'%s' should not be in output:\n", st)
			fmt.Println(read)
			fmt.Printf("\nThe complete command output:\n%s", s.logBuf.String())
			s.Stop()
			s.Suite.T().FailNow()
		}
	}
}

// Grep reads from reader until finds substring with timeout
// return result and content that was read
func (s *IntegrationSuite) Grep(r io.Reader, substr string) (bool, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	var found bool

	foundch := make(chan bool, 1)
	scanner := bufio.NewScanner(r)
	go func() {
		for scanner.Scan() {
			t := scanner.Text()
			fmt.Fprintln(buf, t)
			if strings.Contains(t, substr) {
				found = true
				break
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading input:", err)
		}

		foundch <- found
	}()
	select {
	case <-time.After(GrepTimeout):
		if extraDebug {
			fmt.Printf(" >>>> Grep Timeout reached")
		}

		break
	case found = <-foundch:
	}

	if extraDebug {
		fmt.Printf("----------------\nGrep called for substr %q. Found: %v. Read:\n%s\n\n", substr, found, buf.String())
		fmt.Printf("The complete command output so far:\n%s", s.logBuf.String())
	}

	return found, buf
}
