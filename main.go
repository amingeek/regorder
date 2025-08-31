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

	waitForEnter("Press ENTER to start recording...")

	wavFile := "output.wav"
	mp3File := "output.mp3"

	if err := recordWav(wavFile); err != nil {
		log.Fatal("record error:", err)
	}

	if err := convertToMp3(wavFile, mp3File); err != nil {
		log.Fatal("convert error:", err)
	}

	fmt.Println("File", mp3File, "is ready.")
}

func recordWav(filename string) error {
	in := make([]int16, 64)

	stream, err := portaudio.OpenDefaultStream(1, 0, float64(sampleRate), len(in), in)
	if err != nil {
		return err
	}
	defer stream.Close()

	wavFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer wavFile.Close()

	enc := wav.NewEncoder(wavFile, sampleRate, 16, channels, 1)
	defer enc.Close()

	fmt.Println("Recording... (press ENTER to stop)")
	if err := stream.Start(); err != nil {
		return err
	}

	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				if err := stream.Read(); err != nil {
					log.Println("read error:", err)
					return
				}

				intBuf := make([]int, len(in))
				for i, v := range in {
					intBuf[i] = int(v)
				}

				if err := enc.Write(&audio.IntBuffer{
					Data:   intBuf,
					Format: &audio.Format{SampleRate: sampleRate, NumChannels: channels},
				}); err != nil {
					log.Println("encode error:", err)
					return
				}
			}
		}
	}()

	waitForEnter("")
	close(stop)

	if err := stream.Stop(); err != nil {
		log.Println("stop error:", err)
	}
	fmt.Println("Recording stopped. File saved as:", filename)

	return nil
}

func convertToMp3(wavFile, mp3File string) error {
	fmt.Println("Converting to MP3...")
	cmd := exec.Command("ffmpeg", "-y", "-i", wavFile, mp3File)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func waitForEnter(msg string) {
	if msg != "" {
		fmt.Println(msg)
	}
	_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
}
