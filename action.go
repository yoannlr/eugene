package main

import (
	"strings"
	"regexp"
	"os"
	"fmt"
	"bufio"
	"slices"
	"path/filepath"
	"strconv"
)

func doBuild(args []string, repo string, gens string, config Config) bool {
	newGen := genGetLatest(gens) + 1
	comment := ""
	if len(args) > 1 {
		comment = strings.Join(args[2:], " ")
	}
	newGenDir := genCreate(gens, newGen, comment)

	commentRegex, _ := regexp.Compile("^#")
	emptyLineRegex, _ := regexp.Compile("^$")
	hostname, _ := os.Hostname()

	hasDiff := true

	for _, h := range config.Handlers {
		handlerStatus(h, "build")

		// trouver les fichiers a inclure
		filesRegex, _ := regexp.Compile(fmt.Sprintf("^%s.*$", h.Name))
		filesRegexHostname, _ := regexp.Compile(fmt.Sprintf("^%s_%s.*$", hostname, h.Name))
		repoFiles, _ := os.ReadDir(repo)

		var handlerFiles []string
		for _, f := range repoFiles {
			if filesRegex.MatchString(f.Name()) || filesRegexHostname.MatchString(f.Name()) {
				eugeneMessage("+ include file " + f.Name())
				handlerFiles = append(handlerFiles, f.Name())
			}
		}

		// generer le resultat
		if len(handlerFiles) > 0 {
			var handlerEntries []string
			for _, f := range handlerFiles {
				file, _ := os.Open(f)
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := scanner.Text()
					if ! emptyLineRegex.MatchString(line) && ! commentRegex.MatchString(line) {
						handlerEntries = append(handlerEntries, line)
					}
				}
				file.Close()
			}

			slices.Sort(handlerEntries) // sort
			handlerEntries = slices.Compact(handlerEntries) // uniq

			handlerResult, _ := os.Create(filepath.Join(newGenDir, h.Name))
			for _, p := range handlerEntries {
				handlerResult.WriteString(p + "\n")
			}
			handlerResult.Close()

			if ! hasDiff {
				add, remove := genDiff(gens, genGetLatest(gens), newGen, h)
				if len(add) > 0 || len(remove) > 0 {
					hasDiff = true
				}
			}
		}
	}

	if hasDiff {
		genSetLatest(gens, newGen)
		eugeneMessage(textGreen + "Done building generation " + strconv.Itoa(newGen) + textReset)
		return true
	} else {
		genDelete(gens, newGen)
		eugeneMessage("No difference with the latest generation, build removed")
		return false
	}
}

func doSwitch(config Config, gens string, targetGen int, dryRun bool) bool {
	if genSwitch(config, gens, targetGen, dryRun) {
		if dryRun {
			eugeneMessage("Switched to generation " + strconv.Itoa(targetGen) + textYellow + " (dry run)" + textReset)
		} else {
			eugeneMessage("Switched to generation " + strconv.Itoa(targetGen))
		}
		return true
	} else {
		eugeneError("Switch to generation " + strconv.Itoa(targetGen) + " failed")
		return false
	}
}

func doAlign(gens string) {
	allGens := genGetAll(gens)
	currentGen := genGetCurrent(gens)
	latestGen := genGetLatest(gens)
	for i, g := range allGens {
		if g != i {
			eugeneMessage(strconv.Itoa(g) + " -> " + strconv.Itoa(i))
			genRenumber(gens, g, i)
			if g == currentGen {
				genSetCurrent(gens, i)
				eugeneMessage("current -> " + strconv.Itoa(i))
			}
			if g == latestGen {
				genSetLatest(gens, i)
				eugeneMessage("latest -> " + strconv.Itoa(i))
			}
		}
	}
	eugeneMessage("Generations aligned")
}