
package main

import (
"code.google.com/p/portaudio-go/portaudio"
"fmt"
"flag"
"math/rand"
"os"
"syscall"
"time"
)

type Config struct {
	Vol  float64
	//Var  float64
	//Freq float64
	Time int64
}

func main() {
	config := Config{}

	flag.Float64Var(&config.Vol, "volume", 0.5, "output volume")
	//flag.Float64Var(&config.Var, "amplitude", 0.1, "amplitude variance")
	//flag.Float64Var(&config.Freq, "rate", 0.1, "rate of variance")
	flag.Int64Var(&config.Time, "time", 3600, "length of play")
	flag.Parse();

	fmt.Println("gn v0.0.0")

	// Shut. Up.
	dev_null, _ := os.Open("/dev/null")
	syscall.Dup2(int(dev_null.Fd()), 1)
   	syscall.Dup2(int(dev_null.Fd()), 2)

	play(&config);
}

func play(config *Config) {
	portaudio.Initialize()
	defer portaudio.Terminate()

	h, err := portaudio.DefaultHostApi()
	chk(err)

	device := h.DefaultOutputDevice
	params := portaudio.HighLatencyParameters(nil, device)
	callback := func(out []int32) {
		for i := range out {
			out[i] = int32(float64(rand.Uint32()) * config.Vol)
		}

		// TODO: Apply a slowly shifting EQ to the block of samples.

	}

	stream, err := portaudio.OpenStream(params, callback)
	chk(err)

	defer stream.Close()
	chk(stream.Start())

	time.Sleep(time.Duration(config.Time) * time.Second)
	chk(stream.Stop())
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
