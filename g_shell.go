package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var lastExitCode int

// For educational purposes, we use os.StartProcess here.
// In reality, it would be easier to use the package os/exec
// to wrap StartProcess.
// TODO: We're leaving zombie processes behind if we return
// prematurely due to an error condition.
func processCmd(tokens []string) {
	attrs := &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}
	path, err := exec.LookPath(tokens[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Resolving binary: %s\n", err)
		return
	}
	processedTokens := make([]string, 0, len(tokens))
	processedTokens = append(processedTokens, tokens[0])
	for i := 1; i < len(tokens); i++ {
		switch tokens[i] {
		case "<":
			fd, err := os.Open(tokens[i+1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Redirecting stdin: %s\n", err)
				return
			}
			defer fd.Close()
			attrs.Files[0] = fd
			i++
		case ">":
			fd, err := os.OpenFile(tokens[i+1], os.O_RDWR|os.O_CREATE, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Redirecting stdout: %s\n", err)
				return
			}
			defer fd.Close()
			attrs.Files[1] = fd
			i++
		case "|":
			pipeRead, pipeWrite, err := os.Pipe()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Creating pipe: %s\n", err)
			}
			newAttrs := &os.ProcAttr{
				Files: []*os.File{attrs.Files[0], pipeWrite, attrs.Files[2]},
			}
			newAttrs.Files[1] = pipeWrite
			attrs.Files[0] = pipeRead // TODO: Close this end of the pipe?
			p, err := os.StartProcess(path, processedTokens, newAttrs)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Forking process: %s\n", err)
				return
			}
			go func() {
				defer pipeWrite.Close()
				_, err := p.Wait()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Reaping process: %s\n", err)
				}
			}()
			i++
			processedTokens = make([]string, 0, len(tokens))
			processedTokens = append(processedTokens, tokens[i])
			path, err = exec.LookPath(tokens[i])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Resolving binary: %s\n", err)
				return
			}
		default:
			processedTokens = append(processedTokens, tokens[i])
		}
	}
	p, err := os.StartProcess(path, processedTokens, attrs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Forking process: %s\n", err)
		return
	}
	pstate, err := p.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Reaping process: %s\n", err)
	}
	lastExitCode = pstate.ExitCode()
}

func tokenizeMetaChars(token string) []string {
	for _, c := range []string{"<", ">", "|"} {
		after, found := strings.CutPrefix(token, c)
		if found {
			res := []string{c}
			if len(after) > 0 {
				res = append(res, after)
			}
			return res
		}
	}
	return []string{token}
}

func tokenize(cmdline string) []string {
	tokens := []string{}
	for _, tok := range strings.Fields(cmdline) {
		tokens = append(tokens, tokenizeMetaChars(tok)...)
	}
	// Shell variable expansion
	for i, tok := range tokens {
		if tok == "$?" {
			tokens[i] = strconv.Itoa(lastExitCode)
		}
	}
	return tokens
}

func main() {
	fmt.Printf("Welcome to G Shell - The Gonçalves Go Shell!\n")
	fmt.Printf("No tab completion and no command history (for now...)\n")
	fmt.Printf(">> ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := scanner.Text()
		tokens := tokenize(cmd)
		if len(tokens) > 0 {
			processCmd(tokens)
		}
		fmt.Printf(">> ")
	}
}
