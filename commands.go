package nustomp

import "fmt"

// Command represents a stomp frame command
type Command string

const (
	Connect     Command = "CONNECT"
	Connected   Command = "CONNECTED"
	Stomp       Command = "STOMP"
	Send        Command = "SEND"
	Subscribe   Command = "SUBSCRIBE"
	Unsubscribe Command = "UNSUBSCRIBE"
	Begin       Command = "BEGIN"
	Commit      Command = "COMMIT"
	Abort       Command = "ABORT"
	Ack         Command = "ACK"
	Nack        Command = "NACK"
	Disconnect  Command = "DISCONNECT"

	Message Command = "MESSAGE"
	Receipt Command = "RECEIPT"
	Error   Command = "ERROR"

	// The heart beat is not actually a command
	// According to the STOMP specifcation clients that don't have any STOMP frame
	// to send can send the EOL character for heartbeats
	HeartBeat Command = "HeartBeat"
)

func parseCommand(r []byte) (Command, error) {
	var cmd Command
	switch string(r) {
	case "\r\n", "\n":
		cmd = HeartBeat
	case "CONNECT":
		cmd = Connect
	case "STOMP":
		cmd = Stomp
	case "CONNECTED":
		cmd = Connected
	case "SEND":
		cmd = Send
	case "SUBSCRIBE":
		cmd = Subscribe
	case "UNSUBSCRIBE":
		cmd = Unsubscribe
	case "BEGIN":
		cmd = Begin
	case "COMMIT":
		cmd = Commit
	case "ABORT":
		cmd = Abort
	case "ACK":
		cmd = Ack
	case "NACK":
		cmd = Nack
	case "DISCONNECT":
		cmd = Disconnect
	case "MESSAGE":
		cmd = Message
	case "RECEIPT":
		cmd = Receipt
	case "ERROR":
		cmd = Error
	default:
		return "", fmt.Errorf("Unable to parse command %s", string(r))
	}

	return cmd, nil
}
