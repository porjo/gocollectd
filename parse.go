package gocollectd

import (
	"bytes"
	"encoding/binary"
	"errors"
)

var ErrorUnsupported = errors.New("Unsupported collectd packet recieved")
var ErrorInvalid = errors.New("Invalid collectd packet recieved")

func Parse(b []byte) (*[]Packet, error) {
	r := make([]Packet, 0)

	buf := bytes.NewBuffer(b)
	var p Packet
	var packetHeader struct {
		PartType   uint16
		PartLength uint16
	}
	var time uint64
	var err error
	var valueCount uint16

	for buf.Len() > 0 {
		err = binary.Read(buf, binary.BigEndian, &packetHeader)
		if err != nil {
			return nil, err
		}
		if packetHeader.PartLength < 5 {
			return nil, ErrorInvalid
		}

		partBytes := buf.Next(int(packetHeader.PartLength) - 4)
		if len(partBytes) < int(packetHeader.PartLength)-4 {
			return nil, ErrorInvalid
		}
		partBuffer := bytes.NewBuffer(partBytes)

		switch packetHeader.PartType {
		case 0:
			str := partBuffer.String()
			p.Hostname = str[0 : len(str)-1]
		case 1:
			err = binary.Read(partBuffer, binary.BigEndian, &time)
			if err != nil {
				return nil, err
			}
			p.CdTime = time << 30
		case 2:
			str := partBuffer.String()
			p.Plugin = str[0 : len(str)-1]
		case 3:
			str := partBuffer.String()
			p.PluginInstance = str[0 : len(str)-1]
		case 4:
			str := partBuffer.String()
			p.Type = str[0 : len(str)-1]
		case 5:
			str := partBuffer.String()
			p.TypeInstance = str[0 : len(str)-1]
		case 6:
			err = binary.Read(partBuffer, binary.BigEndian, &valueCount)
			if err != nil {
				return nil, err
			}

			// make a copy so we lose reference to the underlying slice data
			p.Bytes = make([]byte, 8*valueCount, 8*valueCount)
			// collectd's protocol puts data in a seemingly weird
			// order which appears to be exactly what we want.
			copy(p.Bytes, partBytes[2+valueCount:2+valueCount+(valueCount*8)])
			// todo: what if some data is missing?

			p.DataTypes = make([]uint8, valueCount, valueCount) // holds a copy so we lose reference to the underlying slice data
			for i := range p.DataTypes {
				err = binary.Read(partBuffer, binary.BigEndian, &p.DataTypes[i])
				if err != nil {
					return nil, err
				}
			}

			r = append(r, p)
		case 7:
			// interval, ignore
		case 8:
			// high res time
			err = binary.Read(partBuffer, binary.BigEndian, &p.CdTime)
			if err != nil {
				return nil, err
			}
		case 9:
			// interval, ignore
		case 0x100:
			// message (notifications), ignore
		case 0x101:
			// severity, ignore
		case 0x200:
			// Signature (HMAC-SHA-256), todo
			return nil, ErrorUnsupported
		case 0x210:
			// Encryption (AES-256/OFB/SHA-1), todo
			return nil, ErrorUnsupported
		default:
			return nil, ErrorUnsupported
		}
	}
	return &r, nil
}
