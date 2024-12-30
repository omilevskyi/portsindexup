package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"

	"golang.org/x/sys/unix"
)

const (
	idxSep    = "|"
	rootDir   = "/"
	dirSep    = "/"
	depSep    = " "
	numFields = 13
)

var (
	// DEBUG is the string equivalent of Dbg
	DEBUG, version, gitCommit string // -ldflags -X main.DEBUG=[[:digit:]] -X main.version=v0.0.0 -X main.gitCommit=[[:xdigit:]]

	// Dbg is debug level, 0 - no noise
	Dbg int

	portsDir    string
	indexFile   string
	helpFlag    bool
	verboseFlag bool
	makeBin     string = "make"
)

func readStdout(cmdPath string, args []string) (string, error) {
	var output bytes.Buffer

	command := exec.Command(cmdPath, args...)

	stdout, err := command.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("error setting up stdout pipe: %w", err)
	}

	if err := command.Start(); err != nil {
		return "", fmt.Errorf("error running the command: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		output.WriteString(scanner.Text()) // result is concatenated strings without \n !!
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading command output: %w", err)
	}

	if err := command.Wait(); err != nil {
		return "", fmt.Errorf("error waiting for command to finish: %w", err)
	}

	return output.String(), nil
}

func sysCtlUint32(param string) (string, error) {
	data, err := unix.SysctlRaw(param) // unix.Sysctl(param) does not work for osRelDate
	if err != nil {
		return "", fmt.Errorf("error reading sysctl: %w", err)
	}

	if len(data) < 4 {
		return "", fmt.Errorf("unexpected data length: %d", len(data))
	}

	return fmt.Sprint(binary.LittleEndian.Uint32(data)), nil // FreeBSD amd64 has Little Endian
}

func strip(input string) string {
	if pos := strings.LastIndexByte(input, '-'); pos >= 0 {
		return input[:pos+1]
	}
	return input
}

func replace(s, search, replace string) string {
	if pos := strings.Index(s, search); pos >= 0 {
		return s[:pos] + replace + s[pos+len(search):]
	}
	return s
}

func checkDirAccess(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist")
		}
		return fmt.Errorf("error accessing directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory")
	}

	return nil
}

// TODO: переделать на []error
func processOrigin(origins map[string][]string, portsDir, origin string) []string {
	var errList []string
	var cmdPath string
	if filepath.IsAbs(origin) {
		cmdPath = origin
	} else {
		cmdPath = filepath.Join(portsDir, origin)
	}

	if err := checkDirAccess(cmdPath); err != nil {
		return []string{fmt.Sprintf("%s: %v", cmdPath, err)}
	}

	cmd := exec.Command(makeBin, "-C", cmdPath, "describe")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return []string{fmt.Sprintf("error creating stdout pipe for %s: %v", origin, err)}
	}

	if err := cmd.Start(); err != nil {
		return []string{fmt.Sprintf("error starting command for %s: %v", origin, err)}
	}

	// $(make describe) output's line record format is slightly different from INDEX
	// 0            1       2            3       4          5          6          7            8          9          10            11       12
	// name-version|portdir|local_prefix|comment|descr_file|maintainer|categories|extract_deps|patch_deps|fetch_deps|build_depends|run_deps|www
	lineCount := 0
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		lineCount++
		fields := strings.Split(scanner.Text(), idxSep)
		if n := len(fields); n < numFields {
			errList = append(errList, fmt.Sprintf("line %d: invalid number of fields: %d", lineCount, n))
			continue
		}
		origins[fields[0]] = fields[1:numFields]
	}

	if err := scanner.Err(); err != nil {
		errList = append(errList, fmt.Sprintf("error reading output for %s: %v", origin, err))
	}

	if err := stdout.Close(); err != nil {
		errList = append(errList, fmt.Sprintf("error closing stdout pipe for %s: %v", origin, err))
	}

	if err := cmd.Wait(); err != nil {
		errList = append(errList, fmt.Sprintf("wait for %s failed: %v", origin, err))
	}

	return errList
}

