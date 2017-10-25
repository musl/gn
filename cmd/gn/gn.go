package main

import (
	"flag"
	"fmt"
	"github.com/gordonklaus/portaudio"
	"github.com/mjibson/go-dsp/fft"
	"math"
	"math/rand"
	"os"
	"syscall"
	"time"
)

type Config struct {
	Vol        float64
	Time       int64
	SampleRate float64
	Quiet      bool
	FrameSize  int
}

type Frame struct {
	Left  []float64
	Right []float64
}

func (self *Frame) Fill() {
	for i := range self.Left {
		self.Left[i] = rand.Float64()
		self.Right[i] = rand.Float64()
	}
}

func (self *Frame) ApplyFilter(config *Config) {
	a := 1.0
	max_freq := 20000.0
	cutoff := 20.0
	volume_factor := 1000.0

	lc := fft.FFTReal(self.Left)
	rc := fft.FFTReal(self.Right)

	for i := range lc {
		f := (float64(i) / float64(len(lc))) * max_freq
		if f <= cutoff {
			lc[i] = complex(0, 0)
			rc[i] = complex(0, 0)
			continue
		}
		lc[i] *= complex(a*(1.0/math.Pow(f, a)), 0)
		rc[i] *= complex(a*(1.0/math.Pow(f, a)), 0)
	}

	lc = fft.IFFT(lc)
	rc = fft.IFFT(rc)

	for i := range self.Left {
		if real(lc[i]) > 1.0 || real(lc[i]) < -1.0 {
			lc[i] = complex(0, 0)
		}
		self.Left[i] = real(lc[i]) * volume_factor * config.Vol
		if real(rc[i]) > 1.0 || real(rc[i]) < -1.0 {
			rc[i] = complex(0, 0)
		}
		self.Right[i] = real(rc[i]) * volume_factor * config.Vol
	}

}

func (self *Frame) Copy(out []float32) {
	for i := range self.Left {
		out[i*2] = float32(self.Left[i])
		out[i*2+1] = float32(self.Right[i])
	}
}

func play(config *Config) {
	portaudio.Initialize()
	defer portaudio.Terminate()

	frames := make(chan Frame, 16)

	produce := func() {
		for {
			frames <- make_frame(config)
		}
	}

	consume := func(out []float32) {
		frame := <-frames
		frame.Copy(out)
	}

	stream, err := portaudio.OpenDefaultStream(0, 2, config.SampleRate, config.FrameSize, consume)
	check_error(err)

	check_error(stream.Start())
	defer stream.Close()

	// pre-fill the buffer
	for i := 0; i < len(frames); i++ {
		frames <- make_frame(config)
	}

	go produce()

	time.Sleep(time.Duration(config.Time) * time.Second)
	check_error(stream.Stop())
}

func make_frame(config *Config) Frame {
	f := Frame{}
	f.Left = make([]float64, config.FrameSize)
	f.Right = make([]float64, config.FrameSize)
	f.Fill()
	f.ApplyFilter(config)
	return f
}

func check_error(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	config := Config{}

	flag.Float64Var(&config.Vol, "volume", 0.5, "output volume as a float")
	flag.Int64Var(&config.Time, "time", 3600, "length of play in seconds")
	flag.IntVar(&config.FrameSize, "frame-size", 131072, "frame size")
	flag.Float64Var(&config.SampleRate, "sample-rate", 44100, "samples per second")
	flag.BoolVar(&config.Quiet, "quiet", true, "Don't print messages.")
	flag.Parse()

	if config.Quiet {
		// Shut. Up.
		dev_null, _ := os.Open("/dev/null")
		syscall.Dup2(int(dev_null.Fd()), 1)
		syscall.Dup2(int(dev_null.Fd()), 2)
	}

	fmt.Println("gn v0.0.0")
	play(&config)
}
