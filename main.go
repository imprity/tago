package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

	if tagoFiles, err := findTagosForFile(fileName); err != nil {
		panic(err)
	} else if len(tagoFiles) > 0 {

		fmt.Printf("tago files\n")
		for _, file := range tagoFiles {
			fmt.Printf("    %s\n", file)
		}
		fmt.Printf("\n")

		keyValue := make(map[string]string)

		//for _, file := range tagoFiles {
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
func findTagosForFile(filePath string) ([]string, error) {
	{
		// check if file even exists
		info, err := os.Stat(filePath)
		if err != nil {
			return nil, err
		}

		// check if file is regular
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular file", filePath)
		}
	}

	var tagos []string

	// get abs
	fileAbsPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
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

		// if we find tago file with the same name, that is the tago file
		if name == fileName {
			fileTago = direntPath
		}

		// instead if we root.tago file, that is the tago file
		if name == "root" {
			rootTago = direntPath
		}
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

			if name == "root" {
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
