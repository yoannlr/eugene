package main

import (
    "fmt"
    "strings"
    "os"
    "bufio"
    "path/filepath"
)

func handlerMessage(h Handler, msg string) {
    fmt.Print(blockHandler)
    fmt.Println(msg)
}

func handlerStatus(h Handler, msg string) {
    fmt.Print(blockHandler)
    fmt.Println(textCyan + "Handler " + textBold + h.Name + textCyan + " :: " + msg + textReset)
}

func hookMessage(msg string) {
    fmt.Print(blockHook)
    fmt.Println(msg)
}

func handlerExec(cmd string, dryRun bool) bool {
    if dryRun {
        fmt.Println(cmd + textYellow + " (dry run)" + textReset)
        return true
    } else {
        return commandExec(cmd)
    }
}

func handlerExecEntries(h Handler, entries []string, cmd string, dryRun bool) bool {
    if len(entries) < 1 {
        handlerMessage(h, "Skipped, nothing to do")
        return true
    }
    if cmd == "" {
        handlerMessage(h, "Skipped, command undefined")
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
    handlerStatus(h, "sync")
    cmd := h.Sync
    if cmd == "" {
        handlerMessage(h, "Skipped, command undefined")
        return true
    } else {
        return handlerExec(cmd, dryRun)
    }
}

func handlerAdd(h Handler, entries []string, dryRun bool) bool {
    handlerStatus(h, "add")
    return handlerExecEntries(h, entries, h.Add, dryRun)
}

func handlerRemove(h Handler, entries []string, dryRun bool) bool {
    handlerStatus(h, "remove")
    return handlerExecEntries(h, entries, h.Remove, dryRun)
}

func handlerSetup(h Handler, gens string, dryRun bool, repair bool) bool {
    setupFile := filepath.Join(gens, ".setup-" + h.Name)
    if repair || ! fileExists(setupFile) {
        if h.Setup != "" {
            handlerStatus(h, "setup")
            res := handlerExec(h.Setup, dryRun)
            if ! dryRun && res {
                os.Create(setupFile)
            }
            return res
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

func handlerHook(h Handler, step string, dryRun bool) bool {
    if step == "before_switch" && h.HookPre != "" {
        hookMessage(textPurple + "Running hook " + textBold + "before_switch" + textPurple + " for handler " + h.Name + textReset)
        return handlerExec(h.HookPre, dryRun)
    } else if step == "after_switch" && h.HookPost != "" {
        hookMessage(textPurple + "Running hook " + textBold + "after_switch" + textPurple + " for handler " + h.Name + textReset)
        return handlerExec(h.HookPost, dryRun)
    }
    return true
}