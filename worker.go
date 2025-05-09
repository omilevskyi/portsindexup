package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

// Task -
type Task struct {
	Origin, Source string
	Cmd            string
	Args           []string
}

// WorkerPool -
type WorkerPool struct {
	tasks    chan Task
	wg       sync.WaitGroup
	muOut    sync.Mutex
	maxCount int
}

// NewWorkerPool -
func NewWorkerPool(maxCount int) *WorkerPool {
	return &WorkerPool{
		tasks:    make(chan Task),
		maxCount: maxCount,
	}
}

// Start -
func (wp *WorkerPool) Start(stdout map[string][]string, stderr *chan error) { // converting "*chan error" to "*chan<- error" is not easy and clear in go1.23.4
	wp.wg.Add(wp.maxCount)
	for i := 0; i < wp.maxCount; i++ {
		go wp.worker(i, stdout, stderr)
	}
}

// Stop -
func (wp *WorkerPool) Stop() {
	close(wp.tasks)
	wp.wg.Wait()
}

// AddTask -
func (wp *WorkerPool) AddTask(task Task) {
	wp.tasks <- task
}

func (wp *WorkerPool) worker(id int, stdoutMap map[string][]string, errPtr *chan error) {
	defer wp.wg.Done()
	for task := range wp.tasks {
		if verboseFlag {
			fmt.Printf("[Worker %d] executing: %s %v for %s (%s)\n", id, task.Cmd, task.Args, task.Origin, task.Source)
		}

		cmd := exec.Command(filepath.Clean(task.Cmd), task.Args...) //#nosec G204

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			if errPtr != nil {
				*errPtr <- fmt.Errorf("error creating stdout pipe for %s (%s): %w", task.Origin, task.Source, err)
			}
			return
		}

		if err := cmd.Start(); err != nil {
			if errPtr != nil {
				*errPtr <- fmt.Errorf("error starting command for %s (%s): %w", task.Origin, task.Source, err)
			}
			return
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
				if errPtr != nil {
					*errPtr <- fmt.Errorf("command for %s (%s), line %d: invalid number of fields: %d", task.Origin, task.Source, lineCount, n)
				}
				continue
			}
			wp.muOut.Lock()
			stdoutMap[fields[0]] = fields[1:numFields]
			wp.muOut.Unlock()
		}

		if err := scanner.Err(); err != nil && errPtr != nil {
			*errPtr <- fmt.Errorf("error reading stdout for %s (%s): %w", task.Origin, task.Source, err)
		}

		// looks like cmd.Wait() closes both stdout and stderr
		if err := cmd.Wait(); err != nil && errPtr != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					*errPtr <- fmt.Errorf("command for %s (%s) exited with code: %d", task.Origin, task.Source, status.ExitStatus())
				}
			} else {
				*errPtr <- fmt.Errorf("wait for %s (%s) failed: %w", task.Origin, task.Source, err)
			}
		}
	}
}
