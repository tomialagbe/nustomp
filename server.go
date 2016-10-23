package nustomp

// Server represents an instance of the STOMP server
type Server struct {
	port int
}

// NewServer creates a new STOMP server instance
func NewServer(port int) *Server {
	return &Server{port}
}

// Start the server
func (s *Server) Start() {

}
