package main

import (
	"encoding/binary"
	"github.com/hajimehoshi/oto"
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestGenerateBrownNoiseAlphaValues(t *testing.T) {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))
	buffer := make([]byte, bufferSizeInBytes)

	alphas := []float64{0.005, 0.01, 0.05, 0.1}

	for _, alpha := range alphas {
		generateBrownNoise(r, buffer, alpha)

		// Calculate the average and standard deviation of the samples
		avg, stddev := calculateAverageAndStdDev(buffer)

		// Check if the average and standard deviation are within expected ranges
		if avg < -1500 || avg > 1500 {
			t.Errorf("Unexpected average for alpha %f: %f", alpha, avg)
		}

		if stddev < 500 || stddev > 5000 {
			t.Errorf("Unexpected standard deviation for alpha %f: %f", alpha, stddev)
		}

		// Check if the maximum and minimum sample values are within the expected range
		minSample, maxSample := calculateMinMax(buffer)

		if minSample < -32767 || maxSample > 32767 {
			t.Errorf("Unexpected min and max sample values for alpha %f: min %f, max %f", alpha, minSample, maxSample)
		}

		// Check if the zero-crossing rate is within the expected range
		zeroCrossingRate := calculateZeroCrossingRate(buffer)

		if zeroCrossingRate < 0.001 || zeroCrossingRate > 0.2 {
			t.Errorf("Unexpected zero-crossing rate for alpha %f: %f", alpha, zeroCrossingRate)
		}

	}
}

func TestGenerateBrownNoiseOutputRange(t *testing.T) {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))
	buffer := make([]byte, bufferSizeInBytes)

	generateBrownNoise(r, buffer, 0.01)

	for i := 0; i < len(buffer); i += bytesPerSample {
		sample := int16(binary.LittleEndian.Uint16(buffer[i : i+2]))

		if sample < -32768 || sample > 32767 {
			t.Errorf("Sample out of range: %d", sample)
		}
	}
}

func TestOtoContextAndPlayer(t *testing.T) {
	context, err := oto.NewContext(sampleRate, channelNum, bitDepthInBytes, bufferSizeInBytes)
	if err != nil {
		t.Errorf("Failed to create Oto context: %v", err)
	}
	defer context.Close()

	player := context.NewPlayer()
	defer player.Close()

	if player == nil {
		t.Error("Failed to create Oto player")
	}
}

// HELPERS

// Helper function to calculate the average and standard deviation sample values
func calculateAverageAndStdDev(buffer []byte) (float64, float64) {
	sum := 0.0
	sqSum := 0.0
	count := 0

	for i := 0; i < len(buffer); i += bytesPerSample {
		sample := float64(int16(binary.LittleEndian.Uint16(buffer[i : i+2])))

		sum += sample
		sqSum += sample * sample
		count++
	}

	avg := sum / float64(count)
	variance := (sqSum / float64(count)) - (avg * avg)
	stddev := math.Sqrt(variance)

	return avg, stddev
}

// Helper function to calculate the minimum and maximum sample values
func calculateMinMax(buffer []byte) (float64, float64) {
	minSample := math.MaxFloat64
	maxSample := -math.MaxFloat64

	for i := 0; i < len(buffer); i += bytesPerSample {
		sample := float64(int16(binary.LittleEndian.Uint16(buffer[i : i+2])))

		if sample < minSample {
			minSample = sample
		}

		if sample > maxSample {
			maxSample = sample
		}
	}

	return minSample, maxSample
}

// Helper function to calculate the zero-crossing rate
func calculateZeroCrossingRate(buffer []byte) float64 {
	zeroCrossings := 0
	totalSamples := len(buffer) / bytesPerSample

	for i := 0; i < totalSamples-1; i++ {
		currentSample := int16(binary.LittleEndian.Uint16(buffer[i*bytesPerSample : (i+1)*bytesPerSample]))
		nextSample := int16(binary.LittleEndian.Uint16(buffer[(i+1)*bytesPerSample : (i+2)*bytesPerSample]))

		if (currentSample >= 0 && nextSample < 0) || (currentSample < 0 && nextSample >= 0) {
			zeroCrossings++
		}
	}

	return float64(zeroCrossings) / float64(totalSamples)
}
