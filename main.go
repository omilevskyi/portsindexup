package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
)

const (
	idxSep    = "|"
	depSep    = " "
	numFields = 13
)

var (
	version, gitCommit string // -ldflags -X main.version=v0.0.0 -X main.gitCommit=[[:xdigit:]] -X main.makeBin=/usr/bin/make

	portsDir    string
	indexFile   string
	helpFlag    bool
	verboseFlag bool
	versionFlag bool

	rootDir string
	makeBin string

	pathSep           = string([]byte{os.PathSeparator})
	errNotExistingDir = errors.New("directory does not exist")
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
		output.WriteString(scanner.Text()) // result is concatenated strings without \n
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading command output: %w", err)
	}

	if err := command.Wait(); err != nil {
		return "", fmt.Errorf("error waiting for command to finish: %w", err)
	}

	return output.String(), nil
}

func strip(input string) string {
	if pos := strings.LastIndexByte(input, '-'); pos >= 0 {
		return input[:pos+1]
	}
	return input
}

func replace(source, search, replace string) string {
	if pos := strings.Index(source, search); pos >= 0 {
		return source[:pos] + replace + source[pos+len(search):]
	}
	return source
}

func checkDirAccess(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errNotExistingDir
		}
		return fmt.Errorf("error accessing directory: %w", err)
	}

	if !info.IsDir() {
		return errors.New("path is not a directory")
	}

	return nil
}

<<<<<<< HEAD
func processOrigin(origins map[string][]string, removed map[string]struct{}, portsDir, origin string) []error {
=======
func processOrigin(origins map[string][]string, portsDir, origin string) []error {
>>>>>>> 8f1f82b (Add small improvements)
	var errList []error
	var cmdPath string
	if filepath.IsAbs(origin) {
		cmdPath = origin
	} else {
		cmdPath = filepath.Join(portsDir, origin)
	}

	if err := checkDirAccess(cmdPath); err != nil {
<<<<<<< HEAD
		if errors.Is(err, errNotExistingDir) {
			splitted := strings.Split(cmdPath, pathSep)
			if n := len(splitted); n > 1 {
				removed[filepath.Join(splitted[n-2:]...)] = struct{}{}
				return nil
			}
		}
=======
>>>>>>> 8f1f82b (Add small improvements)
		return []error{fmt.Errorf("%s: %v", cmdPath, err)}
	}

	cmd := exec.Command(makeBin, "-C", cmdPath, "describe")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return []error{fmt.Errorf("error creating stdout pipe for %s: %v", origin, err)}
	}

	if err := cmd.Start(); err != nil {
		return []error{fmt.Errorf("error starting command for %s: %v", origin, err)}
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
			errList = append(errList, fmt.Errorf("line %d: invalid number of fields: %d", lineCount, n))
			continue
		}
		origins[fields[0]] = fields[1:numFields]
	}

	if err := scanner.Err(); err != nil {
		errList = append(errList, fmt.Errorf("error reading output for %s: %v", origin, err))
	}

	if err := stdout.Close(); err != nil {
		errList = append(errList, fmt.Errorf("error closing stdout pipe for %s: %v", origin, err))
	}

	if err := cmd.Wait(); err != nil {
		errList = append(errList, fmt.Errorf("wait for %s failed: %v", origin, err))
	}

	return errList
}

func rootDirectory() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		parentDir := filepath.Dir(currentDir)

		if currentDir == parentDir {
			return currentDir, nil
		}

		currentDir = parentDir
	}
}

func updatePath(dst, src []string, idx int, prefix string, count int) {
	if idx < len(src) && idx < len(dst) && src[idx] != "" {
		splitted := strings.Split(src[idx], pathSep)
		if n := len(splitted); n >= count {
			dst[idx] = filepath.Join(prefix, filepath.Join(splitted[n-count:]...))
		}
	}
}

func safeUpdate(dst []string, didx int, src []string, sidx int) {
	if sidx < len(src) && didx < len(dst) && src[sidx] != "" {
		dst[didx] = src[sidx]
	}
}

func updateDependency(pstr *string, replacements map[string]string, from, to string) {
	if pstr != nil && *pstr != "" {
		var builder strings.Builder
		builder.Grow(len(*pstr))
		for i, f := range strings.Fields(*pstr) {
			if v, ok := replacements[strip(f)]; ok {
				f = v
			}
			if i > 0 {
				builder.WriteString(depSep)
			}
			builder.WriteString(replace(f, from, to))
		}
		if proposed := builder.String(); proposed != *pstr {
			*pstr = proposed
		}
	}
}

