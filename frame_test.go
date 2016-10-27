package nustomp

import "testing"
import "bytes"

var frameHeadertests = []struct {
	testHeader          string
	expectedFrameHeader FrameHeader
}{
	{
		"ack:client",
		FrameHeader{"ack", "client"},
	},
	{
		"receipt-id:message-12345",
		FrameHeader{"receipt-id", "message-12345"},
	},
}

var rawMessage = `RECEIPT
receipt-id:77
        
` + "\x00"

var rawMessage2 = `MESSAGE
content-length:14
subscription:0
message-id:007
destination:/queue/a
content-type:text/plain

hello queue a` + "\x00"

var rawMessage3 = `MESSAGE
content-length:12
subscription:1
message-id:005
destination:/queue/b
content-type:text/plain

hello queue` + "\x00"

var frameTests = []struct {
	testFrame     string
	expectedFrame Frame
}{
	{
		rawMessage,
		Frame{
			command: Receipt,
			headers: []FrameHeader{
				{key: "receipt-id", value: "77"},
			},
			body: []byte{},
		},
	},
	{
		rawMessage2,
		Frame{
			command: Message,
			headers: []FrameHeader{
				// {key: "content-length", value: fmt.Sprintf("%d", len([]byte(rawMessage2)))},
				{key: "content-length", value: "14"},
				{key: "subscription", value: "0"},
				{key: "message-id", value: "007"},
				{key: "destination", value: "/queue/a"},
				{key: "content-type", value: "text/plain"},
			},
			body: []byte{},
		},
	},
	{
		rawMessage3,
		Frame{
			command: Message,
			headers: []FrameHeader{
				// {key: "content-length", value: fmt.Sprintf("%d", len([]byte(rawMessage2)))},
				{key: "content-length", value: "12"},
				{key: "subscription", value: "1"},
				{key: "message-id", value: "005"},
				{key: "destination", value: "/queue/b"},
				{key: "content-type", value: "text/plain"},
			},
			body: []byte{},
		},
	},
}

func TestParseFrameHeader(t *testing.T) {
	for _, test := range frameHeadertests {
		fh, err := parseFrameHeader([]byte(test.testHeader))
		if err != nil {
			t.Error(err)
		}
		if fh.key != test.expectedFrameHeader.key {
			t.Errorf("Expected header key to be 'receipt-id'. Found value %s", fh.key)
		}
		if fh.value != test.expectedFrameHeader.value {
			t.Errorf("Expected header value to be 'messahe-12345'. Found value %s", fh.value)
		}
	}
}

func TestParseFrame(t *testing.T) {
	for _, test := range frameTests {
		frame, err := parseFrame(bytes.NewReader([]byte(test.testFrame)))
		if err != nil {
			t.Error(err)
		}

		if frame.command != test.expectedFrame.command {
			t.Errorf("Expected frame to have command %s instead of %s", frame.command, test.expectedFrame.command)
		}

		if len(frame.headers) != len(test.expectedFrame.headers) {
			t.Errorf("Expected frame to have %d headers instead of available %d", len(test.expectedFrame.headers), len(frame.headers))
		}

		for i := 0; i < len(test.expectedFrame.headers); i++ {
			expectedKey := test.expectedFrame.headers[i].key
			expectedValue := test.expectedFrame.headers[i].value

			if expectedKey != frame.headers[i].key {
				t.Errorf("Expected frame header %s but found %s", test.expectedFrame.headers[i].key, frame.headers[i].key)
			}

			if expectedValue != frame.headers[i].value {
				t.Errorf("Expected the value of the frame header %s to be %s instead of %s", expectedKey, expectedValue, frame.headers[i].value)
			}
		}
	}
}
