package main

// Currently assumes files are in sorted order

import (
    "bufio"
    "flag"
    "fmt"
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
    orig_fp *os.File
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
        out_fh, err := os.Create(out_file)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Couldn't open file %s for output.", out_file)
            os.Exit(-1)
        }
        defer out_fh.Close()

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
            file.orig_fp.Close()
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
        fp, err := os.Open(path)
        if err != nil {
            return files, fmt.Errorf("couldn't open file %s for input: %s", path, err)
        }

        // defer fp.Close()

        spec := new(FileSpec)
        spec.path = path
        spec.orig_fp = fp
        spec.reader = bufio.NewReader(fp)

        files = append(files, spec)
    }

    return files, nil
}
