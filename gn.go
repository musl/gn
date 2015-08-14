
package main

import (
"code.google.com/p/portaudio-go/portaudio"
"github.com/mjibson/go-dsp/fft"
//"github.com/mjibson/go-dsp/window"
"flag"
"fmt"
"math"
//"math/cmplx"
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
		self.Left[i] = rand.Float64();
		self.Right[i] = rand.Float64();
	}
}

func (self *Frame) Filter(config *Config) {
	i := 0
	//p := math.Pi * 0.0625
	p := 0.0
	t := 0.0
	s := 0.1 / config.SampleRate
	m := 0.0
	n := 0.0001
	v := 0.0
	x := 0.0

	l := fft.FFTReal(self.Left)
	r := fft.FFTReal(self.Right)

	m = float64(len(l))
	for i = range l {
		p = 0.5 * math.Sin(2.0 * math.Pi * t)
		_, t = math.Modf(t + s)
		x = float64(i)
		v = 0.5 + 0.5 * math.Cos(p + ((1.0 + n) / ((n * m / x) * m + x)) * math.Pi * x)
		self.Left[i] = real(l[i]) * v
		self.Right[i] = real(r[i]) * v
	}

	l = fft.IFFTReal(self.Left)
	r = fft.IFFTReal(self.Right)

	for i := range self.Left {
		self.Left[i] = real(l[i]) * config.Vol
		self.Right[i] = real(r[i]) * config.Vol
	}
}

func (self *Frame) DePop(last *Frame) {
	n := 8
	l := last.Left[n - 1]
	r := last.Right[n - 1]
	a := 0.0
	b := 0.0

	for i := 0; i < n; i ++ {
		a = float64(i) / float64(n)
		b = 1.0 - a
		self.Left[i] = self.Left[i] * a + l * b
		self.Right[i] = self.Right[i] * a + r * b
	}
}

func (self *Frame) Copy(out []float32) {
	for i := range self.Left {
		out[i * 2] = float32(self.Left[i])
		out[i * 2 + 1] = float32(self.Right[i])
	}
}

func play(config *Config) {
	portaudio.Initialize()
	defer portaudio.Terminate()

	frames := make(chan Frame, 4)
	last := make_frame(config)

	produce := func() {
		for {
			frames<- make_frame(config)
		}
	}

	consume := func(out []float32) {
		frame := <-frames
		frame.DePop(&last)
		frame.Copy(out)
		last = frame
	}

	stream, err := portaudio.OpenDefaultStream(0, 2, config.SampleRate, config.FrameSize, consume)
	check_error(err)

	check_error(stream.Start())
	defer stream.Close()

	// pre-fill the buffer
	for i := 0; i < len(frames); i++ {
		frames<- make_frame(config)
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
	f.Filter(config)
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
	play(&config);
}

