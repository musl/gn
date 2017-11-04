[![Build Status](https://travis-ci.org/musl/gn.svg?branch=master)](https://travis-ci.org/musl/gn)

# gn
A simple **G**enerator **N**oise for use when music would be too distracting.

How to Build this
-------

Install PortAudio: `brew install ...`, `yum install ...`, `apt install ...`, `pacman -S ...`, `pkg install ...`, etc.

```
go get github.com/musl/gn
cd $GOROOT/src/github.com/musl/gn
make vendor commands
./cmd/gn/gn <options>
```

This assumes you have a `$GOPATH` setup correctly and that
`$GOPATH/bin` is in your path.  Refer to `go help gopath` for more
information.
  
Description
-----------
Beats the heck out of `dd if=/dev/urandom of=/dev/dsp`.

