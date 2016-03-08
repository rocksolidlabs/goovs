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
	err = client.CreateBridge("br-dummy")
	if err != nil {
		fmt.Println(err.Error())
	}
	ports, err := client.FindAllPortsOnBridge("br-dummy")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	for _, p := range ports {
		fmt.Println(p)
	}

	_, err = client.PortExists("br-dummy")
	if err != nil {
		fmt.Println(err.Error())
	}

	err = client.DeleteBridge("br-dummy")
	if err != nil {
		fmt.Println(err.Error())
	}
}
