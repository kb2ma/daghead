package router

const (
	PAGE_ONE_DISPATCH byte  = 0xF1
	CRITICAL_6LoRH byte     = 0x80
	MASK_6LoRH byte         = 0xE0
	TYPE_6LoRH_RPI byte     = 0x05
	IANA_IPv6HOPHEADER byte = 0
    RPI_FLAG_MASK byte      = 0x1F
    RPI_I_FLAG byte         = 0x02
    RPI_K_FLAG byte         = 0x01
    // there is no IANA for IPV6 HEADER right now, we use NHC identifier for it
    // https://tools.ietf.org/html/rfc6282#section-4.2
    IPV6_HEADER byte        = 0xEE
)

/*
Read a data packet from the root node, and returns a map of 6LoWPAN field data
found.

Presently, supports only reading RPL DAO to maintain source routing southbound
into the mesh.
*/
func ReadData(preHop byte, data []byte) (map[string]int) {
	ipFields := make(map[string]int)
	ipFields["pre_hop"] = int(preHop)

	// RFC 8025
	// Expect 6LoWPAN adaptation header to begin with a context switch to Page 1.
	i := 0
	if data[i] == PAGE_ONE_DISPATCH {
		// RFC 8138
		// Handle when the next two bytes specify RPL Packet Information type of
		// 6LoWPAN routing header. The first byte specifies a 6LoRH hop-by-hop header
		// (0b100xxxxx), as well as flags for RPI. The 6LoRH type of RPI is specified
		// in the second byte.
		i++
		if (data[i] & MASK_6LoRH == CRITICAL_6LoRH) && (data[i+1] == TYPE_6LoRH_RPI) {
			ipFields["next_header"] = int(IANA_IPv6HOPHEADER)
			// RPI flags in the 5 least signficiant bits of the first byte.
			ipFields["hop_flags"] = int(data[i] & RPI_FLAG_MASK)
			i += 2

			// Next 0 or 1 byte is RPL Instance ID
			if ipFields["hop_flags"] & int(RPI_I_FLAG) == 0 {
				ipFields["hop_rplInstanceID"] = int(data[i])
				i++
			} else {
				// elided when only one RPL instance
				ipFields["hop_rplInstanceID"] = 0
			}

			// Next 1 or 2 bytes RPL sender rank. If one byte, must be a multiple
			// of 256, so LSB elided.
			if ipFields["hop_flags"] & int(RPI_K_FLAG) == 0 {
				ipFields["hop_senderRank"] = (int(data[i]) << 8) + int(data[i+1])
				i += 2
			} else {
				ipFields["hop_senderRank"] = int(data[i]) << 8
				i++
			}

			// expect IPHC after 6LoRH RPI
			ipFields["hop_next_header"] = int(IPV6_HEADER)

		} else {
			// No support yet for handling 6LoRH of IP in IP or deadline,
			// as implemented in OpenVisualizer.
		}
	}

	// payload
	ipFields["version"] = 6
	ipFields["traffic_class"] = 0
	ipFields["payload"] = i
	ipFields["payload_length"] = len(data) - i

	return ipFields
}

