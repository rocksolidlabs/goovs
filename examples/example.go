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
	brName := "br-dummy"
	internalPortName := "port-dummy"
	internalPortTag := 10

	err = client.CreateBridge(brName)
	if err != nil {
		fmt.Println(err.Error())
	}
	ports, err := client.FindAllPortsOnBridge(brName)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("All ports on br-dummy are:")
	for _, p := range ports {
		fmt.Println(p)
	}

	_, err = client.PortExists(brName)
	if err != nil {
		fmt.Println(err.Error())
	}

	err = client.DeletePort(brName, brName)
	if err != nil {
		fmt.Println(err.Error())
	}

	err = client.CreateInternalPort(brName, internalPortName, internalPortTag)
	if err != nil {
		fmt.Println(err.Error())
	}

	err = client.DeleteBridge(brName)
	if err != nil {
		fmt.Println(err.Error())
	}

}
