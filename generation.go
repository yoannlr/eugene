package main

import (
	"os"
	"path/filepath"
	"strconv"
	"slices"
	"bufio"
)

func genCreate(gens string, num int, comment string) string {
	thisGenDir := filepath.Join(gens, strconv.Itoa(num))
	os.Mkdir(thisGenDir, os.ModePerm)
	if comment != "" {
		commentFile, _ := os.Create(filepath.Join(thisGenDir, "_comment"))
		commentFile.WriteString(comment + "\n")
		commentFile.Close()
	}
	return thisGenDir
}

func genTag(gens string, num int, tag string) {
	currentDir, _ := os.Getwd()
	os.Chdir(gens)
	os.Remove(tag)
	os.Symlink(strconv.Itoa(num), tag)
	os.Chdir(currentDir)
}

func genGetTagged(gens string, tag string) int {
	g, _ := os.Readlink(filepath.Join(gens, tag))
	num, _ := strconv.Atoi(g)
	return num
}

func genSetCurrent(gens string, num int) {
	genTag(gens, num, "current")
}

func genSetLatest(gens string, num int) {
	genTag(gens, num, "latest")
}

func genGetCurrent(gens string) int {
	return genGetTagged(gens, "current")
}

func genGetLatest(gens string) int {
	return genGetTagged(gens, "latest")
}

func genExists(gens string, num int) bool {
	return fileExists(filepath.Join(gens, strconv.Itoa(num)))
}

func genDiff(gens string, a int, b int, h Handler) ([]string, []string) {
	var add []string
	var remove []string

	inGenA := handlerGetEntries(gens, a, h)
	inGenB := handlerGetEntries(gens, b, h)

	// in a and not in b => remove
	for _, entry := range inGenA {
		if !slices.Contains(inGenB, entry) {
			remove = append(remove, entry)
		}
	}

	// in b and not in a => add
	for _, entry := range inGenB {
		if !slices.Contains(inGenA, entry) {
			add = append(add, entry)
		}
	}

	return add, remove
}

func genParse(gens string, arg string) int {
	if arg == "current" {
		return genGetCurrent(gens)
	} else if arg == "latest" {
		return genGetLatest(gens)
	} else {
		num, err := strconv.Atoi(arg)
		if err != nil || ! genExists(gens, num) {
			return -1
		}
		return num
	}
}

func genGetComment(gens string, num int) string {
	if genExists(gens, num) {
		commentFilePath := filepath.Join(genGetPath(gens, num), "_comment")
		if fileExists(commentFilePath) {
			commentFile, _ := os.Open(commentFilePath)
			scanner := bufio.NewScanner(commentFile)
			comment := ""
			for scanner.Scan() {
				comment = scanner.Text()
				// on ne lit que la premiere ligne
				break
			}
			return string(comment)
		}
	}
	return ""
}

func genDelete(gens string, num int) bool {
	if num == 0 {
		eugeneError("Deleting generation 0 is forbidden")
		return false
	}

	if num == genGetCurrent(gens) {
		eugeneError("Deleting the current generation is forbidden")
		return false
	}

	if ! genExists(gens, num) {
		eugeneError("Generation " + strconv.Itoa(num) + " does not exist")
		return false
	}
	
	if num == genGetLatest(gens) {
		prevGen := num - 1
		for ! genExists(gens, prevGen) {
			prevGen = prevGen - 1
			// on tombe sur la generation 0 au bout d'un moment
		}
		genSetLatest(gens, prevGen)
		eugeneMessage("The latest generation is now " + strconv.Itoa(prevGen))
	}

	os.RemoveAll(genGetPath(gens, num))
	eugeneMessage("Deleted generation " + strconv.Itoa(num))

	return true
}

func genGetPath(gens string, num int) string {
	if genExists(gens, num) {
		return filepath.Join(gens, strconv.Itoa(num))
	}
	return ""
}

func genSwitch(config Config, gens string, targetGen int, dryRun bool) bool {
	// todo handlerAdd/Remove renvoie un booleen selon le retour de la commande
	for _, h := range config.Handlers {
		handlerSetup(h, gens, dryRun)
		handlerHook(h, "before_switch", dryRun)
		handlerSync(h, dryRun)
		add, remove := genDiff(gens, genGetCurrent(gens), targetGen, h)
		handlerRemove(h, remove, dryRun)
		handlerAdd(h, add, dryRun)
		handlerHook(h, "after_switch", dryRun)
		eugeneMessage("")
	}

	if ! dryRun {
		genSetCurrent(gens, targetGen)
	}

	return true
}