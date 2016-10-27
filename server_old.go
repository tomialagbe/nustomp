package nustomp

// import (
// 	"fmt"
// 	"log"
// 	"net"
// )

// type client struct {
// 	remoteAddr string
// 	conn       net.Conn
// }

// // Server represents an instance of the STOMP server
// type Server struct {
// 	port          int
// 	clients       map[int]client
// 	clientCounter int
// }

// // NewServer creates a new STOMP server instance
// func NewServer(port int) *Server {
// 	s := new(Server)
// 	s.clientCounter = 0
// 	s.port = port
// 	s.clients = make(map[int]client)
// 	return s
// }

// func (s *Server) addClient(c client) {
// 	s.clients[s.clientCounter] = c
// 	s.clientCounter++

// 	// start communication with client over this connection
// 	go s.handleClient(c)
// }

// func (s *Server) removeClient(id int) {
// 	c := s.clients[id]
// 	delete(s.clients, id)
// 	c.conn.Close()
// }

// // Start the server
// func (s *Server) Start() {
// 	l, err := net.Listen("tcp", fmt.Sprintf("%d", s.port))
// 	if err != nil {
// 		panic(err)
// 	}

// 	clientchan := make(chan client)
// 	i := 0
// 	go func() {
// 		for {
// 			c, err := l.Accept()
// 			if err != nil {
// 				continue
// 			}
// 			i++
// 			log.Printf("Client %d: %v <-> %v\n", i, c.LocalAddr(), c.RemoteAddr())
// 			cl := client{c.RemoteAddr().String(), c}
// 			clientchan <- cl
// 		}
// 	}()

// 	select {
// 	case cl := <-clientchan:
// 		s.addClient(cl)
// 	default:
// 		// do nothing
// 	}
// }

// func (s Server) errorFrame(c net.Conn, err error) {
// 	f := Frame{
// 		command: Error,
// 		body:    []byte(err.Error()),
// 	}
// 	c.Write(f.ToBytes())
// 	c.Close()
// }

// func (s *Server) handleClient(c client) {

// }
