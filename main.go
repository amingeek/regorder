package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/gordonklaus/portaudio"
)

const (
	sampleRate   = 44100
	channels     = 1
	voiceDir     = "voice"
	textDir      = "texts"
	historyFile  = "history.txt"
	whisperPath  = "./whisper.cpp/build/bin/whisper-cli"
	whisperModel = "./whisper.cpp/models/ggml-medium.bin"

	apiKey = "sk-oDDoxgFAPictp5jKcVRXaudTfWSGoK6hu9dnI21fyrkLjgRu"
	apiURL = "https://api.gapapi.com/v1/chat/completions"
)

func main() {
	ensureDir(voiceDir)
	ensureDir(textDir)

	if err := portaudio.Initialize(); err != nil {
		log.Fatal(err)
	}
	defer portaudio.Terminate()

	reader := bufio.NewReader(os.Stdin)
	for {
		nextVoiceNum := getNextNumber(voiceDir)
		nextTextNum := getNextNumber(textDir)

		fmt.Println("\nPress ENTER to start recording (or type 'exit' to quit)...")
		inputStr, _ := reader.ReadString('\n')
		inputStr = strings.TrimSpace(inputStr)
		if inputStr == "exit" {
			fmt.Println("Exiting...")
			break
		}

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
		reader.ReadBytes('\n')
		if err := stream.Stop(); err != nil {
			log.Fatal(err)
		}

		voiceFile := fmt.Sprintf("%s/%d.wav", voiceDir, nextVoiceNum)
		textFile := fmt.Sprintf("%s/%d.txt", textDir, nextTextNum)
		saveWav(input, voiceFile)
		fmt.Println("🛑 Recording stopped. File saved as:", voiceFile)

		fmt.Println("📝 Transcribing audio with Whisper...")
		runCmd(whisperPath,
			"-f", voiceFile,
			"-m", whisperModel,
			"-l", "auto",
			"--output-txt",
			"--no-prints",
		)

		whisperTxt := voiceFile + ".txt"
		textBytes, err := os.ReadFile(whisperTxt)
		if err != nil {
			log.Fatal(err)
		}
		text := strings.TrimSpace(string(textBytes))
		fmt.Println("💬 Whisper text:", text)
		if err := os.WriteFile(textFile, []byte(text), 0644); err != nil {
			log.Fatal(err)
		}

		fmt.Println("🤖 Sending to GAP API...")

		response, err := callGapAPI(text)
		if err != nil {
			log.Println("API error:", err)
			continue
		}

		fmt.Println("💡 LLM Response:\n", response)

		// ✅ ذخیره در history.txt
		saveHistory(text, response)
	}
}

func callGapAPI(prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if len(res.Choices) > 0 {
		return res.Choices[0].Message.Content, nil
	}
	return "(no response)", nil
}

func saveHistory(userMsg, aiMsg string) {
	f, err := os.OpenFile(historyFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Error writing history:", err)
		return
	}
	defer f.Close()

	history := fmt.Sprintf("User: %s\nAI: %s\n\n", userMsg, aiMsg)
	f.WriteString(history)
}

func saveWav(input []int16, path string) {
	data := make([]int, len(input))
	for i, v := range input {
		data[i] = int(v)
	}

	buf := &audio.IntBuffer{
		Data:           data,
		Format:         &audio.Format{SampleRate: sampleRate, NumChannels: channels},
		SourceBitDepth: 16,
	}

	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	encoder := wav.NewEncoder(f, sampleRate, 16, channels, 1)
	if err := encoder.Write(buf); err != nil {
		log.Fatal(err)
	}
	encoder.Close()
}

func runCmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}

func ensureDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}
}

func getNextNumber(dir string) int {
	files := make([]int, 0)
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && (strings.HasSuffix(d.Name(), ".wav") || strings.HasSuffix(d.Name(), ".txt")) {
			base := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			if n, err := strconv.Atoi(base); err == nil {
				files = append(files, n)
			}
		}
		return nil
	})
	if len(files) == 0 {
		return 1
	}
	sort.Ints(files)
	return files[len(files)-1] + 1
}
