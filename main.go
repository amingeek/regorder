package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/gordonklaus/portaudio"
)

const (
	sampleRate = 44100
	channels   = 1
)

func main() {
	if err := portaudio.Initialize(); err != nil {
		log.Fatal(err)
	}
	defer portaudio.Terminate()

	fmt.Println("Press ENTER to start recording...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')

	fmt.Println("🎤 Recording... (press ENTER to stop)")

	input := make([]int16, 0)
	stream, err := portaudio.OpenDefaultStream(channels, 0, sampleRate, 0, func(in []int16) {
		input = append(input, in...)
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		log.Fatal(err)
	}

	bufio.NewReader(os.Stdin).ReadBytes('\n')
	if err := stream.Stop(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("🛑 Recording stopped. File saved as: output.wav")

	// تبدیل []int16 به []int
	data := make([]int, len(input))
	for i, v := range input {
		data[i] = int(v)
	}

	buf := &audio.IntBuffer{
		Data:           data,
		Format:         &audio.Format{SampleRate: sampleRate, NumChannels: channels},
		SourceBitDepth: 16,
	}

	f, err := os.Create("output.wav")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	encoder := wav.NewEncoder(f, sampleRate, 16, channels, 1)
	if err := encoder.Write(buf); err != nil {
		log.Fatal(err)
	}
	encoder.Close()

	fmt.Println("🎶 Converting to MP3...")

	cmd := exec.Command("ffmpeg", "-y", "-i", "output.wav", "output.mp3")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal("ffmpeg conversion error:", err)
	}

	fmt.Println("📝 Transcribing audio with Whisper...")

	// مسیر درست whisper-cli در build/bin
	whisperCmd := "./whisper.cpp/build/bin/whisper-cli"
	cmd = exec.Command(whisperCmd,
		"-f", "output.mp3",
		"-m", "./whisper.cpp/models/ggml-medium.bin",
		"-l", "auto",
		"--output-txt",
		"--no-prints",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal("transcribe error:", err)
	}

	fmt.Println("✅ Transcription saved in: text.txt")
}
