// Package mbserver implments a Modbus server (slave).
package mbserver

import (
	"io"
	"net"

	"github.com/goburrow/serial"
)

// Server is a Modbus slave with allocated memory for discrete inputs, coils, etc.
type Server struct {
	// Debug enables more verbose messaging.
	Debug            bool
	listeners        []net.Listener
	ports            []serial.Port
	RequestChan      chan *Request
	function         [256](func(*Server, Framer) ([]byte, *Exception))
	DiscreteInputs   []byte
	Coils            []byte
	HoldingRegisters []uint16
	InputRegisters   []uint16
}

// Request contains the connection and Modbus Frame.
type Request struct {
	Conn  io.ReadWriteCloser
	Frame Framer
}

// NewServer creates a new Modbus server (slave).
func NewServer() *Server {
	s := &Server{}

	// Allocate Modbus memory maps.
	s.DiscreteInputs = make([]byte, 65536)
	s.Coils = make([]byte, 65536)
	s.HoldingRegisters = make([]uint16, 65536)
	s.InputRegisters = make([]uint16, 65536)

	// Add default functions.
	s.function[1] = ReadCoils
	s.function[2] = ReadDiscreteInputs
	s.function[3] = ReadHoldingRegisters
	s.function[4] = ReadInputRegisters
	s.function[5] = WriteSingleCoil
	s.function[6] = WriteHoldingRegister
	s.function[15] = WriteMultipleCoils
	s.function[16] = WriteHoldingRegisters

	//s.RequestChan = make(chan *Request)
	//go s.handler()

	return s
}

// RegisterFunctionHandler override the default behavior for a given Modbus function.
func (s *Server) RegisterFunctionHandler(funcCode uint8, function func(*Server, Framer) ([]byte, *Exception)) {
	s.function[funcCode] = function
}

func (s *Server) Handle(request *Request) Framer {
	var exception *Exception
	var data []byte

	response := request.Frame.Copy()

	function := request.Frame.GetFunction()
	if s.function[function] != nil {
		data, exception = s.function[function](s, request.Frame)
		response.SetData(data)
	} else {
		exception = &IllegalFunction
	}

	if exception != &Success {
		response.SetException(exception)
	}

	return response
}

// All requests are handled synchronously to prevent modbus memory corruption.
func (s *Server) Handler() {
	for {
		request := <-s.RequestChan
		response := s.Handle(request)
		request.Conn.Write(response.Bytes())
	}
}

// Close stops listening to TCP/IP ports and closes serial ports.
func (s *Server) Close() {
	for _, listen := range s.listeners {
		listen.Close()
	}
	for _, port := range s.ports {
		port.Close()
	}
}
