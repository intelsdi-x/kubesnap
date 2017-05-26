package main

import (
	"fmt"
	"encoding/json"
	"os"
	"io/ioutil"
	"bufio"
)

func main() {
	dontpanic := func(err error) {
		if err != nil {
			// okay, panic now
			panic(err)
		}
	}
	allb, err := ioutil.ReadAll(bufio.NewReader(os.Stdin))
	dontpanic(err)
	var obj interface{}
	err = json.Unmarshal(allb, &obj)
	dontpanic(err)
	allb, err = json.MarshalIndent(obj, "", "  ") 
	dontpanic(err)
	fmt.Println(string(allb))
	return
}

