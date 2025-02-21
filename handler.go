package main

import (
    "fmt"
    "strings"
    "os"
    "bufio"
    "path/filepath"
)

func handlerExec(cmd string, dryRun bool) bool {
    logCommand(cmd, dryRun)
    if dryRun {
        return true
    } else {
        return commandExec(cmd)
    }
}

func handlerExecEntries(h Handler, entries []string, cmd string, dryRun bool) bool {
    if len(entries) < 1 {
        logHandler(h.Name, ">> Skipped, nothing to do")
        return true
    }
    if cmd == "" {
        logHandler(h.Name, ">> Skipped, command undefined")
        return true
    }
    if h.Multiple {
        return handlerExec(fmt.Sprintf(cmd, strings.Join(entries, " ")), dryRun)
    } else {
        for _, entry := range entries {
            if ! handlerExec(fmt.Sprintf(cmd, entry), dryRun) {
                return false
            }
        }
        return true
    }
}

func handlerSync(h Handler, dryRun bool) bool {
    logHandler(h.Name, "Synchronizing")
    cmd := h.Sync
    if cmd == "" {
        logHandler(h.Name, ">> Skipped, command undefined")
        return true
    } else {
        return handlerExec(cmd, dryRun)
    }
}

func handlerAdd(h Handler, entries []string, dryRun bool) bool {
    logHandler(h.Name, "Adding new entries")
    return handlerExecEntries(h, entries, h.Add, dryRun)
}

func handlerRemove(h Handler, entries []string, dryRun bool) bool {
    logHandler(h.Name, "Removing previous entries")
    return handlerExecEntries(h, entries, h.Remove, dryRun)
}

func handlerShouldRun(h Handler) bool {
    if h.RunIf == "" {
        return true
    }
    return commandExec(h.RunIf)
}

func handlerSetup(h Handler, gens string, dryRun bool, repair bool) bool {
    setupFile := filepath.Join(gens, ".setup-" + h.Name)
    if repair || ! fileExists(setupFile) {
        if h.Setup != nil {
            logHandler(h.Name, "Setting up")

            success := false
            for _, setup := range h.Setup {
                if commandExec(setup.When) {
                    success = handlerExec(setup.Run, dryRun)
                    break
                }
            }

            if ! success {
                return false
            }

            if ! dryRun {
                os.Create(setupFile)
            }
        }
    }
    return true
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

func handlerPreSwitch(h Handler, dryRun bool) bool {
    if h.HookPre != "" {
        logHandler(h.Name, "Running pre-switch command")
        return handlerExec(h.HookPre, dryRun)
    }
    return true
}

func handlerPostSwitch(h Handler, dryRun bool) bool {
    if h.HookPost != "" {
        logHandler(h.Name, "Running post-switch command")
        return handlerExec(h.HookPost, dryRun)
    }
    return true
}

func handlerUpgrade(h Handler, dryRun bool) bool {
    logHandler(h.Name, "Upgrading")
    if h.Upgrade == "" {
        logHandler(h.Name, "Command undefined")
        return true
    } else {
        return handlerExec(h.Upgrade, dryRun)
    }
}