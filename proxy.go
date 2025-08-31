package main

import (
	"io"
	"log"
	"net"
	"os"
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

	// Open IQ recording file
	recordF, err := os.OpenFile(recordFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
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

	// Proxy server->client (IQ samples), record IQ
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := serverConn.Read(buf)
			if n > 0 {
				// Write IQ data to file
				_, _ = recordF.Write(buf[:n])
				// Forward to client
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