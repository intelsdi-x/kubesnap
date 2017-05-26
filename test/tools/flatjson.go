package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"bufio"
	"./util"
)

func main() {

	dontpanic := func(err error) {
		if err != nil {
			// okay, panic
			panic(err)
		}
	}
	allb, err := ioutil.ReadAll(bufio.NewReader(os.Stdin))
	dontpanic(err)
	walker, err := util.NewJsonWalker(string(allb))
	dontpanic(err)
	walker.Walk("/", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			fmt.Fprintf(os.Stdout, "%v,%v\n", path, info.Sys())
		}
		return nil
	})
	//var obj interface{}
	//err = json.Unmarshal(allb, &obj)
	//dontpanic(err)
	//allb, err = json.MarshalIndent(obj, "", "  ")
	//dontpanic(err)
	//fmt.Println(string(allb))
	return
}

