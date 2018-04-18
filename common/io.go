package common

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

const (
	// MaxLineLen is a maximum length of line in Assuan protocol, including
	// space after command and LF.
	MaxLineLen = 1000
)

// ReadLine reads raw request/response in following format: command <parameters>
//
// Empty lines and lines starting with # are ignored as specified by protocol.
// Additinally, status information is silently discarded for now.
func ReadLine(pipe io.Reader) (cmd string, params string, err error) {
	scanner := bufio.NewScanner(pipe)

	var line string
	for {
		if ok := scanner.Scan(); !ok {
			return "", "", scanner.Err()
		}
		line = scanner.Text()

		// We got something that looks like a message. Let's parse it.
		if !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "S ") && len(strings.TrimSpace(line)) != 0 {
			break
		}
	}

	// Part before first whitespace is a command. Everything after first whitespace is paramters.
	parts := strings.SplitN(line, " ", 2)

	// If there is no parameters... (huh!?)
	if len(parts) == 1 {
		return strings.ToUpper(parts[0]), "", nil
	}

	params, err = unescapeParameters(parts[1])
	if err != nil {
		return "", "", err
	}

	// Command is "normalized" to upper case since peer can send
	// commands in any case.
	return strings.ToUpper(parts[0]), params, nil
}

// WriteLine writes request/response to underlying pipe.
// Contents of params is escaped according to requirements of Assuan protocol.
func WriteLine(pipe io.Writer, cmd string, params string) error {
	if len(cmd)+len(params)+2 > MaxLineLen {
		// 2 is for whitespace after command and LF
		return errors.New("too long command or parameters")
	}

	line := []byte(strings.ToUpper(cmd) + " " + escapeParameters(params) + "\n")
	_, err := pipe.Write(line)
	return err
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// WriteData sends passed byte slice using one or more D commands.
// Note: Error may occur even after some data is written so it's better
// to just CAN transaction after WriteData error.
func WriteData(pipe io.Writer, input []byte) error {
	encoded := []byte(escapeParameters(string(input)))
	chunkLen := MaxLineLen - 3 // 3 is for 'D ' and line feed.
	for i := 0; i < len(encoded); i += chunkLen {
		chunk := encoded[i:min(i+chunkLen, len(encoded))]
		chunk = append([]byte{'D', ' '}, chunk...)
		chunk = append(chunk, '\n')

		if _, err := pipe.Write(chunk); err != nil {
			return err
		}
	}
	return nil
}

// WriteComment is special case of WriteLine. "Command" is # and text is parameter.
func WriteComment(pipe io.Writer, text string) error {
	return WriteLine(pipe, "#", text)
}