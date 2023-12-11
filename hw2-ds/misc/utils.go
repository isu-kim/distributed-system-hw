package misc

import (
	"fmt"
	"github.com/fatih/color"
)

func PrintLogo() {
	fmt.Println("                    __  ")
	fmt.Println("   ________  ____  / /_ ")
	fmt.Println("  / ___/ _ \\/ __ \\/ __ \\")
	fmt.Println(" (__  )  __/ /_/ / / / /")
	fmt.Println("/____/\\___/ .___/_/ /_/ ")
	fmt.Println("         /_/            ")
	fmt.Println("Simple Distributed Storage")
	fmt.Println("        32190984 - Isu Kim")
}

// InitColoredLogs initializes colored log messages, WARN, ERROR, INFO
func InitColoredLogs() {
	ColoredClient = color.New(color.FgHiGreen).Sprint("CLIENT")
	ColoredReplica = color.New(color.FgHiYellow).Sprint("REPLICA")
}
