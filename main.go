package main

import (
	"encoding/binary"
	"github.com/hajimehoshi/oto"
	"math"
	"math/rand"
	"time"
)

const (
	sampleRate        = 44100
	channelNum        = 2
	bitDepthInBytes   = 2
	bytesPerSample    = channelNum * bitDepthInBytes
	bufferSizeInBytes = sampleRate * bytesPerSample
)

func main() {
	seed := time.Now().UnixNano()
	privateRand := rand.New(rand.NewSource(seed))

	// Create a new Oto context for audio playback
	context, err := oto.NewContext(sampleRate, channelNum, bitDepthInBytes, bufferSizeInBytes)
	if err != nil {
		panic(err)
	}
	defer context.Close()

	// Create a new Oto player
	player := context.NewPlayer()
	defer player.Close()

	// Create a buffer
	b := make([]byte, bufferSizeInBytes)

	// Generate brown noise and play it back
	for {
		generateBrownNoise(privateRand, b, 0.01)
		_, err := player.Write(b)
		if err != nil {
			panic(err)
		}
	}
}

func generateBrownNoise(r *rand.Rand, buffer []byte, alpha float64) {
	var lastSample float64

	for i := 0; i < len(buffer); i += bytesPerSample {
		// Generate a random number between -1 and 1
		randomSample := 2*r.Float64() - 1

		// Apply an exponential moving average filter to create deeper brown noise
		currentSample := alpha*randomSample + (1-alpha)*lastSample
		lastSample = currentSample

		// Convert the float64 sample to int16
		intSample := int16(math.Round(currentSample * 32767))

		// Write the sample to the buffer for both channels
		binary.LittleEndian.PutUint16(buffer[i:i+2], uint16(intSample))
		binary.LittleEndian.PutUint16(buffer[i+2:i+4], uint16(intSample))
	}
}
