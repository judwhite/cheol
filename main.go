package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

var crlf *bool = flag.Bool("crlf", false, `\n to \r\n`)
var lf *bool = flag.Bool("lf", false, `\r\n to \n`)
var recursive *bool = flag.Bool("r", false, "recurse subdirectories")
var verbosity *int = flag.Int("v", 1, "verbosity\n\t0 = only errors\n\t1 = only changed files\n\t2 = report all files\n\t")
var buf []byte = make([]byte, 512)

var changed int64

func main() {
	flag.Parse()

	if *crlf == *lf || *verbosity < 0 || *verbosity > 2 {
		flag.PrintDefaults()
		return
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	start := time.Now()

	processDir(dir)

	if *verbosity >= 1 {
		fmt.Printf("time_taken=%q files_changed=%d\n", time.Since(start), changed)
	}
}

type FileInfos []os.FileInfo

func (fis FileInfos) Len() int {
	return len(fis)
}

func (fis FileInfos) Swap(i, j int) {
	fis[i], fis[j] = fis[j], fis[i]
}

func (fis FileInfos) Less(i, j int) bool {
	a, b := fis[i], fis[j]
	if a.IsDir() && !b.IsDir() {
		return true
	}
	return a.Name() < b.Name()
}

func processDir(dir string) {
	if *verbosity == 2 {
		fmt.Printf("directory=\"%s\"\n", dir)
	}

	var fis FileInfos
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	sort.Sort(fis)
	for _, fi := range fis {
		if fi.IsDir() {
			if *recursive {
				if strings.HasPrefix(fi.Name(), ".") {
					continue
				}
				subdir := filepath.Join(dir, fi.Name())
				processDir(subdir)
			}
			continue
		}

		processFile(dir, fi.Name())
	}
}

func processFile(dir, shortFilename string) {
	fullFilename := filepath.Join(dir, shortFilename)
	f, err := os.Open(fullFilename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	r := bufio.NewReader(f)
	n, err := r.Read(buf)
	if err != nil {
		f.Close()
		log.Fatal(err)
	}

	contentType := http.DetectContentType(buf[:n])

	if strings.HasPrefix(contentType, "text/") {
		_, err := f.Seek(0, os.SEEK_SET)
		if err != nil {
			log.Fatal(err)
		}

		r := bufio.NewReader(f)

		tmp, err := ioutil.TempFile(dir, shortFilename)
		if err != nil {
			log.Fatal(err)
		}
		defer tmp.Close()

		w := bufio.NewWriter(tmp)

		line, isPrefix, rErr := r.ReadLine()
		for rErr != io.EOF {
			_, err := w.Write(line)
			if !isPrefix && err == nil {
				if *crlf {
					_, err = w.Write([]byte{'\r', '\n'})
				} else {
					err = w.WriteByte('\n')
				}
			}
			if err != nil {
				log.Fatal(err)
			}

			line, isPrefix, rErr = r.ReadLine()
		}
		w.Flush()
		tmp.Close()
		f.Close()

		fi1, err := os.Stat(fullFilename)
		if err != nil {
			log.Fatal(err)
		}
		fi2, err := os.Stat(tmp.Name())
		if err != nil {
			log.Fatal(err)
		}
		diff := fi1.Size() - fi2.Size()

		if diff != 0 {
			tmp2, err := ioutil.TempFile(dir, shortFilename)
			if err != nil {
				log.Fatal(err)
			}
			tmp2.Close()
			if err = os.Remove(tmp2.Name()); err != nil {
				log.Fatal(err)
			}

			if err = os.Rename(fullFilename, tmp2.Name()); err != nil {
				log.Fatal(err)
			}

			if err = os.Rename(tmp.Name(), fullFilename); err != nil {
				log.Fatal(err)
			}

			if err = os.Remove(tmp2.Name()); err != nil {
				log.Fatal(err)
			}

			if *verbosity >= 1 {
				fmt.Printf("file=\"%s\" lines_changed=%d\n", fullFilename, abs(diff))
			}
			atomic.AddInt64(&changed, 1)
		} else {
			if err = os.Remove(tmp.Name()); err != nil {
				log.Fatal(err)
			}
			if *verbosity == 2 {
				fmt.Printf("file=\"%s\" lines_changed=%d\n", fullFilename, abs(diff))
			}
		}
	} else {
		if *verbosity == 2 {
			fmt.Printf("file=\"%s\" skipped\n", fullFilename)
		}
	}
}

func abs(a int64) int64 {
	if a < 0 {
		return -a
	}
	return a
}
