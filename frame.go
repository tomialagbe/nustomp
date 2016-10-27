package nustomp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

const (
	maxHeaderLines  = 20          // a frame should not have more than maxHeaderLines headers
	maxHeaderLength = 1024        // a frame header line should not be longer than maxHeaderLength bytes
	maxBodyLength   = 1024 * 1000 // 1MB maximum frame body length
)

// Frame represents a STOMP frame
type Frame struct {
	command  Command
	headers  []FrameHeader
	body     []byte
	mimeType string
}

// ToBytes Converts a frame to a byte stream
func (f Frame) ToBytes() []byte {
	// write command
	buffer := bytes.NewBufferString(string(f.command))
	buffer.WriteString("\n")

	// write headers
	for _, header := range f.headers {
		buffer.WriteString(fmt.Sprintf("%s\n", header.ToString()))
	}
	buffer.WriteString("\n") // blank line between header and body

	// write body
	buffer.Write(f.body)
	// write null octet
	buffer.WriteByte(0)
	return buffer.Bytes()
}

// GetHeader returns the value of a header in the frame
// If the header does not exist, an empty string is returned
func (f Frame) GetHeader(key string) string {
	for _, header := range f.headers {
		if header.key == key {
			return header.value
		}
	}
	return ""
}

// FrameHeader represents the headers sent in a stomp frame
type FrameHeader struct {
	key   string
	value string
}

// ToBytes converts a FrameHeader to a byte stream
func (h FrameHeader) ToBytes() []byte {
	return []byte(h.ToString())
}

// ToString returns the string representation of a frame header
func (h FrameHeader) ToString() string {
	return fmt.Sprintf("%s:%s", h.key, h.value)
}

func parseFrame(r io.Reader) (*Frame, error) {
	rd := bufio.NewReader(r)

	readline := func() ([]byte, error) {
		line := bytes.NewBuffer([]byte{})
		for ln, isPfx, err := rd.ReadLine(); ; {
			if err != nil {
				return nil, err
			}

			line.Write(ln)
			if !isPfx {
				break
			}
		}

		return line.Bytes(), nil
	}

	// get the command
	commandln, err := readline()
	if err != nil {
		return nil, err
	}

	command, err := parseCommand(commandln)
	log.Printf("Parsed frame command %s", string(commandln))
	if err != nil {
		return nil, err
	}

	// headers
	var contlen = -1
	var conttype = ""
	headers := []FrameHeader{}
	for {
		ln, err := readline()
		if err != nil {
			return nil, err
		}

		// blank line between headers and body
		if len(strings.TrimSpace(string(ln))) == 0 {
			break
		}

		// if there are more headers than allowed
		if len(headers) == maxHeaderLength {
			return nil, fmt.Errorf("Too many headers. The maximum number of allowed headers is %d", maxHeaderLength)
		}

		if len(ln) > maxHeaderLength {
			return nil, fmt.Errorf("Header lines should not be longer that %d bytes", maxHeaderLength)
		}
		header, err := parseFrameHeader(ln)
		log.Printf("Parsed frame header %s", string(ln))
		if err != nil {
			return nil, err
		}
		if header.key == "content-length" {
			contlen, err = strconv.Atoi(header.value)
			if err != nil {
				return nil, err
			}
		}
		if header.key == "content-type" {
			conttype = strings.TrimSpace(header.value)
		}

		headers = append(headers, header)
	}

	// get the body
	var body []byte
	if contlen != -1 {
		// read contlen bytes from the rest of the stream
		log.Printf("Reading %d bytes from body", contlen)
		for i := 0; i < contlen; i++ {
			b, err := rd.ReadByte()
			if err != nil && err != io.EOF {
				return nil, err
			}
			if len(body) == maxBodyLength {
				return nil, fmt.Errorf("Content too large. The body should not be more than %d bytes", maxBodyLength)
			}
			body = append(body, b)
		}

		// if the last byte in the body is not the null octet, then return an error
		if body[len(body)-1] != byte('\x00') {
			return nil, fmt.Errorf("The last octet in the body stream should be the null octet. After reading %d bytes from the body as specified by the content-length header, the null octet was not present.", contlen)
		}
	} else {
		// read the rest of the stream until the first occurrence of the null character (0x00)
		body, err = rd.ReadBytes(byte('\x00'))
	}
	// remove the trailing null character
	body = body[:len(body)-1]
	log.Printf("Read frame body %s", string(body))

	frame := new(Frame)
	frame.command = command
	frame.headers = headers
	frame.body = body
	frame.mimeType = conttype

	return frame, nil
}

func parseFrameHeader(b []byte) (FrameHeader, error) {
	var h FrameHeader
	headerln := string(b)
	parts := strings.Split(headerln, ":")
	if len(parts) != 2 {
		return h, fmt.Errorf("Invalid header %s. Expected a header in the form <header_key>:<header_value>", headerln)
	}

	h = FrameHeader{key: parts[0], value: parts[1]}
	return h, nil
}

// handles a stomp frame and return the appropriate response frame
func handleFrame(client *Client, frame *Frame) (*Frame, error) {
	switch frame.command {
	case Connect, Stomp:
		return handleConnectFrame(client, frame)
	default:
		return nil, errors.New("Currently unable to handle this frame")
	}
}

// handles the CONNECT or STOMP frames
// Handle the accept-version header
// As per the STOMP specification, pick the highest version number from the list of versions the client supports
// and continue with that version
func handleConnectFrame(client *Client, frame *Frame) (*Frame, error) {

	// handle version
	version, err := getSupportedVersion(*frame)
	if err != nil {
		return nil, err
	}
	client.stompVersion = version

	// handle heart-beat settings
	heartbeat := frame.GetHeader("heart-beat")
	if heartbeat != "" {

	}

	// construct the response CONNECTED frame
	connectedFrame := new(Frame)
	connectedFrame.command = Connected

	headers := []FrameHeader{
		{"version", fmt.Sprintf("%1.1f", version)},
	}
	connectedFrame.headers = headers
	return connectedFrame, nil
}

func getSupportedVersion(frame Frame) (float64, error) {
	// handle version
	var highestVersion = 1.0
	versionstr := frame.GetHeader("accept-version")
	if versionstr != "" {
		split := strings.Split(versionstr, ",")
		for _, ver := range split {
			ver = strings.TrimSpace(ver)
			versionnum, err := strconv.ParseFloat(ver, 64)
			//bad header value
			if err != nil {
				return 0, fmt.Errorf("Failed to parse 'accept-version' header %s. \nBad value %s. \nSupported Protocol versions are 1.0, 1.1, 1.2.", versionstr, ver)
			}

			if versionnum > 1.2 || versionnum < 1.0 {
				return 0, fmt.Errorf("Invalid version number. Supported versions are 1.0, 1.1, 1.2")
			}

			// header is ok
			if versionnum > highestVersion {
				highestVersion = versionnum
			}
		}
	}
	return highestVersion, nil
}
