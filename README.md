# gn
A simple **N**oise **G**enerator for use when music would be too distracting written in go.  Yeah, the letters are in an order different to the initialism.  It's meant to sound like the noise polite people make when they're realizing that their coworkers are loud and don't want to tell them to pipe down.

Summary
-------
    go get code.google.com/p/portaudio-go/portaudio
    go get github.com/musl/gn
    gn <options>

This assumes you have a `$GOPATH` setup correctly and that `$GOPATH/bin` is in your path.  Refer to `go help gopath` for more information.  I assume you know how to build go apps.  I don't yet care if this app doesn't build on platform *x*, architecture *y*, or planet *z*.  You'll have to give me a reason.
  
Description
-----------
Beats the heck out of `dd if=/dev/urandom of=/dev/dsp`. Soon, this code will support modulating the overall volume, a configurable modulating FFT filter, and more.