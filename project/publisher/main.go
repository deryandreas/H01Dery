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
	DefaultServerIP   = "127.0.0.1"
	DefaultServerPort = "7380"
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

func main() {
	serverIP, serverPort := ResolveConfig()
	alpn := ResolveALPN()

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{alpn},
	}

	conn, err := quic.DialAddr(context.Background(), net.JoinHostPort(serverIP, serverPort), tlsConfig, &quic.Config{})
	if err != nil {
		log.Fatalf("Failed to connect via QUIC: %v", err)
	}
	defer conn.CloseWithError(0x0, "No Error")

	packetA := utils.LRTJPIDSPacket{
		LRTJPIDSPacketFixed: utils.LRTJPIDSPacketFixed{
			TransactionId:          1,
			IsNewTrain:             1,
			TrainNumber:            1002,
			EstimatedArrivalHour:   5,
			EstimatedArrivalMinute: 34,
			DestinationLength:      len("Manggarai"),
		},
		Destination: "Manggarai",
	}

	packetB := utils.LRTJPIDSPacket{
		LRTJPIDSPacketFixed: utils.LRTJPIDSPacketFixed{
			TransactionId:          2,
			IsUpdateTrain:          1,
			TrainNumber:            1002,
			EstimatedArrivalHour:   5,
			EstimatedArrivalMinute: 35,
			DestinationLength:      len("Velodrome"),
		},
		Destination: "Velodrome",
	}

	packets := []utils.LRTJPIDSPacket{packetA, packetB}

	for _, packet := range packets {
		stream, err := conn.OpenStreamSync(context.Background())
		if err != nil {
			log.Fatalf("Failed to open stream: %v", err)
		}

		data := utils.Encoder(packet)
		_, err = stream.Write(data)
		if err != nil {
			log.Fatalf("Failed to write to stream: %v", err)
		}

		buf := make([]byte, BufferSize)
		n, err := stream.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatalf("Failed to read from stream: %v", err)
		}

		ackPacket := utils.Decoder(buf[:n])
		if ackPacket.IsAck != 0 {
			fmt.Printf("[Publisher] Received ACK for TxID: %d\n", ackPacket.TransactionId)
		}
		_ = stream.Close()
	}
}
