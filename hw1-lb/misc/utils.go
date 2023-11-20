package misc

import (
	"fmt"
	"github.com/fatih/color"
	"lb/common"
)

// PrintLBLogo prints out the load balancer logo
func PrintLBLogo() {
	fmt.Println("   _____ _                 _             _      ____  ")
	fmt.Println("  / ____(_)               | |           | |    |  _ \\ ")
	fmt.Println(" | (___  _ _ __ ___  _ __ | | ___ ______| |    | |_) |")
	fmt.Println("  \\___ \\| | '_ ` _ \\| '_ \\| |/ _ \\______| |    |  _ < ")
	fmt.Println("  ____) | | | | | | | |_) | |  __/      | |____| |_) |")
	fmt.Println(" |_____/|_|_| |_| |_| .__/|_|\\___|      |______|____/ ")
	fmt.Println("                    | |                               ")
	fmt.Println("                    |_|                               ")
	fmt.Println("")
	fmt.Println("               Simple Load Balancer - 32190984 Isu Kim")
}

// InitColoredLogs initializes colored log messages, WARN, ERROR, INFO
func InitColoredLogs() {
	common.ColoredWarn = color.New(color.FgHiYellow).Sprint("[WARN]")
	common.ColoredError = color.New(color.FgHiRed).Sprint("[ERROR]")
	common.ColoredInfo = color.New(color.FgHiGreen).Sprintf("[INFO]")
	common.ColorCmdRegister = color.New(color.FgBlue).Sprintf("[REGISTER]")
	common.ColorCmdUnregister = color.New(color.FgRed).Sprintf("[UNREGISTER]")
}

// AreMapsEqual checks if two maps are equal
func AreMapsEqual(map1 map[string]string, map2 map[string]string) bool {
	if len(map1) != len(map2) {
		return false
	}
	for key, value1 := range map1 {
		value2, ok := map2[key]
		if !ok || value1 != value2 {
			return false
		}
	}
	return true
}

// ConvertProtoToString converts a proto as string
func ConvertProtoToString(proto uint8) string {
	switch proto {
	case common.TypeProtoTCP:
		return "tcp"
	case common.TypeProtoUDP:
		return "udp"
	default:
		return "unknown"
	}
}
