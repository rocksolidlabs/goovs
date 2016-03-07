package main

import (
	"fmt"

	"github.com/kopwei/goovs"
)

func main() {
	client, err := goovs.GetOVSClient("unix", "")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	ports, err := client.FindAllPortsOnBridge("br-mgmt")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	for _, p := range ports {
		fmt.Println(p)
	}
}