func main() {
	flag.StringVar(&portsDir, "ports-dir", "", "Path to the ports directory")
	flag.StringVar(&indexFile, "index-file", "", "Path to the index file")
	flag.BoolVar(&helpFlag, "help", false, "Display help message")
	flag.BoolVar(&verboseFlag, "verbose", false, "Enable verbose output")
	flag.BoolVar(&versionFlag, "version", false, "Show version information")
	flag.Parse()

	if helpFlag {
		fmt.Fprintln(os.Stderr, "Usage: portsindexup [-ports-dir ..] [-index-file] [-help] [-verbose] [port_origins] [< port_origins]")
		os.Exit(0)
	}

	if versionFlag {
		fmt.Fprintln(os.Stderr, "Version: "+version+", Commit: "+gitCommit)
		os.Exit(0)
	}

	var err error
	if rootDir, err = rootDirectory(); err != nil {
		panic(err)
	}

	osRelDate, err := sysCtlUint32("kern.osreldate")
	if err != nil {
		panic(err)
	}

	badOsRelDate := osRelDate[:2] + strings.Repeat("9", len(osRelDate)-2)

	portsDirDefault, err := readStdout(makeBin, []string{"-C", rootDir, "-V", "PORTSDIR"})
	if err != nil {
		panic(err)
	}

	if portsDir == "" {
		portsDir = portsDirDefault
	}

	if verboseFlag {
		fmt.Fprintf(os.Stderr, "make:\t%s\n", makeBin)
		fmt.Fprintf(os.Stderr, "osRelDate:\t%s -> %s\n", badOsRelDate, osRelDate)
		fmt.Fprintf(os.Stderr, "portsDirDefault:\t%s\n", portsDirDefault)
		fmt.Fprintf(os.Stderr, "portsDir:\t%s\n", portsDir)
	}

	origins, removedOrigs := make(map[string][]string), make(map[string]struct{})
	for _, origin := range flag.Args() {
<<<<<<< HEAD
		for _, err = range processOrigin(origins, removedOrigs, portsDir, origin) {
=======
		for _, err = range processOrigin(origins, portsDir, origin) {
>>>>>>> 8f1f82b (Add small improvements)
			fmt.Fprintln(os.Stderr, "processOrigin(argv) error:", err)
		}
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
<<<<<<< HEAD
			for _, err = range processOrigin(origins, removedOrigs, portsDir, scanner.Text()) {
=======
			for _, err = range processOrigin(origins, portsDir, scanner.Text()) {
>>>>>>> 8f1f82b (Add small improvements)
				fmt.Fprintln(os.Stderr, "processOrigin(stdin) error:", err)
			}
		}
		if err = scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading standard input:", err)
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
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
	}()

	file, err := os.Open(indexFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	if verboseFlag {
		fmt.Fprintf(os.Stderr, "index_file:\t%s\n", indexFile)
		fmt.Fprintf(os.Stderr, "temp_file:\t%s\n", tempFile.Name())
	}

	// TODO: detect removal of the origin directory, delete lines from the INDEX file, and update dependency fields
	// 0            1       2            3       4          5          6          7             8        9   10           11         12
	// name-version|portdir|local_prefix|comment|descr_file|maintainer|categories|build_depends|run_deps|www|extract_deps|patch_deps|fetch_deps
	lineCount, changedCount, removedCount := 0, 0, 0
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
		splitted := strings.Split(fields[1], pathSep)
		if n := len(splitted); n > 1 {
			origin := filepath.Join(splitted[n-2:]...)
			if _, ok := removedOrigs[origin]; ok {
				fmt.Fprintf(os.Stderr, "Line %d: %s (%s) has been removed\n", lineCount, namever, origin)
				removedCount++
				continue
			}
		}

		fields = fields[1:numFields]
		if origin, ok := strippedOrigins[strip(namever)]; ok {
			namever = origin

			if described, ok := origins[namever]; ok {
				updatePath(fields, described, 0, portsDirDefault, 2) // portdir: /usr/ports/.dev/devel/readline -> /usr/ports/devel/readline
				updatePath(fields, described, 3, portsDirDefault, 3) // description_file: /usr/ports/.dev/devel/readline/pkg-descr -> /usr/ports/devel/readline//pkg-descr

				safeUpdate(fields, 1, described, 1)  // local_prefix
				safeUpdate(fields, 2, described, 2)  // comment
				safeUpdate(fields, 4, described, 4)  // maintainer
				safeUpdate(fields, 5, described, 5)  // categories
				safeUpdate(fields, 8, described, 11) // www
			}
		}

		updateDependency(&fields[6], strippedOrigins, badOsRelDate, osRelDate)  // build_deps
		updateDependency(&fields[7], strippedOrigins, badOsRelDate, osRelDate)  // run_deps
		updateDependency(&fields[9], strippedOrigins, badOsRelDate, osRelDate)  // exract_deps
		updateDependency(&fields[10], strippedOrigins, badOsRelDate, osRelDate) // patch_deps
		updateDependency(&fields[11], strippedOrigins, badOsRelDate, osRelDate) // fetch_deps

		result := replace(namever, badOsRelDate, osRelDate) + idxSep + strings.Join(fields, idxSep)
		if line != result {
			changedCount++
		}

		fmt.Fprintln(tempFile, result)
	}

	if err = scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading index file:", err)
	}

<<<<<<< HEAD
	if changedCount+removedCount > 0 {
=======
	if changedCount > 0 {
>>>>>>> 8f1f82b (Add small improvements)
		if err = file.Close(); err != nil {
			panic(err)
		}
		if err = tempFile.Close(); err != nil {
			panic(err)
		}
		if err = os.Rename(tempFile.Name(), indexFile); err != nil {
			panic(err)
		}
	}

	fmt.Fprintf(os.Stderr, "%d lines read, %d changed, %d removed\n", lineCount, changedCount, removedCount)
}
