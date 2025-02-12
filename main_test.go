package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/hajimehoshi/oto"
)

// -----------------------------------------------------------------------------
// TestMain for optional CPU profiling
// -----------------------------------------------------------------------------

func TestMain(m *testing.M) {
	// OPTIONAL: Enable CPU profiling
	// Comment out if you don't want an always-on CPU profiling.
	// To run with CPU profiling: go test -bench=. -run=^$ -cpuprofile=cpu.prof
	// and then analyze with: go tool pprof cpu.prof
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	pprof.StartCPUProfile(f)
	code := m.Run()
	pprof.StopCPUProfile()
	os.Exit(code)
}

// -----------------------------------------------------------------------------
// Test: Sub-tests for multiple alpha values
// -----------------------------------------------------------------------------

func TestGenerateBrownNoiseAlphaValues(t *testing.T) {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))

	// Build a buffer sized for bufferDuration
	framesPerBuffer := int(float64(sampleRate) * bufferDuration.Seconds())
	bufferSizeInBytes := framesPerBuffer * channelNum * bitDepthInBytes
	buffer := make([]byte, bufferSizeInBytes)

	alphas := []float64{0.005, 0.01, 0.05, 0.1}

	for _, alpha := range alphas {
		t.Run(fmt.Sprintf("alpha=%.3f", alpha), func(t *testing.T) {
			generateBrownNoise(r, buffer, alpha)

			// Calculate average, stddev, min/max, zero-crossing
			avg, stddev := calculateAverageAndStdDev(buffer)
			minSample, maxSample := calculateMinMax(buffer)
			zeroCrossingRate := calculateZeroCrossingRate(buffer)

			// Check if results are within expected ranges (example thresholds)
			if avg < -1500 || avg > 1500 {
				t.Errorf("Unexpected average: %f", avg)
			}
			if stddev < 500 || stddev > 5000 {
				t.Errorf("Unexpected std dev: %f", stddev)
			}
			if minSample < -32767 || maxSample > 32767 {
				t.Errorf("Min/Max out of range: %f, %f", minSample, maxSample)
			}
			if zeroCrossingRate < 0.001 || zeroCrossingRate > 0.2 {
				t.Errorf("Zero-crossing rate out of range: %f", zeroCrossingRate)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Test: Range check (basic int16 limit check)
// -----------------------------------------------------------------------------

func TestGenerateBrownNoiseOutputRange(t *testing.T) {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))

	framesPerBuffer := int(float64(sampleRate) * bufferDuration.Seconds())
	bufferSizeInBytes := framesPerBuffer * channelNum * bitDepthInBytes
	buffer := make([]byte, bufferSizeInBytes)

	generateBrownNoise(r, buffer, 0.01)

	bytesPerSample := channelNum * bitDepthInBytes
	for i := 0; i < len(buffer); i += bytesPerSample {
		sample := int16(binary.LittleEndian.Uint16(buffer[i : i+2]))
		if sample < -32768 || sample > 32767 {
			t.Errorf("Sample out of range: %d", sample)
		}
	}
}

// -----------------------------------------------------------------------------
// Test: Oto context/player creation
// -----------------------------------------------------------------------------

func TestOtoContextAndPlayer(t *testing.T) {
	framesPerBuffer := int(float64(sampleRate) * bufferDuration.Seconds())
	bufferSizeInBytes := framesPerBuffer * channelNum * bitDepthInBytes

	context, err := oto.NewContext(sampleRate, channelNum, bitDepthInBytes, bufferSizeInBytes)
	if err != nil {
		t.Errorf("Failed to create Oto context: %v", err)
		return
	}
	defer context.Close()

	player := context.NewPlayer()
	defer player.Close()

	if player == nil {
		t.Error("Failed to create Oto player")
	}
}

// -----------------------------------------------------------------------------
// Test: Continuity check (ensures last sample in buffer #1 is close
// to first sample in buffer #2, if we preserve lastSample globally)
// -----------------------------------------------------------------------------

func TestBrownNoiseContinuity(t *testing.T) {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))

	framesPerBuffer := int(float64(sampleRate) * bufferDuration.Seconds())
	bufferSizeInBytes := framesPerBuffer * channelNum * bitDepthInBytes
	buf1 := make([]byte, bufferSizeInBytes)
	buf2 := make([]byte, bufferSizeInBytes)

	alpha := 0.01

	// Generate first buffer
	generateBrownNoise(r, buf1, alpha)
	// Grab the last sample from buf1 (left channel)
	bytesPerSample := channelNum * bitDepthInBytes
	lastSampleBuf1 := int16(binary.LittleEndian.Uint16(
		buf1[len(buf1)-bytesPerSample : len(buf1)-bytesPerSample+2],
	))

	// Generate second buffer
	generateBrownNoise(r, buf2, alpha)
	firstSampleBuf2 := int16(binary.LittleEndian.Uint16(buf2[0:2]))

	// They won't be identical, but if the noise is continuous, they should be close.
	diff := math.Abs(float64(lastSampleBuf1 - firstSampleBuf2))
	if diff > 500 { // pick a tolerance that makes sense for alpha=0.01
		t.Errorf("Continuity check failed: last of buf1 = %d, first of buf2 = %d, diff=%f",
			lastSampleBuf1, firstSampleBuf2, diff)
	}
}

// -----------------------------------------------------------------------------
// Test: Check for no excessive DC drift over multiple buffers
// -----------------------------------------------------------------------------

