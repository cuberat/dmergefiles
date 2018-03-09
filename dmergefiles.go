package main

// Currently assumes files are in sorted order

import (
    "flag"
    "fmt"
    "os"
)

type Context struct {
    verbose bool
}

type FileSpec struct {
    path string
    fp *os.File
}

func main() {
    opts := flag.NewFlagSet("dmergefiles", flag.ExitOnError)

    opts.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage: %s [options] files ...\n\n", os.Args[0])
        fmt.Fprintf(os.Stderr, "Options:\n\n")
        opts.PrintDefaults()
    }

    var (
        verbose bool
    )

    opts.BoolVar(&verbose, "v", false, "Be verbose")

    opts.Parse(os.Args[1:])

    files := opts.Args()

    if len(files) == 0 {
        opts.Usage()
        os.Exit(-1)
    }

    ctx := new(Context)
    ctx.verbose = verbose

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

    

    return nil
}

func open_files(ctx *Context, file_paths []string) ([]*FileSpec, error) {
    files := make([]*FileSpec, 0, len(file_paths))
    
    for _, path := range file_paths {
        fp, err := os.Open(path)
        if err != nil {
            return files, fmt.Errorf("couldn't open file %s for input: %s", path, err)
        }

        defer fp.Close()

        spec := new(FileSpec)
        spec.path = path
        spec.fp = fp

        files = append(files, spec)
    }

    return files, nil
}
