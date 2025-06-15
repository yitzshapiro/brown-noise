package main

import (
	"encoding/binary"
	"fmt"
	"github.com/hajimehoshi/oto"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	sampleRate      = 44100
	channelNum      = 2
	bitDepthInBytes = 2

	// Adjust this duration to experiment with latency and CPU usage.
	// Smaller durations => lower latency but more overhead (more Writes).
	// Larger durations => higher latency but less overhead (fewer Writes).
	bufferDuration = 100 * time.Millisecond
)

// We'll keep the lastSample as a package-level variable so that
// brown noise remains continuous across buffers.
var lastSample float64

var alphaVal atomic.Value
var pitchVal atomic.Value
var volumeVal atomic.Value
var tonePhase float64

func init() {
	alphaVal.Store(0.01)
	pitchVal.Store(0.0)
	volumeVal.Store(1.0)
}

func main() {
	seed := time.Now().UnixNano()
	privateRand := rand.New(rand.NewSource(seed))

	go startServer()

	framesPerBuffer := int(float64(sampleRate) * bufferDuration.Seconds())
	bufferSizeInBytes := framesPerBuffer * channelNum * bitDepthInBytes

	context, err := oto.NewContext(sampleRate, channelNum, bitDepthInBytes, bufferSizeInBytes)
	if err != nil {
		panic(err)
	}
	defer context.Close()

	// Create a player from the context.
	player := context.NewPlayer()
	defer player.Close()

	// We use two channels:
	//   1) bufferPool: holds "free" buffers ready to be filled with noise
	//   2) audioCh:    holds "filled" buffers ready for playback
	const poolSize = 4 // # of buffers we can cycle through
	bufferPool := make(chan []byte, poolSize)
	audioCh := make(chan []byte, poolSize)

	// Pre-allocate buffers and send them to bufferPool.
	for i := 0; i < poolSize; i++ {
		bufferPool <- make([]byte, bufferSizeInBytes)
	}

	// Start a goroutine that generates brown noise into free buffers
	// and sends them to audioCh for playback.
	go func() {
		for {
			buf := <-bufferPool // get an empty buffer
			a := alphaVal.Load().(float64)
			p := pitchVal.Load().(float64)
			v := volumeVal.Load().(float64)
			generateBrownNoise(privateRand, buf, a, p, v)
			audioCh <- buf // send the filled buffer for playback
		}
	}()

	// Main loop: read filled buffers from audioCh and write them to the player.
	// After playback, return the buffer to bufferPool for reuse.
	for {
		buf := <-audioCh
		if _, err := player.Write(buf); err != nil {
			panic(err)
		}
		bufferPool <- buf // recycle buffer
	}
}

// generateBrownNoise fills `buffer` with brown noise samples.
// Using a single 32-bit write for both stereo channels.
func generateBrownNoise(r *rand.Rand, buffer []byte, alpha, pitch, vol float64) {
	for i := 0; i < len(buffer); i += 4 {
		randomSample := 2*r.Float64() - 1

		lastSample = alpha*randomSample + (1-alpha)*lastSample
		sample := lastSample
		if pitch > 0 {
			sample += math.Sin(2 * math.Pi * pitch * tonePhase / float64(sampleRate))
		}
		tonePhase++

		s := int16(sample * 32767 * vol)

		u := uint32(s) & 0xFFFF
		binary.LittleEndian.PutUint32(buffer[i:i+4], (u<<16)|u)
	}
}

func startServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err == nil {
			if v := r.FormValue("alpha"); v != "" {
				if val, err := strconv.ParseFloat(v, 64); err == nil {
					alphaVal.Store(val)
				}
			}
			if v := r.FormValue("pitch"); v != "" {
				if val, err := strconv.ParseFloat(v, 64); err == nil {
					pitchVal.Store(val)
				}
			}
			if v := r.FormValue("volume"); v != "" {
				if val, err := strconv.ParseFloat(v, 64); err == nil {
					volumeVal.Store(val)
				}
			}
		}

		a := alphaVal.Load().(float64)
		p := pitchVal.Load().(float64)
		v := volumeVal.Load().(float64)
		fmt.Fprintf(w, `<html><body>
<form method="POST">
Alpha: <input name="alpha" value="%.3f"><br>
Pitch (Hz): <input name="pitch" value="%.2f"><br>
Volume: <input name="volume" value="%.2f"><br>
<input type="submit" value="Update">
</form>
</body></html>`, a, p, v)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
