The dmergefiles program merges columns in tab-delimited files for the same "key"
in the first column of each file.

Columns from each file are added in the order the files are specified. E.g., with
input from two files:

file1:

```
bar	11	22	33
cat	01	02	03
foo	1	2	3
```

file2:

```
bar	44	55	66
car	7	8	9
foo	4	5	6
```

The output is

```
bar	11	22	33	44	55	66
car				7	8	9
cat	01	02	03			
foo	1	2	3	4	5	6
```


## Installation

To get the latest changes

    go get github.com/cuberat/dmergefiles

## Usage

    Usage: ./dmergefiles [options] files ...

    Options:

    -outfile string
        Output file
    -stdout
        Write to standard output
    -v    Be verbose


