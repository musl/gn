package main

import (
	"flag"
	"fmt"
	"github.com/Arafatk/glot"
	"github.com/gordonklaus/portaudio"
	"github.com/mjibson/go-dsp/dsputils"
	"github.com/mjibson/go-dsp/fft"
	//"github.com/mjibson/go-dsp/window"
	//"math"
	"math/rand"
	"os"
	"runtime"
	"syscall"
	"time"
)

func check_error(err error) {
	if err != nil {
		panic(err)
	}
}

type Config struct {
	Quiet      bool
	SampleRate int
	Time       int
	Vol        float64
}

type Buffer struct {
	Left         []float64
	Right        []float64
	SegmentCount int
	SegmentSize  int
}

type Filter struct {
	Overlap int
	C       float64
	Gain    float64
}

func (self *Buffer) Dump(path string) {
	f, err := os.Create(path)
	defer f.Close()
	check_error(err)

	for i := range self.Left {
		fmt.Fprintf(f, "%d\t%f\n", i, self.Left[i])
	}

	fmt.Fprintf(f, "\n\n")

	for i := range self.Right {
		fmt.Fprintf(f, "%d\t%f\n", i, self.Right[i])
	}

	f.Sync()
}

func (self *Buffer) Plot() {
	left, _ := glot.NewPlot(2, true, false)
	left.AddPointGroup("Left", "points", self.Left)
	left.SetTitle("Left")

	right, _ := glot.NewPlot(2, true, false)
	right.AddPointGroup("Right", "points", self.Right)
	right.SetTitle("Right")
}

func NewBuffer(segment_count, segment_size int) Buffer {
	buffer := Buffer{
		Left:         make([]float64, segment_count*segment_size),
		Right:        make([]float64, segment_count*segment_size),
		SegmentCount: segment_count,
		SegmentSize:  segment_size,
	}

	return buffer
}

func (self *Buffer) Fill() {
	for i := range self.Left {
		self.Left[i] = rand.Float64()
		self.Right[i] = rand.Float64()
	}
}

func (self *Buffer) ShiftAndFill() {
	last_segment := len(self.Left) - self.SegmentSize

	for i := range self.Left {
		if i < self.SegmentSize {
			self.Left[i] = self.Left[last_segment+i]
			self.Right[i] = self.Right[last_segment+i]
		} else {
			self.Left[i] = rand.Float64()
			self.Right[i] = rand.Float64()
		}
	}
}

func (self *Buffer) Mul(factor float64) {
	for i := range self.Left {
		self.Left[i] *= factor
		self.Right[i] *= factor
	}
}

func (self *Buffer) Copy(out []float32) {
	for i := range self.Left {
		out[i*2] = float32(self.Left[i])
		out[i*2+1] = float32(self.Right[i])
	}
}

func (self Filter) Apply(in, out *Buffer) {
	for i := 1; i < in.SegmentCount-1; i++ {
		a := i*in.SegmentSize - self.Overlap
		b := (i+1)*in.SegmentSize + self.Overlap

		l := dsputils.ToComplex(in.Left[a:b])
		r := dsputils.ToComplex(in.Right[a:b])

		l = fft.FFT(l)
		r = fft.FFT(r)

		for j := range l {
			f := 0.0
			if j > 0 {
				f = 1.0 / (self.C*float64(j) + 1.0)
			}
			l[j] *= complex(f, 0.0)
			r[j] *= complex(f, 0.0)
		}

		l = fft.IFFT(l)
		r = fft.IFFT(r)

		for j := 0; j < in.SegmentSize; j++ {
			n := (i-1)*in.SegmentSize + j
			out.Left[n] = self.Gain * real(l[self.Overlap+j])
			out.Right[n] = self.Gain * real(r[self.Overlap+j])
		}
	}
	/*
		out.Plot()
		os.Exit(-1)
	*/
}

func play(config *Config) {
	portaudio.Initialize()
	defer portaudio.Terminate()

	buffers := make(chan Buffer, 32)

	go func() {
		noise := NewBuffer(12, 16384)
		filtered := NewBuffer(10, 16384)
		filter := Filter{
			Overlap: 8192,
			C:       0.5,
			Gain:    20.0,
		}

		noise.Fill()

		for {
			noise.ShiftAndFill()
			filter.Apply(&noise, &filtered)
			filtered.Mul(config.Vol)
			buffers <- filtered
		}
	}()

	consume := func(out []float32) {
		buf := <-buffers
		buf.Copy(out)
	}

	stream, err := portaudio.OpenDefaultStream(
		0, 2, float64(config.SampleRate), 200000, consume)
	check_error(err)

	check_error(stream.Start())
	defer stream.Close()

	fmt.Println("waiting")
	time.Sleep(time.Duration(config.Time) * time.Second)
	check_error(stream.Stop())
}

func main() {
	config := Config{}

	flag.BoolVar(&config.Quiet, "quiet", true, "Don't print messages.")
	flag.IntVar(&config.SampleRate, "sample-rate", 44100, "samples per second")
	flag.IntVar(&config.Time, "time", 3600, "length of play in seconds")
	flag.Float64Var(&config.Vol, "volume", 1.0, "output volume as a float")
	flag.Parse()

	if config.Quiet {
		dev_null, _ := os.Open("/dev/null")
		syscall.Dup2(int(dev_null.Fd()), 1)
		syscall.Dup2(int(dev_null.Fd()), 2)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Println("gn v0.0.0")
	play(&config)
}
