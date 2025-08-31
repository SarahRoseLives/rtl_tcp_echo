package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	var (
		mode      string
		listen    string
		forward   string
		record    string
		playback  string
	)

	flag.StringVar(&mode, "mode", "proxy", "Mode: proxy or playback")
	flag.StringVar(&listen, "listen", "0.0.0.0:1234", "Listen address (proxy or playback)")
	flag.StringVar(&forward, "forward", "127.0.0.1:1234", "Forward address (proxy mode only)")
	flag.StringVar(&record, "record", "iq_recording.bin", "IQ recording file (proxy mode only)")
	flag.StringVar(&playback, "playback", "iq_recording.bin", "Playback IQ file (playback mode only)")
	flag.Parse()

	switch mode {
	case "proxy":
		log.Printf("Starting RTL_TCP_ECHO in proxy mode")
		log.Printf("Listening on %s, forwarding to %s, recording to %s", listen, forward, record)
		runProxy(listen, forward, record)
	case "playback":
		log.Printf("Starting RTL_TCP_ECHO in playback mode")
		log.Printf("Serving IQ data from %s on %s", playback, listen)
		runPlayback(listen, playback)
	default:
		fmt.Println("Unknown mode:", mode)
		flag.Usage()
		os.Exit(1)
	}
}