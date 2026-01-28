package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"time"
)

// runProxy listens for a client, connects to the real rtl_tcp server, and proxies all data.
// It also records the IQ samples from the server to a file.
func runProxy(listenAddr, forwardAddr, recordFile string) {
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
		go handleProxyConn(clientConn, forwardAddr, recordFile)
	}
}

func handleProxyConn(clientConn net.Conn, forwardAddr, recordFile string) {
	defer clientConn.Close()
	serverConn, err := net.Dial("tcp", forwardAddr)
	if err != nil {
		log.Printf("Forward dial error: %v", err)
		return
	}
	defer serverConn.Close()

	// Open IQ recording file (truncate to start fresh)
	recordF, err := os.OpenFile(recordFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Could not open recording file: %v", err)
		return
	}
	defer recordF.Close()

	done := make(chan struct{})

	// Proxy client->server (commands)
	go func() {
		_, err := io.Copy(serverConn, clientConn)
		if err != nil {
			log.Printf("Client->Server error: %v", err)
		}
		done <- struct{}{}
	}()

	// Proxy server->client (IQ samples), record IQ with timestamps
	go func() {
		buf := make([]byte, 32*1024)
		headerSkipped := false
		startTime := time.Now()
		
		for {
			n, err := serverConn.Read(buf)
			if n > 0 {
				data := buf[:n]
				// Skip the 12-byte rtl_tcp header on first read
				if !headerSkipped {
					if n >= 12 {
						data = buf[12:n]
						headerSkipped = true
					} else {
						// Header split across reads - skip what we have
						headerSkipped = true
						data = nil
					}
				}
				// Write timestamped chunk: [8-byte timestamp ns][4-byte length][data]
				if len(data) > 0 {
					timestamp := time.Since(startTime).Nanoseconds()
					
					// Write timestamp (8 bytes, little-endian)
					tsBytes := make([]byte, 8)
					binary.LittleEndian.PutUint64(tsBytes, uint64(timestamp))
					_, _ = recordF.Write(tsBytes)
					
					// Write data length (4 bytes, little-endian)
					lenBytes := make([]byte, 4)
					binary.LittleEndian.PutUint32(lenBytes, uint32(len(data)))
					_, _ = recordF.Write(lenBytes)
					
					// Write IQ data
					_, _ = recordF.Write(data)
				}
				// Forward original data (with header) to client
				_, err2 := clientConn.Write(buf[:n])
				if err2 != nil {
					log.Printf("Server->Client write error: %v", err2)
					break
				}
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
}