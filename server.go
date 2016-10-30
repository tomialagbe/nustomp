package nustomp

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// Server represents a STOMP Server
type Server struct {
	bindPort int

	clients         map[int]*Client
	clientCount     int
	clientCountLock sync.Mutex

	heartBeatX int // the minimum number of seconds between server-sent heartbeats, or zero for none
	heartBeatY int // the desired number of seconds between heartbeats from client that can send heartbeats

	subscriptions   map[string][]subscription // a map of destinations to client subscriptions
	messageChannels map[string]chan message   // a map of destinations to the channels over which messages would be dispatched
	unackedMessages map[int][]message         // a map of subscription ids to unacked messages
}

// NewServer creates a new STOMP Server.
// port is the port that the server should listen on
func NewServer(port int) *Server {
	s := new(Server)
	s.bindPort = port
	s.clients = make(map[int]*Client)
	s.subscriptions = make(map[string][]subscription)
	s.messageChannels = make(map[string]chan message)
	s.unackedMessages = make(map[int][]message)
	return s
}

// SetHeartBeat sets the heart-beat interval for this server
// For example the call s.SetHeartBeat(30000, 60000) this server expects heartbeats from clients (that support heartbeats)
// every 30 seconds and guarantees approximately a minute between the heartbeats it sends
func (s *Server) SetHeartBeat(x, y int) {
	s.heartBeatX = x
	s.heartBeatY = y
}

// Start starts the STOMP server.
// Once the server is started it listens for connections from incoming clients
// on the port specified during creation
func (s *Server) Start() {
	log.Printf("Starting server on port :%d", s.bindPort)

	// start listening for and accepting connections
	s.acceptConnections()

	// TODO: start listening for, and sending out heartbeats
}

// canSendHeartBeatToClient returns true if the server can send heartbeats to the client
// specified by clientID
func (s *Server) canSendHeartBeatToClient(clientID int) bool {
	client, ok := s.clients[clientID]
	if !ok {
		return false
	}

	if s.heartBeatX > 0 && client.heartBeatY > 0 {
		return true
	}
	return false
}

// acceptConnections listens for incoming connections
func (s *Server) acceptConnections() {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.bindPort))
	if err != nil {
		panic(err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}

		log.Printf("Recieved connection from %s", conn.RemoteAddr().String())

		// handle new connection on new goroutine
		go func() {
			id := s.addClient(conn)
			// start conversation with this new client
			s.startConversation(id)
		}()
	}
}

// add a new client to this server and returns the id of the new client
func (s *Server) addClient(conn net.Conn) int {
	s.clientCountLock.Lock()
	id := s.clientCount

	client := new(Client)
	client.id = id
	client.remoteAddr = conn.RemoteAddr().String()
	client.conn = conn
	client.stompVersion = 1.2
	client.server = s

	s.clients[id] = client
	s.clientCount++
	s.clientCountLock.Unlock()

	log.Printf("Added client %d: from %s", id, conn.RemoteAddr())
	return id
}

// remove a client from this server and close its connection.
func (s *Server) removeClient(id int) {
	client, ok := s.clients[id]
	if ok {
		// close the connection
		client.Close()
		client.server = nil

		// remove all subscriptions
		s.removeAllSubscriptionsForClient(client.id)

		// then, delete the client
		delete(s.clients, id)
	}
}

// subscribes a client with clientID to messages delivered on destination
func (s *Server) addSubscription(destination string, subscriptionID, clientID int, mode ackMode) {
	if s.subscriptions[destination] == nil {
		s.subscriptions[destination] = make([]subscription, 0, 100)
		// if this the first ever subscription to this destination, we need to create a new entry in the messageChannels map
		s.messageChannels[destination] = make(chan message, 100) // create a buffered channel that can buffer 100 messages
		go func() {
			for {
				msg, more := <-s.messageChannels[destination]
				if !more {
					return
				}
				s.dispatchMessage(msg, destination)
			}
		}()
	}

	subscr := subscription{subscriptionID, clientID, destination, mode}
	s.subscriptions[destination] = append(s.subscriptions[destination], subscr)
}

func (s *Server) removeSubscription(destination string, subscriptionID int) {
	subscrs := s.subscriptions[destination]
	if subscrs == nil {
		return
	}

	pos := -1
	for idx, sub := range subscrs {
		if subscriptionID == sub.id {
			pos = idx
			break
		}
	}

	if pos != -1 {
		// delete the subscription
		newsubs := append(subscrs[:pos], subscrs[pos+1:]...)
		if len(newsubs) == 0 {
			close(s.messageChannels[destination]) // close the channel if there are no more subscriptions
		}
		s.subscriptions[destination] = newsubs
	}
}

func (s *Server) removeAllSubscriptionsForClient(clientID int) {
	toremove := make(map[string][]int)
	for dest, subs := range s.subscriptions {
		for _, sub := range subs {
			if sub.clientID == clientID {
				if toremove[dest] == nil {
					toremove[dest] = []int{sub.id}
				} else {
					toremove[dest] = append(toremove[dest], sub.id)
				}
			}
		}
	}

	for k, ids := range toremove {
		for _, id := range ids {
			s.removeSubscription(k, id)
		}
	}
}

