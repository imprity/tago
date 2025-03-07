package main

import (
	"fmt"
	"os"
	"log"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

var ErrLogger *log.Logger = log.New(os.Stderr, "ERROR: ", log.Lshortfile)
var WarnLogger *log.Logger = log.New(os.Stderr, "WARN: ", log.Lshortfile)
var InfoLogger *log.Logger = log.New(os.Stdout, "INFO: ", log.Lshortfile)

func printHelp() {
	println("NOT IMPLEMENTED")
}

func main() {
	args := os.Args[1:]

	if len(args) <= 0 {
		printHelp()
		os.Exit(1)
	}

	var fileName = args[0]

	tagoFiles, err := findTagosForFile(fileName)

	// if we couldn't find any tago files
	// just exit normally
	if len(tagoFiles) <= 0 && err == nil{
		fmt.Printf("could not find any tago files")
		os.Exit(0)
	}

	// report errors
	if err != nil {
		if len(tagoFiles) <= 0 {
			ErrLogger.Fatalf("could not find any tago files: %s", err)
		}else {
			WarnLogger.Printf("error while finding tago files: %s", err)
		}
	}

	fmt.Printf("tago files:\n")
	for _, file := range tagoFiles {
		fmt.Printf("    %s\n", file)
	}
	fmt.Printf("\n")

	// print key values from tago files
	{
		keyValue := make(map[string]string)
		for i := len(tagoFiles) - 1; i >= 0; i-- {
			file := tagoFiles[i]
			fileText, err := os.ReadFile(file)
			if err != nil {
				panic(err)
			}

			kv, err := parseTagoFile(fileText)
			if err != nil {
				panic(err)
			}

			for k, v := range kv {
				keyValue[k] = v
			}
		}

		printKeyValue(keyValue)
	}
}

func getNameAndExt(path string) (string, string) {
	path = filepath.Base(path)
	ext := filepath.Ext(path)
	name := path[0 : len(path)-len(ext)]
	return name, ext
}

func isPathSame(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)

	return a == b
}

func isTagoFile(path string) bool {
	ext := filepath.Ext(path)
	if strings.ToLower(ext) == ".tago" {
		return true
	}
	return false
}

func fileExistsAndRegular(path string) error {
	// check if file exists
	info, err := os.Stat(path)

	if err == nil { // file exists
		mode := info.Mode()
		if !mode.IsRegular() { // but it's not regular
			return fmt.Errorf("%s is not a regular file", path)
		}
		return nil
	}

	return err
}

// returns empty path if it couldn't find any
func findTagosForFile(filePath string) ([]string, error) {
	filePath = filepath.Clean(filePath) // why not

	if err := fileExistsAndRegular(filePath); err != nil {
		return nil, err
	}

	var tagos []string

	// get abs
	fileAbsPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf(
			"could not get absolute path of %s: %w",
			filePath, err,
		)
	}

	fileDir := filepath.Dir(fileAbsPath)
	fileName, fileExt := getNameAndExt(filePath)
	_ = fileExt

	dirents, err := os.ReadDir(fileDir)
	if err != nil {
		return nil, err
	}

	var fileTago string
	var rootTago string

	for _, dirent := range dirents {
		if !dirent.Type().IsRegular() {
			continue
		}

		direntPath := filepath.Join(fileDir, dirent.Name())

		// skip the same file
		if isPathSame(fileAbsPath, direntPath) {
			continue
		}
		// skip none tago file
		if !isTagoFile(dirent.Name()) {
			continue
		}

		name, _ := getNameAndExt(dirent.Name())

		// we found tago file with the same name
		if name == fileName {
			fileTago = direntPath
		}

		// if we found tago.tago file
		if name == "tago" {
			rootTago = direntPath
		}
	}

	if isTagoFile(fileAbsPath) {
		// if file it self is a tago file
		// add it to tagos
		fileTago = fileAbsPath
	}

	if len(fileTago) > 0 {
		tagos = append(tagos, fileTago)
	}
	if len(rootTago) > 0 {
		tagos = append(tagos, rootTago)
	}

	curDir := fileDir
	for {
		prevCurDir := curDir
		curDir = filepath.Dir(prevCurDir)
		if curDir == prevCurDir || len(curDir) == 0 { // we have reached the top
			return tagos, nil
		}

		dirents, err = os.ReadDir(curDir)
		if err != nil {
			return tagos, err
		}

		foundRootTago := false
		for _, dirent := range dirents {
			if !dirent.Type().IsRegular() {
				continue
			}
			direntPath := filepath.Join(curDir, dirent.Name())

			// skip none tago file
			if !isTagoFile(dirent.Name()) {
				continue
			}

			name, _ := getNameAndExt(dirent.Name())

			// we found tago.tago file
			if name == "tago" {
				tagos = append(tagos, direntPath)
				foundRootTago = true
				break
			}
		}

		if !foundRootTago {
			return tagos, nil
		}
	}

	return tagos, nil
}

func parseTagoFile(file []byte) (map[string]string, error) {
	if !utf8.Valid(file) {
		return nil, fmt.Errorf("file is not a valid utf8")
	}

	text := string(file)

	keyValue := make(map[string]string)

	text = strings.ReplaceAll(text, "\r\n", "\n")

	lines := strings.Split(text, "\n")

	var insideMultiline bool = false
	var multiLineKey string = ""
	var multiLineValue = ""

	for _, line := range lines {
		line := strings.TrimSpace(line)

		if insideMultiline {
			if line == "]" {
				keyValue[multiLineKey] = multiLineValue

				insideMultiline = false
				multiLineKey = ""
				multiLineValue = ""
			} else {
				if len(multiLineValue) <= 0 {
					multiLineValue += line
				} else {
					multiLineValue += "\n" + line
				}
			}
		} else {
			if strings.HasPrefix(line, "//") { // skip comments
				continue
			}

			sepAt := strings.Index(line, ":")
			if sepAt >= 0 {
				key := strings.TrimSpace(line[0:sepAt])

				valueLine := line[sepAt+1:]
				valueLine = strings.TrimSpace(valueLine)

				if valueLine == "[" {
					insideMultiline = true
					multiLineKey = key
					multiLineValue = ""
				} else {
					keyValue[key] = valueLine
				}
			}
		}
	}

	return keyValue, nil
}

func printKeyValue(keyValue map[string]string) {
	keys := make([]string, len(keyValue))
	{
		i := 0
		for k := range keyValue {
			keys[i] = k
			i++
		}
	}

	slices.Sort(keys)

	//for k, v := range keyValue {
	for _, k := range keys {
		v := keyValue[k]
		if strings.Index(v, "\n") < 0 {
			fmt.Printf("%s: %s\n", k, v)
		} else {
			fmt.Printf("%s: [\n", k)

			lines := strings.Split(v, "\n")
			for _, line := range lines {
				fmt.Printf("    %s\n", line)
			}

			fmt.Printf("]\n")
		}
		fmt.Printf("\n")
	}
}
