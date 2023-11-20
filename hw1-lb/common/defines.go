package common

// Defne colored outputs
var (
	ColoredWarn  string
	ColoredError string
	ColoredInfo  string
)

const (
	TypeProtoTCP = 1
	TypeProtoUDP = 2
)

// Predefined types of commands
const (
	CmdTypeRegister   = 1
	CmdTypeUnregister = 2
	CmdTypeHello      = 3
)