func (s *Server) addUnackedMessage(sub subscription, msg message) {
	if s.unackedMessages[sub.id] == nil {
		s.unackedMessages[sub.id] = make([]message, 0, 50)
	}
	s.unackedMessages[sub.id] = append(s.unackedMessages[sub.id], msg)
}

func (s *Server) dispatchMessage(msg message, destination string) {
	subs := s.subscriptions[destination]
	idx := -1
	for i, sub := range subs {
		if msg.sender == sub.clientID {
			idx = i
			break
		}
	}

	if idx != -1 {
		// the message should be dispatched to all other clients except the sender
		subs = append(subs[:idx], subs[idx+1:]...)
	}

	// dispatch to all clients
	for _, sub := range subs {
		client := s.clients[sub.clientID]
		// send message to client
		sendMessageToClient(*client, msg, sub)
	}
}

// start a conversation with a client with the given id
func (s *Server) startConversation(id int) {
	client, ok := s.clients[id]
	if !ok {
		panic(fmt.Errorf("Client %d not found", id))
	}

	var err error
	for frame, err := parseFrame(client.conn); err == nil; {
		if frame.command != HeartBeat {
			// handle the frame (non heart-beat frames)
			responseFrame, err := handleFrame(client, frame)
			if err != nil {
				s.sendErrorFrame(id, frame, err)
				return
			}

			if responseFrame != nil {
				client.conn.Write(responseFrame.ToBytes())
			}

			// if the command was a DISCONNECT command, disconnect the client after sending back the response
			if frame.command == Disconnect {
				client.conn.Close()
				client.server.removeClient(client.id)
				return
			}
		}

		// reset the heart-beat timer
		client.resetHeartBeatTimer()

		// read the next frame
		frame, err = parseFrame(client.conn)
	}

	// there was an error parsing the frame,return an error frame back to the client and disconnect
	s.sendErrorFrame(id, nil, err)
}

func (s *Server) sendErrorFrame(clientid int, clientFrame *Frame, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	} else {
		errMsg = "An unknown server error occurred."
	}

	headers := []FrameHeader{
		{"message", errMsg},
	}
	if clientFrame != nil && clientFrame.GetHeader("receipt") != "" {
		headers = append(headers, FrameHeader{"receipt-id", clientFrame.GetHeader("receipt")})
	}

	bodybuff := bytes.NewBufferString("")
	if clientFrame != nil {
		bodybuff.WriteString("Client Request\n")
		bodybuff.WriteString("----------------\n")
		b := clientFrame.ToBytes()
		b = b[:len(b)-1] // remove trailing null octet
		bodybuff.Write(b)
		bodybuff.WriteString("\n---------------\n")
	}

	f := Frame{
		command: Error,
		headers: headers,
		body:    bodybuff.Bytes(),
	}

	client, ok := s.clients[clientid]
	if !ok {
		panic("Attempted to send error frame to non-existent client")
	}

	client.conn.Write(f.ToBytes())
	client.Close()
	s.removeClient(clientid)
}

// Client represents a STOMP client
type Client struct {
	id             int
	remoteAddr     string
	conn           net.Conn
	server         *Server
	stompVersion   float64 // the stomp version to use in communicating with this client
	heartBeatX     int     // the minimum number of milliseconds between heartbeats from this client or zero for none
	heartBeatY     int
	heartBeatTimer *time.Timer
}

// Close terminates the connection with the client
func (c *Client) Close() {
	c.conn.Close()
}

// SetHeartBeat sets the x and y heartbeats for the client
func (c *Client) SetHeartBeat(x, y int) {
	c.heartBeatX = x
	c.heartBeatY = y

	// start listening for heartbeats only if the client and server support it
	if c.canSendHeartBeat() {
		c.resetHeartBeatTimer()
		go func() {
			select {
			case <-c.heartBeatTimer.C:
				// disconnect this client because it has been idle up until the timeout
				c.Close()
				c.server.removeClient(c.id)
				return
			}
		}()
	}
}

func (c *Client) resetHeartBeatTimer() {
	// add 10 extra seconds to heart beat interval
	dur := time.Duration(c.heartBeatX) + (time.Second * 10)
	if c.heartBeatTimer == nil {
		c.heartBeatTimer = time.NewTimer(dur)
	} else {
		c.heartBeatTimer.Reset(dur)
	}

}

// canSendHeartBeat returns true if this client can send heart beats to a server
func (c *Client) canSendHeartBeat() bool {
	if c.heartBeatY > 0 && c.server.heartBeatX > 0 {
		return true
	}
	return false
}

func (c *Client) canRecieveHeartBeat() bool {
	// if the client's x heartbeat is not zero and the servers y-heartbeat is not zero
	if c.heartBeatX > 0 && c.server.heartBeatY > 0 {
		return true
	}
	return false
}
