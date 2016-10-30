package nustomp

import (
	"fmt"
	"strconv"
)

func handleSubscribeFrame(client *Client, frame *Frame) (*Frame, error) {
	receiptFrame, err := handleReceiptHeader(frame, false)
	if err != nil {
		return nil, err
	}

	destination := frame.GetHeader("destination")
	if destination == "" {
		return nil, fmt.Errorf("Unable to process frame. The 'destination' header is reqiured")
	}

	subscriptionIDStr := frame.GetHeader("id")
	if subscriptionIDStr == "" {
		return nil, fmt.Errorf("Unable to process frame. The 'id' header for this subscription is required")
	}
	subscriptionID, err := strconv.Atoi(subscriptionIDStr)
	if err != nil {
		return nil, fmt.Errorf("Unable to process frame. Invalid value '%s' for the 'id' header. Expected a numeric value", subscriptionIDStr)
	}

	ack := auto // ACK mode defaults to auto
	ackstr := frame.GetHeader("ack")
	if ackstr != "" {
		ack = strToAckMode(ackstr)
	}

	// subscribe client
	client.server.addSubscription(destination, subscriptionID, client.id, ack)

	if receiptFrame != nil {
		return receiptFrame, nil
	}

	return nil, nil
}

func handleSendFrame(client *Client, frame *Frame) (*Frame, error) {
	destination := frame.GetHeader("destination")
	if destination == "" {
		return nil, fmt.Errorf("Unable to process frame. The 'destination' header is reqiured")
	}

	conttype := ""
	if len(frame.body) > 0 {
		conttype = frame.GetHeader("content-type")
		if conttype == "" {
			return nil, fmt.Errorf("Unable to process frame. The content-type header is required")
		}

		//handling the content-length header is optional since the STOMP specification uses SHOULD
	}

	// TODO: handle user-defined headers

	// send message
	client.server.messageChannels[destination] <- message{destination, conttype, frame.body, client.id, false}

	receiptFrame, err := handleReceiptHeader(frame, false)
	if err != nil {
		return nil, err
	}
	if receiptFrame != nil {
		return receiptFrame, nil
	}
	return nil, nil
}
