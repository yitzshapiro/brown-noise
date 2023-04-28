# Brown Noise Generator

High Performance / Low Footprint Brown Noise Generator in Golang.

Brown noise, also known as Brownian noise or red noise, is a type of noise signal that has a power spectral density inversely proportional to the square of the frequency. This creates a noise signal with a deeper sound compared to white or pink noise, making it ideal for various applications such as sleep, relaxation, and concentration.

## Features

- Generates brown noise in real-time
- Utilizes the [Oto](https://github.com/hajimehoshi/oto) library for cross-platform audio playback
- Efficient noise generation algorithm
- Can be toggled on and off using a simple command or script

## Installation

1. Clone the repository:

```bash
git clone https://github.com/ulsc/brown-noise.git
```

2. Change to the cloned directory:

```bash
cd brown-noise
```

4. Build the Go application:

```bash
go build -o brown_noise main.go
```

This will generate an executable file called `brown_noise`.

## Usage

### Basic usage

To start the brown noise generator, simply run the `brown_noise` executable:

```bash
./brown_noise
```

Press `Ctrl + C` (`Command + C` on macOS) to stop the generator.

### Background execution

You can run the brown noise generator in the background by using the `nohup` command:

```bash
nohup ./brown_noise &
```

To stop the background process, find its process ID (PID) and use the `kill` command:

```bash
pgrep -f "./brown_noise" | xargs kill
```

### Toggle script

Create a script named `toggle_noise.sh` to easily toggle the brown noise generator on and off:

```bash
#!/bin/zsh

pid=$(pgrep -f "./brown_noise")

if [ -z "$pid" ]; then
  nohup ./brown_noise > /dev/null 2>&1 &
  echo "Brown noise started."
else
  kill $pid
  echo "Brown noise stopped."
fi
```

Make the script executable:

```bash
chmod +x toggle_noise.sh
```

Now, you can run `./toggle_noise.sh` to start or stop the brown noise generator based on its current state.

## Implementation Details

The brown noise generator uses the following key components:

- A custom `rand.Rand` instance seeded with the current Unix time in nanoseconds to ensure unique random sequences for each run.
- The Oto library for cross-platform audio playback.
- A low-pass filter implemented using an exponential moving average algorithm.

The generator creates a buffer containing audio samples with a specified sample rate, number of channels, and bit depth. It generates random samples in the range of -1 to 1, applies a low-pass filter using the exponential moving average algorithm, and writes the filtered samples to the buffer for playback.

The low-pass filter has a tunable parameter `alpha`, which determines the depth of the noise. Lower values of `alpha` result in deeper noise. By adjusting this parameter, you can fine-tune the generated noise to suit your preferences or specific use cases.

The generator continuously creates and plays back buffers of brown noise, ensuring seamless audio playback.

## Performance

The brown noise generator has been designed to minimize CPU and memory usage while providing high-quality noise generation. By generating noise in real-time and using efficient algorithms, the application can run on a wide variety of systems without causing performance issues.

Compared to playing a pre-recorded brown noise MP3 or WAV file, the real-time generation approach used in this application offers the following benefits:

- Infinite, non-repeating noise generation
- No need for large audio files or continuous looping
- Customizable noise depth through the `alpha` parameter

### Benchmarking

The performance of the brown noise generator was measured using Go's built-in benchmarking functionality on a machine with the following specifications:

- OS: macOS (darwin)
- Architecture: amd64
- CPU: Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz

The benchmark results for the `generateBrownNoise` function are:

```bash
BenchmarkGenerateBrownNoise-16 2323 443095 ns/op
BenchmarkFullLoop-16 1 2042681545 ns/op
```

For the `generateBrownNoise` function, it takes approximately 443,095 nanoseconds (around 0.443 milliseconds) to generate a single buffer of brown noise.

The full loop, including Oto library functions, takes approximately 2,042,681,545 nanoseconds (around 2.043 seconds) per iteration.

Keep in mind that performance may vary depending on the hardware and system load.

It's recommended to run the benchmark multiple times and in different conditions to obtain a more accurate and consistent assessment of the performance.

## Contributing

Contributions are welcome! If you have suggestions for improvements, bug reports, or new features, please create an issue or submit a pull request on GitHub.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
