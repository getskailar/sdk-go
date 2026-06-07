// Command speech synthesizes text to an MP3 file.
//
//	SKAILAR_API_KEY=skl_live_... go run ./examples/speech
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	skailar "github.com/getskailar/sdk-go"
)

func main() {
	client, err := skailar.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	rc, err := client.Audio.Speech.Create(context.Background(), skailar.SpeechRequest{
		Input: "Hello from Skailar.",
		Voice: skailar.VoiceNova,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rc.Close()

	out, err := os.Create("speech.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	n, err := io.Copy(out, rc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %d bytes to speech.mp3\n", n)
}
