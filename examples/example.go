package main

import (
	"fmt"
	"log"
	"os/exec"

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

	err = client.CreateInternalPort(brName, internalPortName, internalPortTag)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Create a veth pair
	_, err = exec.Command("ip", "link", "add", "vethA", "type", "veth", "peer", "name", "vethB").Output()
	if err != nil {
		log.Fatal(err)
	}

	err = client.CreateVethPort(brName, "vethA")
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

	err = client.DeleteBridge(brName)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Cleanup
	_, err = exec.Command("ip", "link", "delete", "vethA").Output()
	if err != nil {
		log.Fatal(err)
	}

}
