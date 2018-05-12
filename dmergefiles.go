// Copyright (c) 2018 Don Owens <don@regexguy.com>.  All rights reserved.
//
// This software is released under the BSD license:
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:
//
//  * Redistributions of source code must retain the above copyright
//    notice, this list of conditions and the following disclaimer.
//
//  * Redistributions in binary form must reproduce the above
//    copyright notice, this list of conditions and the following
//    disclaimer in the documentation and/or other materials provided
//    with the distribution.
//
//  * Neither the name of the author nor the names of its
//    contributors may be used to endorse or promote products derived
//    from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS
// FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE
// COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT,
// INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
// HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT,
// STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED
// OF THE POSSIBILITY OF SUCH DAMAGE.

// The dmergefiles program merges columns in tab-delimited files for the
// same "key" in the first column of each file. It is assumed that the
// input files are pre-sorted asciibetically (if using unix sort, set env
// LC_ALL="C").
//
// Columns from each file are added in the order the files are
// specified. E.g., with input from two files:
//
// file1:
//    bar	11	22	33
//    cat	01	02	03
//    foo	1	2	3
//
// file2:
//    bar	44	55	66
//    car	7	8	9
//    foo	4	5	6
//
// The output is
//    bar	11	22	33	44	55	66
//    car				7	8	9
//    cat	01	02	03			
//    foo	1	2	3	4	5	6
//
// Installation:
//
// To get the latest changes
//
//     go get github.com/cuberat/dmergefiles
//
// Usage
//    Usage: ./dmergefiles [options] files ...
//
//    Options:
//
//    -outfile string
//        Output file
//    -stdout
//        Write to standard output
//    -v    Be verbose
//
// Input and output files are supported with (de)compression, based on the file
// name extension. Supported
// (de)compression: gzip, bzip2, xz.
package main

import (
    "bufio"
    "flag"
    "fmt"
    "github.com/cuberat/go-libutils/libutils"
    "io"
    "os"
    "strings"
)

type Context struct {
    verbose bool
    delim string
    writer io.Writer
}

type FileSpec struct {
    path string
    reader *bufio.Reader
    closer_func libutils.CloseFunc
    num_cols int
    last_host string
    last_cols []string
}

func main() {
    opts := flag.NewFlagSet("dmergefiles", flag.ExitOnError)

    opts.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage: %s [options] files ...\n\n", os.Args[0])
        fmt.Fprintf(os.Stderr, "Options:\n\n")
        opts.PrintDefaults()
    }

    var (
        out_file string
        to_stdout bool
        verbose bool
    )

    opts.StringVar(&out_file, "outfile", "", "Output file")
    opts.BoolVar(&to_stdout, "stdout", false, "Write to standard output")
    opts.BoolVar(&verbose, "v", false, "Be verbose")


    opts.Parse(os.Args[1:])

    files := opts.Args()

    if len(files) == 0 || (out_file == "" && !to_stdout) {
        opts.Usage()
        os.Exit(-1)
    }

    ctx := new(Context)
    ctx.verbose = verbose
    ctx.delim = "\t"

    if to_stdout {
        ctx.writer = os.Stdout
    } else {
        out_fh, closer_func, err := libutils.OpenFileW(out_file)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Couldn't open file %s for output.", out_file)
            os.Exit(-1)
        }

        defer closer_func()

        ctx.writer = out_fh
    }

    err := process_files(ctx, files)

    if err != nil {
        fmt.Fprintf(os.Stderr, "couldn't process files: %s", err)
        os.Exit(-1)
    }
}

func process_files(ctx *Context, file_paths []string) error {

    // FIXME: optionally sort files first

    files, err := open_files(ctx, file_paths)
    if err != nil {
        return err
    }

    defer func() {
        for _, file := range files {
            file.closer_func()
        }
    }()

    num_cols := init_lines(ctx, files)

    first := ""

    for true {
        first = ""
        // find a hostname to work with
        for _, file := range files {
            if file.last_host != "" {
                first = file.last_host
                break
            }
        }

        for _, file := range files {
            if file.last_host != "" && file.last_host <= first {
                first = file.last_host
            }
        }

        if first == "" {
            // We're done
            break
        }

        cols := make([]string, 0, num_cols + 1)
        cols = append(cols, first)

        for _, file := range files {
            if file.last_host == first {
                if len(file.last_cols) < file.num_cols {
                    cols = append(cols, file.last_cols...)

                    // Pad out to the right number of cols
                    these_cols := make([]string, file.num_cols - len(file.last_cols))
                    cols = append(cols, these_cols...)
                } else {
                    // Make sure we only output the number of cols we found
                    // in the first line.
                    these_cols := file.last_cols[0:file.num_cols]
                    cols = append(cols, these_cols...)
                }
                
                get_next_line(ctx, file)
            } else {
                empty_cols := make([]string, file.num_cols)
                cols = append(cols, empty_cols...)
            }
        }

        out := strings.Join(cols, ctx.delim)
        fmt.Fprintf(ctx.writer, "%s\n", out)
    }

    return nil
}

func init_lines(ctx *Context, files []*FileSpec) int {
    num_cols := int(0)
    for _, file := range files {
        get_next_line(ctx, file)
        file.num_cols = len(file.last_cols)
        num_cols += len(file.last_cols)
    }

    return num_cols
}

func get_next_line(ctx *Context, file *FileSpec) {
    line, err := file.reader.ReadString('\n')
    if err != nil {
        file.last_host = ""
        file.last_cols = nil
        return
    }

    line = strings.TrimSpace(line)
    cols := strings.Split(line, ctx.delim)
    file.last_host = cols[0]
    if len(cols) > 1 {
        file.last_cols = cols[1:]
    } else {
        file.last_cols = []string{}
    }
}

func open_files(ctx *Context, file_paths []string) ([]*FileSpec, error) {
    files := make([]*FileSpec, 0, len(file_paths))
    
    for _, path := range file_paths {
        r, closer_func, err := libutils.OpenFileRO(path)
        if err != nil {
            return files, fmt.Errorf("couldn't open file %s for input: %s", path, err)
        }

        // defer fp.Close()

        spec := new(FileSpec)
        spec.path = path
        spec.closer_func = closer_func
        spec.reader = bufio.NewReader(r)

        files = append(files, spec)
    }

    return files, nil
}
