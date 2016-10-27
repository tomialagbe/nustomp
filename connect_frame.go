package nustomp

import (
	"fmt"
	"strconv"
	"strings"
)

// handles the CONNECT or STOMP frames
// Handle the accept-version header
// As per the STOMP specification, pick the highest version number from the list of versions the client supports
// and continue with that version
func handleConnectFrame(client *Client, frame *Frame) (*Frame, error) {

	// construct the response CONNECTED frame
	connectedFrame := new(Frame)
	connectedFrame.command = Connected
	headers := []FrameHeader{}

	// handle version
	version, err := getSupportedVersion(*frame)
	if err != nil {
		return nil, err
	}
	client.stompVersion = version
	headers = append(headers, FrameHeader{"version", fmt.Sprintf("%1.1f", version)})

	// handle heart-beat settings
	heartbeatheader := frame.GetHeader("heart-beat")
	if heartbeatheader != "" {
		hbparts := strings.Split(heartbeatheader, ",")
		if len(hbparts) != 2 {
			return nil, fmt.Errorf("Failed to parse 'heart-beat' header %s", heartbeatheader)
		}

		clientHbX, err := strconv.Atoi(strings.TrimSpace(hbparts[0]))
		if err != nil {
			return nil, fmt.Errorf("Failed to parse 'heart-beat' header %s. \nBad value %s.", heartbeatheader, hbparts[0])
		}

		clientHbY, err := strconv.Atoi(strings.TrimSpace(hbparts[1]))
		if err != nil {
			return nil, fmt.Errorf("Failed to parse 'heart-beat' header %s. \nBad value %s.", heartbeatheader, hbparts[0])
		}

		client.heartBeatX = clientHbX
		client.heartBeatY = clientHbY
		headers = append(headers, FrameHeader{"heart-beat", fmt.Sprintf("%d,%d", client.server.heartBeatX, client.server.heartBeatY)})
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
