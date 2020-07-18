package main

// Functions and data around reading the incoming stream of data from the serial port.

import (
	"bytes"
	"errors"
	"github.com/kb2ma/daghead/internal/log"
	"github.com/kb2ma/daghead/internal/router"
	"github.com/lunixbochs/struc"
	"github.com/mikepb/go-serial"
	"github.com/snksoft/crc"
	"sync"
)

const (
	HDLC_FLAG   byte = 0x7E
	HDLC_ESCAPE byte = 0x7D
	FLOW_ESCAPE byte = 0x12
	FLOW_XON    byte = 0x11
	FLOW_XOFF   byte = 0x13
	FLOW_MASK   byte = 0x10
)

// These values really are constants, but a slice can't be a constant.
var (
	HDLC_FLAG_ARRAY     = []byte{HDLC_FLAG}
	HDLC_FLAG_ESCAPED   = []byte{HDLC_ESCAPE, 0x5E}
	HDLC_ESCAPE_ARRAY   = []byte{HDLC_ESCAPE}
	HDLC_ESCAPE_ESCAPED = []byte{HDLC_ESCAPE, 0x5D}
)

const NOTIFICATION_ERROR int = 0

type IsSync struct {
	IsSync byte
}

// mote_id, component, error_code, arg1, arg2 = struct.unpack('>HBBhH'
type Notification struct {
	MoteId [2]byte
	Component byte
	Code byte
	Arg1 int16
	Arg2 uint16
}

// Handles HDLC escaping
func decodeHdlc(buf []byte) (replBuf []byte, err error) {
	replBuf = bytes.ReplaceAll(buf, HDLC_FLAG_ESCAPED, HDLC_FLAG_ARRAY)
	replBuf = bytes.ReplaceAll(replBuf, HDLC_ESCAPE_ESCAPED, HDLC_ESCAPE_ARRAY)
	crcBuf := replBuf[len(replBuf)-2:]

	hash := crc.CalculateCRC(crc.X25, replBuf[:len(replBuf)-2])
	if (byte(hash & 0xFF) != crcBuf[0]) || (byte((hash & 0xFF00) >> 8) != crcBuf[1]) {
		err = errors.New("CRC decode not valid")
	}
	return
}

// Handles only status type 0, is_sync; example [83 56 66 0 0 61 189]
func readStatusFrame(statusType byte, data []byte) {
	if statusType == 0 {
		buf := bytes.NewBuffer(data)
		o := &IsSync{}
		err := struc.Unpack(buf, o)
		if err != nil {
			log.Printf(log.ERROR, "err? %d\n", err)
		} else {
			log.Printf(log.INFO, "is sync? %d\n", o.IsSync)
		}
	}
}

func readNotificationFrame(notificationLevel int, data []byte) {
	buf := bytes.NewBuffer(data)
	o := &Notification{}
	err := struc.Unpack(buf, o)
	if err != nil {
		log.Printf(log.ERROR, "err? %d\n", err)
	} else {
		log.Printf(log.INFO, "got notification 0x%X: %d, %d\n", o.Code, o.Arg1, o.Arg2)
	}
}

/*
example
00  44 38 42 C3 2D 00 00 00 46 1D 52 44 7B 43 76 78 82 54 7D 13
20  76 65 79 78 F1 83 05 0B 7A 55 3A 82 54 7D 13 76 65 79 78 46
40  1D 52 44 7B 43 76 78 9B 02 E1 08 00 40 00 01 BB BB 00 00 00
60  00 00 00 46 1D 52 44 7B 43 76 78 06 14 00 00 00 AA BB BB 00
80  00 00 00 00 00 46 1D 52 44 7B 43 76 78 B4 D1
*/
func readDataFrame(data []byte) {
	// skip mote ID [:2], asn [2:7], destination [7:15], source [15:23]
	log.Printf(log.INFO, "got data; len total %d, payload %d\n", len(data), len(data)-23)
	if len(data) < 23 {
		return
	}
	//hasHopByHopHeader := false
	ipFields := router.ReadData(data[23], data[24:])

	if ipFields["next_header"] == int(router.IANA_IPv6HOPHEADER) {
		//hasHopByHopHeader := true
		ipFields["next_header"] = ipFields["hop_next_header"]
		// read inner header, expected to be IPHC (RFC 6282)
		//innerFields := router.ReadData(data[23], data[ipFields["payload"]:])
	}
}

/*
Reads incoming byte stream from serial port of root mote. Data formatted as HDLC frames.
Data includes several types of status messages, log notifications, and UDP datagrams.

HDLC frames incoming data with a 0x7E flag bytes to start and end a frame, as
shown below.

   ... 7E 7E xx xx xx xx xx xx 7E 7E ...

Within a frame, an incoming 7E byte is escaped as the sequence 7D 5E. An
incoming 7D byte is escaped as the sequence 7D 5D. Handling for this mechanism
is in decodeHdlc(), after reception of the entire frame.

Note: It seems possible to handle this HDLC escaping inline, but we have implemented
the handling after reception of the entire frame to follow OpenVisualizer for consistency.

Since the serial port uses software based XON/XOFF flow control, XON (0x11) and
XOFF (0x13) data bytes within a frame also must be escaped. The escaped value is sent
XORed with a mask byte (0x10). So, an incoming XON is escaped as the sequence 0x12 0x01,
and XOFF is escaped as the sequence 0x12 0x03. An incoming escape byte (0x12) is
escaped as the sequence 0x12 0x02. Handling for these escaped sequences is inline
in this function.
*/
func readSerial(wg *sync.WaitGroup, port *serial.Port) {
	defer wg.Done()

	frameBuf := make([]byte, 0, 10)
	isInFrame := false
	isEscapingFlow := false

	for true {
		buf := make([]byte, 1)
		_, err := port.Read(buf)
		if err != nil {
			log.Panic(err)
		}

		// At startup, position in stream from device is indeterminate. So must
		// synchronize on first sequence of 0x7E 0x7E framing bytes.
		if buf[0] == HDLC_FLAG {
			if isInFrame && (len(frameBuf) > 0) {
				log.Printf(log.DEBUG, "ending frame, len %d\n", len(frameBuf))
				log.Printf(log.DEBUG, "[% X]\n", frameBuf)

				decoded, err := decodeHdlc(frameBuf)
				if err == nil {
					switch decoded[0] {
					case 'S':
						// decoded[1:3] is the mote ID
						readStatusFrame(decoded[3], decoded[4:])
					case 'E':
						readNotificationFrame(NOTIFICATION_ERROR, decoded[1:])
					case 'D':
						readDataFrame(decoded[1:])
					}
				} else {
					log.Println(log.ERROR, err)
				}
				isInFrame = false
				// defensive; should not be escaping flow if receive HDLC_FLAG
				isEscapingFlow = false
			} else {
				log.Println(log.DEBUG, "starting frame")
				isInFrame = true
				frameBuf = frameBuf[:0]
				isEscapingFlow = false
			}
		} else {
			if isInFrame {
				if buf[0] == FLOW_ESCAPE {
					isEscapingFlow = true
				} else {
					if isEscapingFlow {
						// Expect escaped XON/XOFF/FLOW_ESCAPE byte
						frameBuf = append(frameBuf, buf[0] ^ FLOW_MASK)
						isEscapingFlow = false

					// Disregard raw XON/XOFF bytes as data; should have been escaped.
					} else if (buf[0] != FLOW_XON) && (buf[0] != FLOW_XOFF) {
						frameBuf = append(frameBuf, buf[0])
					}
				}
			}
		}
	}
}
