package udp

import (
	"encoding/binary"
	"net"

	"github.com/chihaya/chihaya/bittorrent"
)

const (
	connectActionID uint32 = iota
	announceActionID
	scrapeActionID
	errorActionID
)

// Option-Types as described in BEP 41 and BEP 45.
const (
	optionEndOfOptions byte = 0x0
	optionNOP               = 0x1
	optionURLData           = 0x2
)

var (
	// initialConnectionID is the magic initial connection ID specified by BEP 15.
	initialConnectionID = []byte{0, 0, 0x04, 0x17, 0x27, 0x10, 0x19, 0x80}

	// emptyIPs are the value of an IP field that has been left blank.
	emptyIPv4 = []byte{0, 0, 0, 0}
	emptyIPv6 = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	// eventIDs map values described in BEP 15 to Events.
	eventIDs = []bittorrent.Event{
		bittorrent.None,
		bittorrent.Completed,
		bittorrent.Started,
		bittorrent.Stopped,
	}

	errMalformedPacket = bittorrent.ClientError("malformed packet")
	errMalformedIP     = bittorrent.ClientError("malformed IP address")
	errMalformedEvent  = bittorrent.ClientError("malformed event ID")
	errUnknownAction   = bittorrent.ClientError("unknown action ID")
	errBadConnectionID = bittorrent.ClientError("bad connection ID")
)

// ParseAnnounce parses an AnnounceRequest from a UDP request.
//
// If allowIPSpoofing is true, IPs provided via params will be used.
func ParseAnnounce(r Request, allowIPSpoofing bool) (*bittorrent.AnnounceRequest, error) {
	if len(r.Packet) < 98 {
		return nil, errMalformedPacket
	}

	infohash := r.Packet[16:36]
	peerID := r.Packet[36:56]
	downloaded := binary.BigEndian.Uint64(r.Packet[56:64])
	left := binary.BigEndian.Uint64(r.Packet[64:72])
	uploaded := binary.BigEndian.Uint64(r.Packet[72:80])

	eventID := int(r.Packet[83])
	if eventID >= len(eventIDs) {
		return nil, errMalformedEvent
	}

	ip := r.IP
	ipbytes := r.Packet[84:88]
	if allowIPSpoofing {
		ip = net.IP(ipbytes)
	}
	if !allowIPSpoofing && r.IP == nil {
		// We have no IP address to fallback on.
		return nil, errMalformedIP
	}

	numWant := binary.BigEndian.Uint32(r.Packet[92:96])
	port := binary.BigEndian.Uint16(r.Packet[96:98])

	params, err := handleOptionalParameters(r.Packet)
	if err != nil {
		return nil, err
	}

	return &bittorrent.AnnounceRequest{
		Event:      eventIDs[eventID],
		InfoHash:   bittorrent.InfoHashFromBytes(infohash),
		NumWant:    uint32(numWant),
		Left:       left,
		Downloaded: downloaded,
		Uploaded:   uploaded,
		Peer: bittorrent.Peer{
			ID:   bittorrent.PeerIDFromBytes(peerID),
			IP:   ip,
			Port: port,
		},
		Params: params,
	}, nil
}

// handleOptionalParameters parses the optional parameters as described in BEP
// 41 and updates an announce with the values parsed.
func handleOptionalParameters(packet []byte) (params bittorrent.Params, err error) {
	if len(packet) <= 98 {
		return
	}

	optionStartIndex := 98
	for optionStartIndex < len(packet)-1 {
		option := packet[optionStartIndex]
		switch option {
		case optionEndOfOptions:
			return

		case optionNOP:
			optionStartIndex++

		case optionURLData:
			if optionStartIndex+1 > len(packet)-1 {
				return params, errMalformedPacket
			}

			length := int(packet[optionStartIndex+1])
			if optionStartIndex+1+length > len(packet)-1 {
				return params, errMalformedPacket
			}

			// TODO(chihaya): Actually parse the URL Data as described in BEP 41
			// into something that fulfills the bittorrent.Params interface.

			optionStartIndex += 1 + length
		default:
			return
		}
	}

	return
}

// ParseScrape parses a ScrapeRequest from a UDP request.
func ParseScrape(r Request) (*bittorrent.ScrapeRequest, error) {
	// If a scrape isn't at least 36 bytes long, it's malformed.
	if len(r.Packet) < 36 {
		return nil, errMalformedPacket
	}

	// Skip past the initial headers and check that the bytes left equal the
	// length of a valid list of infohashes.
	r.Packet = r.Packet[16:]
	if len(r.Packet)%20 != 0 {
		return nil, errMalformedPacket
	}

	// Allocate a list of infohashes and append it to the list until we're out.
	var infohashes []bittorrent.InfoHash
	for len(r.Packet) >= 20 {
		infohashes = append(infohashes, bittorrent.InfoHashFromBytes(r.Packet[:20]))
		r.Packet = r.Packet[20:]
	}

	return &bittorrent.ScrapeRequest{
		InfoHashes: infohashes,
	}, nil
}
