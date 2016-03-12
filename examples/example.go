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
	peerPortNameOnDummyBr := "to-peer"

	peerBrName := "br-peer"
	peerPortNameOnPeerBr := "to-dummy"

	err = client.CreateBridge(brName)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = client.CreateBridge(peerBrName)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = client.CreateInternalPort(brName, internalPortName, internalPortTag)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Printf("Successfully created the internal port %s\n", internalPortName)

	err = client.CreatePatchPort(brName, peerPortNameOnDummyBr, peerPortNameOnPeerBr)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Printf("Successfully created the patch port %s\n", peerPortNameOnDummyBr)

	err = client.CreatePatchPort(peerBrName, peerPortNameOnPeerBr, peerPortNameOnDummyBr)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Printf("Successfully created the patch port %s\n", peerPortNameOnPeerBr)

	// Create a veth pair
	_, err = exec.Command("ip", "link", "add", "vethA", "type", "veth", "peer", "name", "vethB").Output()
	if err != nil {
		log.Fatal(err)
	}

	err = client.CreateVethPort(brName, "vethA", 0)
	if err != nil {
		fmt.Println(err.Error())
	}

	ports, err := client.FindAllPortUUIDsOnBridge(brName)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("All ports on br-dummy are:")
	for _, p := range ports {
		fmt.Println(p)
	}

	_, err = client.PortExistsOnBridge(brName, brName)
	if err != nil {
		fmt.Println(err.Error())
	}

	// Cleanup

	err = client.DeletePort(brName, brName)
	if err != nil {
		fmt.Println(err.Error())
	}

	err = client.DeleteBridge(brName)
	if err != nil {
		fmt.Println(err.Error())
	}

	err = client.DeleteBridge(peerBrName)
	if err != nil {
		fmt.Println(err.Error())
	}

	_, err = exec.Command("ip", "link", "delete", "vethA").Output()
	if err != nil {
		log.Fatal(err)
	}

}
