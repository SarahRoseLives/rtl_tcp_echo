package main

import (
	"io"
	"log"
	"net"
	"os"
	"time"
)

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

	// Send minimal rtl_tcp server startup banner
	// (This is just for compatibility. Real rtl_tcp sends a header.)
	banner := make([]byte, 12)
	clientConn.Write(banner)

	buf := make([]byte, 32*1024)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			_, err2 := clientConn.Write(buf[:n])
			if err2 != nil {
				log.Printf("Client write error: %v", err2)
				break
			}
			// Sleep to simulate real IQ rate (adjust as needed)
			time.Sleep(20 * time.Millisecond)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Playback read error: %v", err)
			break
		}
	}
}