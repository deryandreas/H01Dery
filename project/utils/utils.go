package utils

import (
	"bytes"
	"encoding/binary"
)

type LRTJPIDSPacketFixed struct {
	TransactionId          int
	IsAck                  int
	IsNewTrain             int
	IsUpdateTrain          int
	IsDeleteTrain          int
	IsTrainArriving        int
	IsTrainDeparting       int
	TrainNumber            int
	EstimatedArrivalHour   int
	EstimatedArrivalMinute int
	DestinationLength      int
}

type LRTJPIDSPacket struct {
	LRTJPIDSPacketFixed
	Destination string
}

func Encoder(packet LRTJPIDSPacket) []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.BigEndian, uint16(packet.TransactionId))

	var flags uint8 = 0
	if packet.IsAck != 0 {
		flags |= (1 << 7)
	}
	if packet.IsNewTrain != 0 {
		flags |= (1 << 6)
	}
	if packet.IsUpdateTrain != 0 {
		flags |= (1 << 5)
	}
	if packet.IsDeleteTrain != 0 {
		flags |= (1 << 4)
	}
	if packet.IsTrainArriving != 0 {
		flags |= (1 << 3)
	}
	if packet.IsTrainDeparting != 0 {
		flags |= (1 << 2)
	}
	buf.WriteByte(flags)

	_ = binary.Write(buf, binary.BigEndian, uint16(packet.TrainNumber))

	buf.WriteByte(uint8(packet.EstimatedArrivalHour))

	buf.WriteByte(uint8(packet.EstimatedArrivalMinute))

	destBytes := []byte(packet.Destination)
	buf.WriteByte(uint8(len(destBytes)))
	buf.Write(destBytes)

	return buf.Bytes()
}

func Decoder(rawMessage []byte) LRTJPIDSPacket {
	var packet LRTJPIDSPacket
	if len(rawMessage) < 8 {
		return packet
	}

	reader := bytes.NewReader(rawMessage)

	var txId uint16
	_ = binary.Read(reader, binary.BigEndian, &txId)
	packet.TransactionId = int(txId)

	flags, _ := reader.ReadByte()
	if (flags & (1 << 7)) != 0 {
		packet.IsAck = 1
	}
	if (flags & (1 << 6)) != 0 {
		packet.IsNewTrain = 1
	}
	if (flags & (1 << 5)) != 0 {
		packet.IsUpdateTrain = 1
	}
	if (flags & (1 << 4)) != 0 {
		packet.IsDeleteTrain = 1
	}
	if (flags & (1 << 3)) != 0 {
		packet.IsTrainArriving = 1
	}
	if (flags & (1 << 2)) != 0 {
		packet.IsTrainDeparting = 1
	}

	var trainNum uint16
	_ = binary.Read(reader, binary.BigEndian, &trainNum)
	packet.TrainNumber = int(trainNum)

	arrHour, _ := reader.ReadByte()
	packet.EstimatedArrivalHour = int(arrHour)

	arrMin, _ := reader.ReadByte()
	packet.EstimatedArrivalMinute = int(arrMin)

	destLen, _ := reader.ReadByte()
	packet.DestinationLength = int(destLen)

	if destLen > 0 && reader.Len() >= int(destLen) {
		destBytes := make([]byte, destLen)
		_, _ = reader.Read(destBytes)
		packet.Destination = string(destBytes)
	}

	return packet
}