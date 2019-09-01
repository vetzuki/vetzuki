package main

import "fmt"
import "os"
import "io/ioutil"
import "path/filepath"

func main() {
	if len(os.Args) < 1 {
		os.Exit(1)
	}
	dirName := os.Args[1]
	// list the directory
	fmt.Println("Listing ", dirName)
	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		fmt.Println("error: ", err)
		os.Exit(1)
	}
	for _, f := range files {
		fullPath := filepath.Join(dirName, f.Name())
		if ok, _ := filepath.Match("authorized_keys", f.Name()); ok {
			os.Chown(fullPath, 0, 0)
		}
	}
}
