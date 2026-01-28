package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
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
		// Enable TCP_NODELAY for low-latency streaming (important for DMR timing)
		if tcpConn, ok := clientConn.(*net.TCPConn); ok {
			_ = tcpConn.SetNoDelay(true)
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
	var sampleRate uint32 = 1536000 // Default for DSD-Neo DMR
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

	// Consume client commands in background to prevent blocking
	// NOTE: We log but ignore sample rate changes since the recording was made at a fixed rate
	go func() {
		cmd := make([]byte, 5)
		for {
			_, err := io.ReadFull(clientConn, cmd)
			if err != nil {
				return
			}
			// Log sample rate command (0x02) but don't change playback rate
			// The recording was made at a fixed rate - changing it would corrupt timing
			if cmd[0] == 0x02 {
				requestedRate := binary.BigEndian.Uint32(cmd[1:5])
				if requestedRate != sampleRate {
					log.Printf("Client requested sample rate %d Hz (ignoring - playback locked at %d Hz)", requestedRate, sampleRate)
				}
			}
		}
	}()

	// Play back IQ data with original timing
	// File format: [8-byte timestamp ns][4-byte length][data] repeating
	startTime := time.Now()
	var lastTimestamp int64 = 0
	
	for {
		// Read timestamp (8 bytes)
		tsBytes := make([]byte, 8)
		_, err := io.ReadFull(f, tsBytes)
		if err == io.EOF {
			log.Printf("Playback complete")
			break
		}
		if err != nil {
			log.Printf("Timestamp read error: %v", err)
			break
		}
		timestamp := int64(binary.LittleEndian.Uint64(tsBytes))
		
		// Read data length (4 bytes)
		lenBytes := make([]byte, 4)
		_, err = io.ReadFull(f, lenBytes)
		if err != nil {
			log.Printf("Length read error: %v", err)
			break
		}
		dataLen := binary.LittleEndian.Uint32(lenBytes)
		
		// Read IQ data
		data := make([]byte, dataLen)
		_, err = io.ReadFull(f, data)
		if err != nil {
			log.Printf("Data read error: %v", err)
			break
		}
		
		// Wait until the correct time to send this chunk
		targetTime := time.Duration(timestamp)
		elapsed := time.Since(startTime)
		if targetTime > elapsed {
			time.Sleep(targetTime - elapsed)
		}
		
		// Send data to client
		_, err = clientConn.Write(data)
		if err != nil {
			log.Printf("Client write error: %v", err)
			break
		}
		
		lastTimestamp = timestamp
	}
	
	log.Printf("Played back data over %.3f seconds", float64(lastTimestamp)/1e9)
}