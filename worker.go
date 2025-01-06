package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

// RunTask -
type RunTask struct {
	Origin, Source string
	Cmd            string
	Args           []string
}

// WorkerPool -
type WorkerPool struct {
	tasks    chan RunTask
	wg       sync.WaitGroup
	muOut    sync.Mutex
	muErr    sync.Mutex
	maxCount int
}

// NewWorkerPool -
func NewWorkerPool(maxCount int) *WorkerPool {
	return &WorkerPool{
		tasks:    make(chan RunTask),
		maxCount: maxCount,
	}
}

// Start -
func (wp *WorkerPool) Start(stdout map[string][]string, stderr *[]error) {
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
func (wp *WorkerPool) AddTask(task RunTask) {
	wp.tasks <- task
}

func (wp *WorkerPool) worker(id int, stdoutMap map[string][]string, errSlice *[]error) {
	defer wp.wg.Done()
	for task := range wp.tasks {
		if verboseFlag {
			fmt.Printf("[Worker %d] executing: %s %v for %s (%s)\n", id, task.Cmd, task.Args, task.Origin, task.Source)
		}

		cmd := exec.Command(task.Cmd, task.Args...)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			if errSlice != nil {
				wp.muErr.Lock()
				*errSlice = append(*errSlice, fmt.Errorf("error creating stdout pipe for %s (%s): %w", task.Origin, task.Source, err))
				wp.muErr.Unlock()
			}
			continue
		}

		if err := cmd.Start(); err != nil {
			if errSlice != nil {
				wp.muErr.Lock()
				*errSlice = append(*errSlice, fmt.Errorf("error starting command for %s (%s): %w", task.Origin, task.Source, err))
				wp.muErr.Unlock()
			}
			continue
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
				if errSlice != nil {
					wp.muErr.Lock()
					*errSlice = append(*errSlice, fmt.Errorf("command for %s (%s), line %d: invalid number of fields: %d",
						task.Origin, task.Source, lineCount, n))
					wp.muErr.Unlock()
				}
				continue
			}
			wp.muOut.Lock()
			stdoutMap[fields[0]] = fields[1:numFields]
			wp.muOut.Unlock()
		}

		if err := scanner.Err(); err != nil && errSlice != nil {
			wp.muErr.Lock()
			*errSlice = append(*errSlice, fmt.Errorf("error reading stdout for %s (%s): %w", task.Origin, task.Source, err))
			wp.muErr.Unlock()
		}

		// looks like cmd.Wait() closes both stdout and stderr
		if err := cmd.Wait(); err != nil && errSlice != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					wp.muErr.Lock()
					*errSlice = append(*errSlice, fmt.Errorf("command for %s (%s) exited with code: %d", task.Origin, task.Source, status.ExitStatus()))
					wp.muErr.Unlock()
				}
			} else {
				wp.muErr.Lock()
				*errSlice = append(*errSlice, fmt.Errorf("wait for %s (%s) failed: %w", task.Origin, task.Source, err))
				wp.muErr.Unlock()
			}
		}
		runtime.Gosched()
	}
}
