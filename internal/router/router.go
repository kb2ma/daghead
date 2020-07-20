package router

import (
	"errors"
	"fmt"
	"github.com/kb2ma/daghead/internal/log"
)

const (
	PAGE_ONE_DISPATCH  byte = 0xF1
	CRITICAL_6LoRH     byte = 0x80
	MASK_6LoRH         byte = 0xE0
	TYPE_6LoRH_RPI     byte = 0x05
	IANA_IPv6HOPHEADER byte = 0
	IANA_ICMPv6        byte = 0x3A
	RPI_FLAG_MASK      byte = 0x1F
	RPI_O_FLAG         byte = 0x10
	RPI_R_FLAG         byte = 0x08
	RPI_I_FLAG         byte = 0x02
	RPI_K_FLAG         byte = 0x01
	// there is no IANA for IPV6 HEADER right now, we use NHC identifier for it
	// https://tools.ietf.org/html/rfc6282#section-4.2
	IPV6_HEADER        byte = 0xEE
	IPHC_TF_ELIDED     byte = 3
	IPHC_NH_INLINE     byte = 0
	IPHC_HLIM_64       byte = 2
	IPHC_CID_NONE      byte = 0
	IPHC_SAC_STATEFUL  byte = 1
	IPHC_SAM_64B       byte = 1
	IPHC_DAC_STATEFUL  byte = 1
	IPHC_DAM_64B       byte = 1

	RPL_TYPE_TRANSIT_INFORMATION byte = 0x06
	RPL_TYPE_TARGET_INFORMATION byte  = 0x05
)

var (
	NETWORK_PREFIX  = [8]byte{0xBB, 0xBB, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
)

// Container for parsed IP header contents
type IpData struct {
	Source [16]byte
	Dest [16]byte
	Fields map[string]int
}

/*
Reads a data packet from the root node, and returns a map of 6LoWPAN field data
found. Initializes provided IpData as needed.

Presently, supports only reading RPL DAO to maintain source routing southbound
into the mesh.
*/
func ReadData(ip *IpData, preHop byte, data []byte) (err error) {
	if ip.Fields == nil {
		ip.Fields = make(map[string]int)
	}
	ip.Fields["pre_hop"] = int(preHop)

	// RFC 8025
	// Expect 6LoWPAN adaptation header to begin with a parsing context switch
	// to Page 1.
	i := 0
	if data[i] == PAGE_ONE_DISPATCH {
		// RFC 8138
		// Read 6LoRH-RPI (critical) 0b100xxxxx header.
		//  0                   1                   2
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+  ...  -+-+-+
		// |1|0|0|O|R|F|I|K| 6LoRH Type=5  |   Compressed fields  |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+  ...  -+-+-+
		i++
		if (data[i] & MASK_6LoRH == CRITICAL_6LoRH) && (data[i+1] == TYPE_6LoRH_RPI) {
			ip.Fields["next_header"] = int(IANA_IPv6HOPHEADER)
			// RPI flags in the 5 least signficiant bits of the first byte.
			ip.Fields["hop_flags"] = int(data[i] & RPI_FLAG_MASK)
			i += 2

			// Next 0 or 1 byte is RPL Instance ID
			if ip.Fields["hop_flags"] & int(RPI_I_FLAG) == 0 {
				ip.Fields["hop_rplInstanceID"] = int(data[i])
				i++
			} else {
				// elided when only one RPL instance
				ip.Fields["hop_rplInstanceID"] = 0
			}

			// Next 1 or 2 bytes RPL sender rank. If one byte, must be a multiple
			// of 256, so LSB elided.
			if ip.Fields["hop_flags"] & int(RPI_K_FLAG) == 0 {
				ip.Fields["hop_senderRank"] = (int(data[i]) << 8) + int(data[i+1])
				i += 2
			} else {
				ip.Fields["hop_senderRank"] = int(data[i]) << 8
				i++
			}

			// expect IPHC after 6LoRH RPI
			ip.Fields["hop_next_header"] = int(IPV6_HEADER)

		} else {
			// No support yet for handling 6LoRH of IP in IP or deadline,
			// as implemented in OpenVisualizer.
		}

	// RFC 6282
	// Read IPHC compressed IP header, 2 bytes
	//   0                                       1
	//   0   1   2   3   4   5   6   7   8   9   0   1   2   3   4   5
	// +---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+
	// | 0 | 1 | 1 |  TF   |NH | HLIM  |CID|SAC|  SAM  | M |DAC|  DAM  |
	// +---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+---+
	} else {
		if (data[0] >> 5) != 0x03 {
			err = errors.New(fmt.Sprintf("not a 6LowPAN IPHC header 0x%X", data[0]))
			return
		}

		// next byte after IPHC header
		i = 2
		// Traffic Class and Flow Label
		tf := (data[0] >> 3) & 0x3
		if tf == IPHC_TF_ELIDED {
			ip.Fields["flow_label"] = 0
		} else {
			log.Println(log.WARN, "unsupported IPHC TF value")
		}

		// Next Header
		nh := (data[0] >> 2) & 0x1
		if nh == IPHC_NH_INLINE {
			ip.Fields["next_header"] = int(data[i])
			i++
		} else {
			log.Println(log.WARN, "unsupported IPHC NH value")
		}

		// Hop limit
		hlim := data[0] & 0x3
		if hlim == IPHC_HLIM_64 {
			ip.Fields["hop_limit"] = 64
		} else {
			log.Println(log.WARN, "unsupported IPHC HLIM value")
		}

		// Context Identifier extension
		cid := (data[1] >> 7) & 0x1
		if cid != IPHC_CID_NONE {
			log.Println(log.WARN, "unsupported IPHC CID value")
		}

		// Source Address Compression
		sac := (data[1] >> 6) & 0x1
		if sac == IPHC_SAC_STATEFUL {
			copy(ip.Source[:8], NETWORK_PREFIX[:])
		} else {
			log.Println(log.WARN, "unsupported IPHC SAC value")
		}

		// Source Address Mode
		sam := (data[1] >> 4) & 0x3
		if sam == IPHC_SAM_64B {
			copy(ip.Source[8:16], data[i:i+8])
			i += 8
		} else {
			log.Println(log.WARN, "unsupported IPHC SAM value")
		}

		// Multicast Compression (ignored)

		// Destination Address Compression
		dac := (data[1] >> 2) & 0x1
		if dac == IPHC_DAC_STATEFUL {
			copy(ip.Dest[:8], NETWORK_PREFIX[:])
		} else {
			log.Println(log.WARN, "unsupported IPHC DAC value")
		}

		// Destination Address Mode
		dam := data[1] & 0x3
		if dam == IPHC_DAM_64B {
			copy(ip.Dest[8:16], data[i:i+8])
			i += 8
		} else {
			log.Println(log.WARN, "unsupported IPHC DAM value")
		}

		// Next header
		// Skip IPHC_NH_COMPRESSED handling; unsupported presently
		// Skip next header is hop-by-hop handling; unsupported presently
	}

	// payload
	ip.Fields["version"] = 6
	ip.Fields["traffic_class"] = 0
	ip.Fields["payload"] = i
	ip.Fields["payload_length"] = len(data) - i

	return
}

func ReadRpl(source *[16]byte, data []byte) {
	// skip DAO header
	i := 20
	log.Printf(log.INFO, "DAO from [% X]", source[8:])
	if data[i] == RPL_TYPE_TRANSIT_INFORMATION {
		// skip transit info header
		i += 6
		log.Printf(log.INFO, "parent [% X]", data[i+8:i+16])
	}
}

