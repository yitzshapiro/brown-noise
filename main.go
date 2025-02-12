package main

import (
	"encoding/binary"
	"github.com/hajimehoshi/oto"
	"math/rand"
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

func main() {
	seed := time.Now().UnixNano()
	privateRand := rand.New(rand.NewSource(seed))

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
			generateBrownNoise(privateRand, buf, 0.01)
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
func generateBrownNoise(r *rand.Rand, buffer []byte, alpha float64) {
	for i := 0; i < len(buffer); i += 4 {
		randomSample := 2*r.Float64() - 1

		lastSample = alpha*randomSample + (1-alpha)*lastSample

		s := int16(lastSample * 32767)

		u := uint32(s) & 0xFFFF
		binary.LittleEndian.PutUint32(buffer[i:i+4], (u<<16)|u)
	}
}
