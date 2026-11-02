package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

type terminalPrompter struct {
	input  *os.File
	reader *bufio.Reader
	output io.Writer
}

func (p terminalPrompter) Text(ctx context.Context, label string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if _, err := fmt.Fprint(p.output, label); err != nil {
		return "", err
	}
	line, err := p.reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", fmt.Errorf("read terminal input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func (p terminalPrompter) Secret(ctx context.Context, label string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if _, err := fmt.Fprint(p.output, label); err != nil {
		return "", err
	}
	value, err := term.ReadPassword(int(p.input.Fd()))
	_, _ = fmt.Fprintln(p.output)
	if err != nil {
		return "", fmt.Errorf("read hidden terminal input: %w", err)
	}
	return string(value), nil
}
