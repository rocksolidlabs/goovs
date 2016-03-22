package main

import (
	"log"
	"os/exec"

	"github.com/kopwei/goovs"
)

func main() {
	client, err := goovs.GetOVSClient("unix", "")
	if err != nil {
		log.Println(err.Error())
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
	log.Printf("Successfully created the bridge %s", brName)

	err = client.CreateBridge(peerBrName)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("Successfully created the bridge %s", peerBrName)

	err = client.UpdateBridgeController(peerBrName, "tcp:127.0.0.1:6633")
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("Successfully set the bridge controller")

	err = client.CreateInternalPort(brName, internalPortName, internalPortTag)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("Successfully created the internal port %s\n", internalPortName)

	err = client.CreatePatchPort(brName, peerPortNameOnDummyBr, peerPortNameOnPeerBr)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("Successfully created the patch port %s\n", peerPortNameOnDummyBr)

	err = client.CreatePatchPort(peerBrName, peerPortNameOnPeerBr, peerPortNameOnDummyBr)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("Successfully created the patch port %s\n", peerPortNameOnPeerBr)

	// Create a veth pair
	_, err = exec.Command("ip", "link", "add", "vethA", "type", "veth", "peer", "name", "vethB").Output()
	if err != nil {
		log.Fatal(err)
	}

	err = client.CreateVethPort(brName, "vethA", 0)
	if err != nil {
		log.Println(err.Error())
	}
	log.Printf("Successfully created the veth port %s\n", "vethA")

	err = client.UpdatePortTagByName(brName, "vethA", 10)
	if err != nil {
		log.Println(err.Error())
	}
	log.Printf("Successfully updated the veth port %s's tag value to %d\n", "vethA", 10)

	ports, err := client.FindAllPortsOnBridge(brName)
	if err != nil {
		log.Println(err.Error())
		return
	}
	log.Println("All ports on br-dummy are:")
	for _, p := range ports {
		log.Println(p)
	}

	_, err = client.PortExistsOnBridge(brName, brName)
	if err != nil {
		log.Println(err.Error())
	}
	log.Printf("Port %s exists on bridge %s\n", brName, brName)

	// Cleanup

	err = client.DeletePort(brName, brName)
	if err != nil {
		log.Println(err.Error())
	}
	log.Printf("Successfully delete port %s from bridge %s\n", brName, brName)

	err = client.DeleteBridge(brName)
	if err != nil {
		log.Println(err.Error())
	}
	log.Printf("Successfully deleted bridge %s\n", brName)

	err = client.DeleteBridge(peerBrName)
	if err != nil {
		log.Println(err.Error())
	}
	log.Printf("Successfully deleted bridge %s\n", peerBrName)
	_, err = exec.Command("ip", "link", "delete", "vethA").Output()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Successfully removed ip link %s\n", "vethA")

}
