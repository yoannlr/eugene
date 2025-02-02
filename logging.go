package main

import (
	"fmt"
)

const textReset = "\033[0m"
const textCyan = "\033[0;36m"
const textGreen = "\033[0;32m"
const textRed = "\033[0;31m"
const textYellow = "\033[0;33m"
const textBold = "\033[1m"

const dryRunIndicator = textYellow + "(dry-run)" + textReset

func logInfo(msg string) {
	fmt.Println(textGreen + "[info] " + textReset + msg + textReset)
}

func logUsage(msg string) {
	fmt.Println(textRed + "[usage] " + textReset + msg + textReset)
}

func logError(msg string) {
	fmt.Println(textRed + "[error] " + msg + textReset)
}

func logHandler(name string, msg string) {
	fmt.Println(textCyan + "[handler " + name + "] " + textReset + msg + textReset)
}

func logAction(msg string, dryRun bool) {
	if dryRun {
		logInfo(msg + textReset + " " + dryRunIndicator)
	} else {
		logInfo(msg + textReset)
	}
}

func logCommand(cmd string, dryRun bool) {
	fmt.Println("$ " + cmd)
}