package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"compnet-socket-labs/project/utils"

	"github.com/quic-go/quic-go"
)

var (
	DefaultServerIP   = "0.0.0.0"
	DefaultServerPort = "7380"
	ServerType        = "udp4"
	BufferSize        = 2048
	AppLayerProto     = "lrt-jakarta-2406347380"
)

func ResolveConfig() (string, string) {
	ip := os.Getenv("SERVER_ADDR")
	if ip == "" {
		ip = DefaultServerIP
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultServerPort
	}
	return ip, port
}

func ResolveALPN() string {
	alpn := os.Getenv("ALPN")
	if alpn == "" {
		alpn = AppLayerProto
	}
	return alpn
}

func Handler(packet utils.LRTJPIDSPacket) string {
	timeFormatted := fmt.Sprintf("%02d:%02d", packet.EstimatedArrivalHour, packet.EstimatedArrivalMinute)

	if packet.IsNewTrain != 0 {
		return fmt.Sprintf("ADD | %d | %s | %s", packet.TrainNumber, packet.Destination, timeFormatted)
	} else if packet.IsUpdateTrain != 0 {
		return fmt.Sprintf("UPD | %d | %s | %s", packet.TrainNumber, packet.Destination, timeFormatted)
	}
	return ""
}

func main() {
	serverIP, serverPort := ResolveConfig()
	alpn := ResolveALPN()

	localUDPAddress, err := net.ResolveUDPAddr("udp4", net.JoinHostPort(serverIP, serverPort))
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}

	socket, err := net.ListenUDP("udp4", localUDPAddress)
	if err != nil {
		log.Fatalf("Failed to listen on UDP: %v", err)
	}
	defer socket.Close()

	tlsConfig := &tls.Config{
		Certificates: utils.GenerateTLSSelfSignedCertificates(),
		NextProtos:   []string{alpn},
	}

	listener, err := quic.Listen(socket, tlsConfig, &quic.Config{})
	if err != nil {
		log.Fatalf("Failed to start QUIC listener: %v", err)
	}
	defer listener.Close()

	fmt.Printf("[Subscriber] Listening on %s:%s (ALPN: %s)\n", serverIP, serverPort, alpn)

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			continue
		}
		go func() {
			for {
				stream, err := conn.AcceptStream(context.Background())
				if err != nil {
					return
				}
				go func() {
					defer stream.Close()

					buf := make([]byte, BufferSize)
					n, err := stream.Read(buf)
					if err != nil && err != io.EOF {
						return
					}

					packet := utils.Decoder(buf[:n])

					out := Handler(packet)
					if out != "" {
						fmt.Println(out)
					}

					packet.IsAck = 1
					ackBytes := utils.Encoder(packet)
					_, _ = stream.Write(ackBytes)
				}()
			}
		}()
	}
}
