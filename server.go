package nustomp

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"sync"
)

// Server represents a STOMP Server
type Server struct {
	bindPort int
	// connectChan chan net.Conn
	clients         map[int]*Client
	clientCount     int
	clientCountLock sync.Mutex
	heartBeatX      int // the minimum number of seconds between server-sent heartbeats, or zero for none
	heartBeatY      int // the desired number of seconds between heartbeats from client that can send heartbeats
}

// NewServer creates a new STOMP Server.
// port is the port that the server should listen on
func NewServer(port int) *Server {
	s := new(Server)
	s.bindPort = port
	// s.connectChan = make(chan net.Conn)
	s.clients = make(map[int]*Client)
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
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	id := s.addClient(conn)
	// start conversation with this new client
	s.startConversation(id)
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
		// then, delete the client
		delete(s.clients, id)
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
		// handle the frame
		responseFrame, err := handleFrame(client, frame)
		if err != nil {
			s.sendErrorFrame(id, frame, err)
			return
		}
		client.conn.Write(responseFrame.ToBytes())

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
	id           int
	remoteAddr   string
	conn         net.Conn
	server       *Server
	stompVersion float64 // the stomp version to use in communicating with this client
	heartBeatX   int     // the minimum number of milliseconds between heartbeats from this client or zero for none
	heartBeatY   int
}

// Close terminates the connection with the client
func (c Client) Close() {
	c.conn.Close()
}

// canSendHeartBeat returns true if this client can send heart beats to a server
func (c Client) canSendHeartBeat() bool {
	// if the client's x heartbeat is not zero and the servers y-heartbeat is not zero
	if c.heartBeatX > 0 && c.server.heartBeatY > 0 {
		return true
	}
	return false
}
