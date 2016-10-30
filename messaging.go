package nustomp

import "fmt"

type ackMode string

const (
	client           ackMode = "client"
	auto             ackMode = "auto"
	clientIndividual ackMode = "client-individual"
)

type message struct {
	destination string
	contentType string
	content     []byte
	sender      int
	acked       bool
}

type subscription struct {
	id          int
	clientID    int
	destination string
	ack         ackMode
}

func strToAckMode(in string) ackMode {
	switch in {
	case "client":
		return client
	case "client-individual":
		return clientIndividual
	case "auto":
		return auto
	default:
		return auto
	}
}

func newMessage(destination string, contentType string, content []byte, senderID int) message {
	return message{destination, contentType, content, senderID, false}
}

func sendMessageToClient(client Client, msg message, sub subscription) {
	// construct a MESSAGE frame to be sent to the client
	msgFrame := new(Frame)
	msgFrame.command = Message
	msgFrame.headers = []FrameHeader{
		{"subscription", fmt.Sprintf("%d", sub.id)},
		{"destination", sub.destination},
	}
	if msg.contentType != "" {
		msgFrame.headers = append(msgFrame.headers, FrameHeader{"content-type", msg.contentType})
	}
	if len(msg.content) > 0 {
		msgFrame.headers = append(msgFrame.headers, FrameHeader{"content-length", fmt.Sprintf("%d", len(msg.content))})
	}
	// TODO: add other message headers

	msgFrame.body = msg.content

	client.conn.Write(msgFrame.ToBytes())

	// if the acknowledgement mode for this subscription is not auto,
	// add this message to the list of unacked messages
	if sub.ack != auto {
		client.server.addUnackedMessage(sub, msg)
	}
}
