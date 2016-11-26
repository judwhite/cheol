# cheol
Change line endings.

`go get -u github.com/judwhite/cheol`

Usage:
```
  -crlf  \n to \r\n
  -lf    \r\n to \n
  -r     recurse subdirectories
```

Ignores directories starting with `.`, such as `.git`.

Uses [`http.DetectContentType`](https://golang.org/pkg/net/http/#DetectContentType). Only files with a MIME type starting with `text/` get converted.