func TestNoExcessiveDCDrift(t *testing.T) {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))

	framesPerBuffer := int(float64(sampleRate) * bufferDuration.Seconds())
	bufferSizeInBytes := framesPerBuffer * channelNum * bitDepthInBytes
	buffer := make([]byte, bufferSizeInBytes)

	alpha := 0.01
	totalSamples := 0
	sum := 0.0

	const numBuffers = 10
	for i := 0; i < numBuffers; i++ {
		generateBrownNoise(r, buffer, alpha)
		bSum, _ := sumAndSqSum(buffer)
		sum += bSum
		totalSamples += len(buffer) / (channelNum * bitDepthInBytes)
	}

	avg := sum / float64(totalSamples)
	if math.Abs(avg) > 2000 {
		t.Errorf("Excessive DC drift: average = %f", avg)
	}
}

// -----------------------------------------------------------------------------
// Test: Concurrency (optional) - multiple goroutines generating noise
// -----------------------------------------------------------------------------

func TestConcurrentGeneration(t *testing.T) {
	const goroutines = 4
	const iterations = 5

	seed := time.Now().UnixNano()

	framesPerBuffer := int(float64(sampleRate) * bufferDuration.Seconds())
	bufferSizeInBytes := framesPerBuffer * channelNum * bitDepthInBytes

	errCh := make(chan error, goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			r := rand.New(rand.NewSource(seed + int64(id)))
			buf := make([]byte, bufferSizeInBytes)

			for i := 0; i < iterations; i++ {
				generateBrownNoise(r, buf, 0.01)

				bytesPerSample := channelNum * bitDepthInBytes
				for i := 0; i < len(buf); i += bytesPerSample {
					sample := int16(binary.LittleEndian.Uint16(buf[i : i+2]))
					if sample < -32768 || sample > 32767 {
						errCh <- fmt.Errorf("goroutine %d: sample out of range: %d", id, sample)
						return
					}
				}
			}
			errCh <- nil
		}(g)
	}

	for i := 0; i < goroutines; i++ {
		if err := <-errCh; err != nil {
			t.Error(err)
		}
	}
}

// -----------------------------------------------------------------------------
// Benchmarks
// -----------------------------------------------------------------------------

func BenchmarkGenerateBrownNoise(b *testing.B) {
	b.ReportAllocs()

	seed := time.Now().UnixNano()
	privateRand := rand.New(rand.NewSource(seed))

	alpha := 0.01
	framesPerBuffer := int(float64(sampleRate) * bufferDuration.Seconds())
	bufferSizeInBytes := framesPerBuffer * channelNum * bitDepthInBytes
	buffer := make([]byte, bufferSizeInBytes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateBrownNoise(privateRand, buffer, alpha)
	}
}

func BenchmarkFullLoop(b *testing.B) {
	b.ReportAllocs()

	seed := time.Now().UnixNano()
	privateRand := rand.New(rand.NewSource(seed))
	framesPerBuffer := int(float64(sampleRate) * bufferDuration.Seconds())
	bufferSizeInBytes := framesPerBuffer * channelNum * bitDepthInBytes

	context, err := oto.NewContext(sampleRate, channelNum, bitDepthInBytes, bufferSizeInBytes)
	if err != nil {
		b.Fatal(err)
	}
	defer context.Close()

	player := context.NewPlayer()
	defer player.Close()

	buffer := make([]byte, bufferSizeInBytes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateBrownNoise(privateRand, buffer, 0.01)
		_, err := player.Write(buffer)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// -----------------------------------------------------------------------------
// Helper functions
// -----------------------------------------------------------------------------

// calculateAverageAndStdDev computes the mean and stddev of the left-channel samples
func calculateAverageAndStdDev(buffer []byte) (float64, float64) {
	sum := 0.0
	sqSum := 0.0
	count := 0
	bytesPerSample := channelNum * bitDepthInBytes

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

// calculateMinMax finds min and max among all left-channel samples
func calculateMinMax(buffer []byte) (float64, float64) {
	minSample := math.MaxFloat64
	maxSample := -math.MaxFloat64
	bytesPerSample := channelNum * bitDepthInBytes

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

// calculateZeroCrossingRate returns fraction of samples that cross from + to - or vice versa
func calculateZeroCrossingRate(buffer []byte) float64 {
	zeroCrossings := 0
	bytesPerSample := channelNum * bitDepthInBytes
	totalSamples := len(buffer) / bytesPerSample

	for i := 0; i < totalSamples-1; i++ {
		currentSample := int16(binary.LittleEndian.Uint16(buffer[i*bytesPerSample : i*bytesPerSample+2]))
		nextSample := int16(binary.LittleEndian.Uint16(buffer[(i+1)*bytesPerSample : (i+1)*bytesPerSample+2]))

		if (currentSample >= 0 && nextSample < 0) || (currentSample < 0 && nextSample >= 0) {
			zeroCrossings++
		}
	}
	return float64(zeroCrossings) / float64(totalSamples)
}

// sumAndSqSum returns the sum of samples (left channel) and sum of squares
func sumAndSqSum(buffer []byte) (float64, float64) {
	s := 0.0
	sq := 0.0
	bytesPerSample := channelNum * bitDepthInBytes
	for i := 0; i < len(buffer); i += bytesPerSample {
		sample := float64(int16(binary.LittleEndian.Uint16(buffer[i : i+2])))
		s += sample
		sq += sample * sample
	}
	return s, sq
}
