package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

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

	if tagoFile, err := findTagoForFile(fileName); err != nil {
		panic(err)
	} else if len(tagoFile) > 0{
		fmt.Printf("tagoFile : %s\n", tagoFile)
		fileText, err := os.ReadFile(tagoFile)
		if err != nil {
			panic(err)
		}

		keyValue, err := parseTagoFile(fileText)
		if err != nil {
			panic(err)
		}

		for k, v := range keyValue {
			fmt.Printf("%s : %s\n", k, v)
		}
	} else {
		fmt.Printf("couldn't find tago file for %s\n", fileName)
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

// returns empty path if it couldn't find any
func findTagoForFile(filePath string) (string, error) {
	{
		// check if file even exists
		info, err := os.Stat(filePath)
		if err != nil {
			return "", err
		}

		// check if file is regular
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("%s is not a regular file", filePath)
		}
	}

	// get abs
	fileAbsPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	fileDir := filepath.Dir(fileAbsPath)
	fileName, fileExt := getNameAndExt(filePath)
	_ = fileExt

	dirents, err := os.ReadDir(fileDir)
	if err != nil {
		return "", err
	}

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

		// if we find tago file with the same name, that is the tago file
		if name == fileName {
			return direntPath, nil
		}
		// instead if we root.tago file, that is the tago file
		if name == "root" {
			return direntPath, nil
		}
	}

	curDir := fileDir
	for {
		prevCurDir := curDir
		curDir = filepath.Dir(prevCurDir)
		if curDir == prevCurDir || len(curDir) == 0 { // we have reached the top
			return "", nil
		}

		dirents, err = os.ReadDir(curDir)
		if err != nil {
			return "", err
		}

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

			if name == "root" {
				return direntPath, nil
			}
		}
	}

	return "", nil
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
