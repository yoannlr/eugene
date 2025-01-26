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
	"crypto/sha256"
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
	eugeneMessage("System hostname is " + hostname)

	hasDiff := true

	genHash := sha256.New()

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
				file, _ := os.Open(filepath.Join(repo, f))
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
				// hash a partir des entries dans l'ordre => reproductible
				genHash.Write([]byte(p))
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

	hashFile, _ := os.Create(filepath.Join(newGenDir, "_hash"))
	hashFile.WriteString(fmt.Sprintf("%x\n", genHash.Sum(nil)))
	hashFile.Close()

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

func doAlign(gens string, dryRun bool) {
	allGens := genGetAll(gens)
	currentGen := genGetCurrent(gens)
	latestGen := genGetLatest(gens)
	for i, g := range allGens {
		if g != i {
			eugeneMessage(strconv.Itoa(g) + " -> " + strconv.Itoa(i))
			if ! dryRun {
				genRenumber(gens, g, i)
			}
			if g == currentGen {
				if ! dryRun {
					genSetCurrent(gens, i)
				}
				eugeneMessage("current -> " + strconv.Itoa(i))
			}
			if g == latestGen {
				if ! dryRun {
					genSetLatest(gens, i)
				}
				eugeneMessage("latest -> " + strconv.Itoa(i))
			}
		}
	}
	if dryRun {
		eugeneMessage("Generations aligned " + textYellow + "(dry-run)" + textReset)
	} else {
		eugeneMessage("Generations aligned")
	}
}

func doDeleteDups(gens string, dryRun bool) {
	allGens := genGetAll(gens)
	//var hashesToGens map[string][]int // n'alloue pas la map
	hashesToGens := make(map[string][]int)
	for _, g := range allGens {
		hash := genGetHash(gens, g)
		hashesToGens[hash] = append(hashesToGens[hash], g)
	}
	dryCurrent := genGetCurrent(gens)
	dryLatest := genGetLatest(gens)
	for _, gns := range hashesToGens {
		if len(gns) > 1 {
			slices.Sort(gns)
			keepGen := gns[len(gns) - 1]
			for i := 0; i < len(gns) - 1; i++ {
				currentGen := dryCurrent
				latestGen := dryLatest
				if dryRun {
					eugeneMessage("Deleted generation " + strconv.Itoa(gns[i]) + " because it's identical to generation " + strconv.Itoa(keepGen) + textYellow + " (dry-run)" + textReset)
				} else {
					currentGen = genGetCurrent(gens)
					latestGen = genGetLatest(gens)
					genDelete(gens, gns[i])
					eugeneMessage("Deleted generation " + strconv.Itoa(gns[i]) + " because it's identical to generation " + strconv.Itoa(keepGen))
				}
				if gns[i] == currentGen {
					if ! dryRun {
						genSetCurrent(gens, keepGen)
					} else {
						dryCurrent = keepGen
					}
					eugeneMessage("current -> " + strconv.Itoa(keepGen))
				}
				if gns[i] == latestGen {
					if ! dryRun {
						genSetLatest(gens, keepGen)
					} else {
						dryLatest = keepGen
					}
					eugeneMessage("latest -> " + strconv.Itoa(keepGen))
				}
			}
		}
	}
}

func doRollback(config Config, gens string, n int, dryRun bool) bool {
	allGens := genGetAll(gens)
	slices.Sort(allGens)
	slices.Reverse(allGens)
    currentGen := genGetCurrent(gens)
    currentIndex := -1
    for i, g := range allGens {
        if g == currentGen {
            currentIndex = i
            break
        }
    }
    if currentIndex + n >= len(allGens) {
        eugeneError("Not enough generations to rollback " + strconv.Itoa(n) + " generations ago")
        return false
    }
    target := allGens[currentIndex + n]
    eugeneMessage("Rolling back to generation " + strconv.Itoa(target))
    genSwitch(config, gens, target, dryRun)
    return true
}