func main() {
	flag.StringVar(&portsDir, "ports-dir", "", "Path to the ports directory")
	flag.StringVar(&indexFile, "index-file", "", "Path to the index file")
	flag.BoolVar(&helpFlag, "help", false, "Display help message")
	flag.BoolVar(&verboseFlag, "verbose", false, "Enable verbose output")
	flag.Parse()

	if helpFlag {
		fmt.Fprintln(os.Stderr, "Usage: portsindexup [-ports-dir ..] [-index-file] [-help] [-verbose] [port_origins] [< port_origins]")
		os.Exit(201)
	}

	osRelDate, err := sysCtlUint32("kern.osreldate")
	if err != nil {
		panic(err)
	}

	portsDirDefault, err := readStdout(makeBin, []string{"-C", rootDir, "-V", "PORTSDIR"})
	if err != nil {
		panic(err)
	}

	if portsDir == "" {
		portsDir = portsDirDefault
	}

	badOsRelDate := osRelDate[:2] + strings.Repeat("9", len(osRelDate)-2)

	if verboseFlag {
		fmt.Fprintf(os.Stderr, "make:\t%s\n", makeBin)
		fmt.Fprintf(os.Stderr, "osRelDate:\t%s -> %s\n", badOsRelDate, osRelDate)
		fmt.Fprintf(os.Stderr, "portsDirDefault:\t%s\n", portsDirDefault)
		fmt.Fprintf(os.Stderr, "portsDir:\t%s\n", portsDir)
	}

	origins := make(map[string][]string)
	for _, origin := range flag.Args() {
		for _, e := range processOrigin(origins, portsDir, origin) {
			fmt.Fprintf(os.Stderr, "processOrigin(args) error: %s\n", e)
		}
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			for _, e := range processOrigin(origins, portsDir, scanner.Text()) {
				fmt.Fprintf(os.Stderr, "processOrigin(stdin) error: %s\n", e)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "error reading standard input: %v\n", err)
		}
	}

	if verboseFlag {
		fmt.Fprintf(os.Stderr, "%d origin(s) stored\n", len(origins))
	}

	if len(origins) < 1 {
		return
	}

	strippedOrigins := make(map[string]string, len(origins))
	for k := range origins {
		strippedOrigins[strip(k)] = k
	}

	if indexFile == "" {
		fname, err := readStdout(makeBin, []string{"-C", portsDir, "-V", "INDEXFILE"})
		if err != nil {
			panic(err)
		}
		indexFile = filepath.Join(portsDir, fname)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(indexFile), filepath.Base(indexFile)+".")
	if err != nil {
		panic(err)
	}

	if verboseFlag {
		fmt.Fprintf(os.Stderr, "index_file:\t%s\n", indexFile)
		fmt.Fprintf(os.Stderr, "temp_file:\t%s\n", tempFile.Name())
	}

	file, err := os.Open(indexFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// 0            1       2            3       4          5          6          7             8        9   10           11         12
	// name-version|portdir|local_prefix|comment|descr_file|maintainer|categories|build_depends|run_deps|www|extract_deps|patch_deps|fetch_deps
	lineCount, changedCount := 0, 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		fields := strings.Split(line, idxSep)

		if n := len(fields); n < numFields {
			fmt.Fprintf(os.Stderr, "Line %d: invalid number of fields: %d\n", lineCount, n)
			continue
		}

		namever := fields[0]
		fields = fields[1:numFields]
		if origin, ok := strippedOrigins[strip(namever)]; ok {
			namever = origin

			if describedValue, ok := origins[namever]; ok {
				if describedValue[0] != "" {
					splitted := strings.Split(describedValue[0], dirSep)
					if n := len(splitted); n > 2 {
						fields[0] = filepath.Join(portsDirDefault, filepath.Join(splitted[n-2:]...))
					}
				}

				if describedValue[3] != "" {
					splitted := strings.Split(describedValue[3], dirSep)
					if n := len(splitted); n > 3 {
						fields[3] = filepath.Join(portsDirDefault, filepath.Join(splitted[n-3:]...))
					}
				}

				for _, i := range []int{1, 2, 4, 5} {
					if describedValue[i] != "" {
						fields[i] = describedValue[i]
					}
				}

				fields[8] = describedValue[11]
			}
		}

		//                      build_deps
		//                      |  run_deps
		//                      |  |  exract_deps
		//                      |  |  |  patch_deps
		//                      |  |  |  |   fetch_deps
		//                      |  |  |  |   |
		for _, i := range []int{6, 7, 9, 10, 11} {
			deps := strings.Fields(fields[i])
			for ii, dep := range deps {
				if nv, ok := strippedOrigins[strip(dep)]; ok {
					dep = nv
				}
				deps[ii] = replace(dep, badOsRelDate, osRelDate)
			}
			fields[i] = strings.Join(deps, depSep)
		}

		namever = replace(namever, badOsRelDate, osRelDate)
		result := namever + idxSep + strings.Join(fields, idxSep)

		if line != result {
			changedCount++
		}

		fmt.Fprintf(tempFile, "%s\n", result)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading index file: %v\n", err)
	}

	if err := tempFile.Close(); err != nil {
		panic(err)
	}

	if err := file.Close(); err != nil {
		panic(err)
	}

	if changedCount > 0 {
		if err := os.Rename(tempFile.Name(), indexFile); err != nil {
			panic(err)
		}
	} else if err := os.Remove(tempFile.Name()); err != nil {
		panic(err)
	}

	fmt.Fprintf(os.Stderr, "%d line(s) read, %d changed\n", lineCount, changedCount)
}
