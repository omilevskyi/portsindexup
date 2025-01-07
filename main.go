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
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
)

const (
	idxSep       = "|"
	depSep       = " "
	numFields    = 13
	makeFileName = "Makefile"
)

var (
	version, gitCommit string // -ldflags -X main.version=v0.0.0 -X main.gitCommit=[[:xdigit:]] -X main.makeBin=/usr/bin/make

	portsDir    string
	indexFile   string
	helpFlag    bool
	verboseFlag bool
	versionFlag bool

	rootDir string

	makeBin        = "make"
	pathSep        = string([]byte{os.PathSeparator})
	errNotExisting = errors.New("entry does not exist")
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

func checkDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errNotExisting
		}
		return fmt.Errorf("error accessing: %w", err)
	}
	if !info.IsDir() {
		return errors.New("path is not a directory")
	}
	return nil
}

func checkFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return errNotExisting
		}
		return fmt.Errorf("error accessing: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("path is not a file")
	}
	return nil
}

func processOrigin(wp *WorkerPool, removed map[string]struct{}, portsDir, origin, source string) error {
	var cmdDir string
	if filepath.IsAbs(origin) {
		cmdDir = origin
	} else {
		cmdDir = filepath.Join(portsDir, origin)
	}

	if err := checkDir(cmdDir); err != nil {
		if errors.Is(err, errNotExisting) {
			splitted := strings.Split(cmdDir, pathSep)
			if n := len(splitted); n > 1 {
				removed[filepath.Join(splitted[n-2:]...)] = struct{}{}
				return nil
			}
		}
		return fmt.Errorf("%s: %v", cmdDir, err)
	}

	if checkFile(filepath.Join(cmdDir, makeFileName)) == nil {
		wp.AddTask(Task{
			Origin: origin,
			Source: source,
			Cmd:    makeBin,
			Args:   []string{"-C", cmdDir, "describe"},
		})
	}

	return nil
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
	start := time.Now()

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

	nCPU := runtime.NumCPU()

	if versionFlag {
		fmt.Fprintln(os.Stderr, "Version: "+version+", Commit: "+gitCommit+", nCPUs:", nCPU)
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

	origins, chanErrors, removedOrigs, wgErrors := make(map[string][]string), make(chan error, nCPU), make(map[string]struct{}), sync.WaitGroup{}

	wgErrors.Add(1)
	go func() { // [*] read errors from channel and print them to stderr
		defer wgErrors.Done()
		for err := range chanErrors {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	pool := NewWorkerPool(nCPU)
	pool.Start(origins, &chanErrors)

	for _, origin := range flag.Args() {
		if err = processOrigin(pool, removedOrigs, portsDir, origin, "argv"); err != nil {
			fmt.Fprintln(os.Stderr, "processOrigin(argv) error:", err)
		}
	}

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			if err = processOrigin(pool, removedOrigs, portsDir, scanner.Text(), "stdin"); err != nil {
				fmt.Fprintln(os.Stderr, "processOrigin(stdin) error:", err)
			}
		}
		if err = scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "error reading standard input:", err)
		}
	}

	pool.Stop()       // error writers write unwritten data and stop
	close(chanErrors) // close channel to end for loop from goroutine [*]
	wgErrors.Wait()   // wait for goroutine [*] to end

	originLen := len(origins)
	if verboseFlag {
		fmt.Fprintf(os.Stderr, "%d origin(s) stored\n", originLen)
	}

	if originLen < 1 {
		fmt.Fprintf(os.Stderr, "%d origin(s) found\n", originLen)
		return
	}

	strippedOrigins := make(map[string]string, originLen)
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

	writer := bufio.NewWriter(tempFile)
	defer writer.Flush()

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
	lineCount, changedCount, removedCount, writtenCount := 0, 0, 0, 0
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
				if verboseFlag {
					fmt.Fprintf(os.Stderr, "Line %d: %s (%s) has been removed\n", lineCount, namever, origin)
				}
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

		if _, err = fmt.Fprintln(writer, result); err != nil {
			panic(err)
		}
		writtenCount++
	}

	if err = scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading index file:", err)
	}

	if changedCount+removedCount > 0 {
		if err = file.Close(); err != nil {
			panic(err)
		}
		if err = writer.Flush(); err != nil {
			panic(err)
		}
		if err = tempFile.Close(); err != nil {
			panic(err)
		}
		if err = os.Rename(tempFile.Name(), indexFile); err != nil {
			panic(err)
		}
	}

	duration := time.Since(start).Seconds()
	if lineCount == writtenCount {
		fmt.Fprintf(os.Stderr, "%d lines read/written, %d changed, %d removed during %.3f seconds\n",
			lineCount, changedCount, removedCount, duration)
	} else {
		fmt.Fprintf(os.Stderr, "%d lines read, %d changed, %d removed, %d written during %.3f seconds\n",
			lineCount, changedCount, removedCount, writtenCount, duration)
	}
}
