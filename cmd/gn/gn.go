package main

import (
	"flag"
	"fmt"
	"github.com/Arafatk/glot"
	"github.com/gordonklaus/portaudio"
	"github.com/mjibson/go-dsp/dsputils"
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
)

func check_error(err error) {
	if err != nil {
		panic(err)
	}
}

type Config struct {
	Quiet        bool
	SampleRate   int
	Time         int
	Vol          float64
	BufferCount  int
	SegmentSize  int
	SegmentCount int
	Alpha        float64
	Gain         float64
	Cutoff       int
}

type Buffer struct {
	Left         []float64
	Right        []float64
	SegmentCount int
	SegmentSize  int
}

type Filter struct {
	Alpha  float64
	Cutoff int
	Gain   float64
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
		self.Left[i] = -1.0 + 2.0*rand.Float64()
		self.Right[i] = -1.0 + 2.0*rand.Float64()
	}
}

func (self *Buffer) ShiftAndFill() {
	last_segment := len(self.Left) - self.SegmentSize

	for i := range self.Left {
		if i < self.SegmentSize {
			self.Left[i] = self.Left[last_segment+i]
			self.Right[i] = self.Right[last_segment+i]
		} else {
			self.Left[i] = -1.0 + 2.0*rand.Float64()
			self.Right[i] = -1.0 + 2.0*rand.Float64()
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
	var wg sync.WaitGroup
	win := window.Hann
	size := in.SegmentSize
	half := size / 2

	/*
	 * Make Buffers
	 */

	la := make([]float64, len(in.Left))
	ra := make([]float64, len(in.Left))
	lb := make([]float64, len(la)-size)
	rb := make([]float64, len(ra)-size)

	/*
	 * Fill Buffers
	 */
	copy(la, in.Left)
	copy(ra, in.Right)
	copy(lb, la[half:len(la)-half])
	copy(rb, ra[half:len(ra)-half])

	/*
	 * Apply Windowing Functions
	 */
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < len(la); i += size {
			window.Apply(la[i:i+size], win)
			window.Apply(ra[i:i+size], win)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < len(lb); i += size {
			window.Apply(lb[i:i+size], win)
			window.Apply(rb[i:i+size], win)
		}
	}()

	wg.Wait()

	/*
	 * Convert Real to Complex Numbers
	 */
	lc := dsputils.ToComplex(la)
	rc := dsputils.ToComplex(ra)
	ld := dsputils.ToComplex(lb)
	rd := dsputils.ToComplex(rb)

	/*
	 * Apply FFT, Filter, & IFFT
	 */
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < len(la); i += size {
			ls := lc[i : i+size]
			rs := rc[i : i+size]
			ls = fft.FFT(ls)
			rs = fft.FFT(rs)

			for j := range ls {
				f := 0.0
				// Something something DC and Nyquist bins...
				if j > self.Cutoff && j != size/2 {
					f = 1.0 / (self.Alpha*float64(j) + 1.0)
				}
				ls[j] *= complex(f, 0.0)
				rs[j] *= complex(f, 0.0)
			}

			ls = fft.IFFT(ls)
			rs = fft.IFFT(rs)
			copy(lc[i:i+size], ls)
			copy(rc[i:i+size], rs)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < len(lb); i += size {
			ls := ld[i : i+size]
			rs := rd[i : i+size]
			ls = fft.FFT(ls)
			rs = fft.FFT(rs)

			for j := range ls {
				f := 0.0
				// Something something DC and Nyquist bins...
				if j > self.Cutoff && j != size/2 {
					f = 1.0 / (self.Alpha*float64(j) + 1.0)
				}
				ls[j] *= complex(f, 0.0)
				rs[j] *= complex(f, 0.0)
			}

			ls = fft.IFFT(ls)
			rs = fft.IFFT(rs)
			copy(ld[i:i+size], ls)
			copy(rd[i:i+size], rs)
		}
	}()

	wg.Wait()

	/*
	 * Add Buffers
	 */
	for i := range lb {
		ls := (real(lc[i+half]) + real(ld[i]))
		rs := (real(rc[i+half]) + real(rd[i]))
		out.Left[i] = self.Gain * ls
		out.Right[i] = self.Gain * rs
	}

	/*
		out.Plot()
		os.Exit(-1)
	*/
}

func produce(buffers chan Buffer, config *Config) {
	noise := NewBuffer(config.SegmentCount+1, config.SegmentSize)
	filtered := NewBuffer(config.SegmentCount, config.SegmentSize)
	filter := Filter{
		Alpha:  config.Alpha,
		Cutoff: config.Cutoff,
		Gain:   config.Gain,
	}

	noise.Fill()

	for {
		noise.ShiftAndFill()
		filter.Apply(&noise, &filtered)
		filtered.Mul(config.Vol)
		buffers <- filtered
	}
}

func consume(buffers chan Buffer) func([]float32) {
	return func(out []float32) {
		buf := <-buffers
		buf.Copy(out)
	}
}

func play(config *Config) {
	buffers := make(chan Buffer, config.BufferCount)

	portaudio.Initialize()
	defer portaudio.Terminate()

	go produce(buffers, config)

	stream, err := portaudio.OpenDefaultStream(
		0,
		2,
		float64(config.SampleRate),
		config.SegmentSize*config.SegmentCount,
		consume(buffers))
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
	flag.IntVar(&config.BufferCount, "buffer-count", 3, "number of buffers to calculate ahead of time")
	flag.IntVar(&config.SegmentCount, "segment-count", 10, "segments per buffer")
	flag.IntVar(&config.SegmentSize, "segment-size", 16384, "buffer segment size in samples")
	flag.IntVar(&config.Time, "time", 3600, "length of play in seconds")
	flag.Float64Var(&config.Vol, "volume", 1.0, "output volume as a float")
	flag.Float64Var(&config.Alpha, "alpha", 0.1, "0.0 is white noise, 0.1 is pink-ish noise")
	flag.IntVar(&config.Cutoff, "cutoff", 5, "discard fft buckets starting at 0")
	flag.Float64Var(&config.Gain, "gain", 10.0, "use this to adjust for low alpha values")
	flag.Parse()

	if config.Quiet {
		dev_null, _ := os.Open("/dev/null")
		syscall.Dup2(int(dev_null.Fd()), 1)
		syscall.Dup2(int(dev_null.Fd()), 2)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UTC().UnixNano())

	fmt.Println("gn v0.0.0")
	play(&config)
}
