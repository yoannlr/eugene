package main

import (
	"fmt"
	"strings"
	"os"
	"bufio"
	"path/filepath"
)

func handlerMessage(h Handler, msg string) {
	eugeneMessage(msg)
}

func handlerStatus(h Handler, msg string) {
	eugeneMessage(colorCyan + "Handler " + h.Name + " :: " + msg + colorReset)
}

func handlerExec(cmd string, dryRun bool) bool {
	if dryRun {
		fmt.Println(cmd + colorYellow + " (dry run)" + colorReset)
		return true
	} else {
		return commandExec(cmd)
	}
}

func handlerExecEntries(h Handler, entries []string, cmd string, dryRun bool) {
	if len(entries) < 1 {
		handlerMessage(h, "Nothing to do")
		return
	}
	if cmd == "" {
		handlerMessage(h, "Command undefined")
		return
	}
	if h.Multiple {
		handlerExec(fmt.Sprintf(cmd, strings.Join(entries, " ")), dryRun)
	} else {
		for _, entry := range entries {
			handlerExec(fmt.Sprintf(cmd, entry), dryRun)
		}
	}
}

func handlerSync(h Handler, dryRun bool) {
	handlerStatus(h, "sync")
	cmd := h.Sync
	if cmd == "" {
		handlerMessage(h, "Command undefined")
	} else {
		handlerExec(cmd, dryRun)
	}
}

func handlerAdd(h Handler, entries []string, dryRun bool) {
	handlerStatus(h, "add")
	handlerExecEntries(h, entries, h.Add, dryRun)
}

func handlerRemove(h Handler, entries []string, dryRun bool) {
	handlerStatus(h, "remove")
	handlerExecEntries(h, entries, h.Remove, dryRun)
}

func handlerSetup(h Handler, gens string, dryRun bool) {
	setupFile := filepath.Join(gens, ".setup-" + h.Name)
	if ! fileExists(setupFile) {
		if h.Setup != "" {
			handlerStatus(h, "setup")
			handlerExec(h.Setup, dryRun)
			if ! dryRun {
				os.Create(setupFile)
			}
		}
	}
}

func handlerGetEntries(gens string, num int, h Handler) []string {
	var entries []string

	handlerPath := filepath.Join(genGetPath(gens, num), h.Name)
	if fileExists(handlerPath) {
		handlerFile, _ := os.Open(handlerPath)
		scanner := bufio.NewScanner(handlerFile)
		for scanner.Scan() {
			entries = append(entries, scanner.Text())
		}
		handlerFile.Close()
	}

	return entries
}

func handlerHook(h Handler, step string, dryRun bool) bool {
	if step == "before_switch" && h.HookPre != "" {
		eugeneMessage(colorPurple + "Running hook before_switch for handler " + h.Name + colorReset)
		return handlerExec(h.HookPre, dryRun)
	} else if step == "after_switch" && h.HookPost != "" {
		eugeneMessage(colorPurple + "Running hook after_switch for handler " + h.Name + colorReset)
		return handlerExec(h.HookPost, dryRun)
	}
	return false
}