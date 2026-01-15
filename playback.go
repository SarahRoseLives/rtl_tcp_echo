package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"
)

// playbackSampleRate is set from command-line flag
var playbackSampleRate uint32

// runPlayback acts as a fake rtl_tcp server, serving IQ data from a file.
func runPlayback(listenAddr, playbackFile string) {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()

	for {
		clientConn, err := ln.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go handlePlaybackConn(clientConn, playbackFile)
	}
}

func handlePlaybackConn(clientConn net.Conn, playbackFile string) {
	defer clientConn.Close()

	f, err := os.Open(playbackFile)
	if err != nil {
		log.Printf("Playback file open error: %v", err)
		return
	}
	defer f.Close()

	// Default sample rate (can be overridden via -samplerate flag)
	var sampleRate uint32 = 2400000
	if playbackSampleRate > 0 {
		sampleRate = playbackSampleRate
	}

	// Build proper rtl_tcp header
	// Format: "RTL0" magic + tuner type (4 bytes) + tuner gain count (4 bytes)
	header := make([]byte, 12)
	copy(header[0:4], []byte("RTL0"))
	binary.BigEndian.PutUint32(header[4:8], 5)   // Tuner type: R820T
	binary.BigEndian.PutUint32(header[8:12], 29) // Gain count
	_, err = clientConn.Write(header)
	if err != nil {
		log.Printf("Header write error: %v", err)
		return
	}

	// Atomic sample rate for command handler to update
	var currentRate atomic.Uint32
	currentRate.Store(sampleRate)

	// Consume client commands in background to prevent blocking
	go func() {
		cmd := make([]byte, 5)
		for {
			_, err := io.ReadFull(clientConn, cmd)
			if err != nil {
				return
			}
			// Parse sample rate command (0x02)
			if cmd[0] == 0x02 {
				newRate := binary.BigEndian.Uint32(cmd[1:5])
				if newRate > 0 {
					currentRate.Store(newRate)
					log.Printf("Client requested sample rate: %d Hz", newRate)
				}
			}
		}
	}()

	// IQ data is 2 bytes per sample (I + Q as unsigned 8-bit each)
	const bytesPerSample = 2
	buf := make([]byte, 16384) // Smaller buffer for more precise timing

	for {
		n, err := f.Read(buf)
		if n > 0 {
			_, err2 := clientConn.Write(buf[:n])
			if err2 != nil {
				log.Printf("Client write error: %v", err2)
				break
			}

			// Calculate sleep based on sample rate
			// samples = bytes / 2, time = samples / sample_rate
			rate := currentRate.Load()
			samples := float64(n) / float64(bytesPerSample)
			sleepDuration := time.Duration(samples / float64(rate) * float64(time.Second))
			time.Sleep(sleepDuration)
		}
		if err == io.EOF {
			log.Printf("Playback complete")
			break
		}
		if err != nil {
			log.Printf("Playback read error: %v", err)
			break
		}
	}
}