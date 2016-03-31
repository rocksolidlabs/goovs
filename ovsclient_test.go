package goovs

import (
	"testing"

	"github.com/kopwei/libovsdb"
)

func preparingEnv() *ovsClient {
	client, _ := GetOVSClient("tcp", "ovs:6640")
	return client.(*ovsClient)
}

func tearDown(client OvsClient) {
	client.Disconnect()
}

func TestGetOVSClient(t *testing.T) {
	client, _ := GetOVSClient("tcp", "ovs:6640")
	if client == nil {
		t.Fatal("Get ovs client test failed")
	}
}

func TestDisconnect(t *testing.T) {
	client := preparingEnv()
	client.Disconnect()
}

func TestTransact(t *testing.T) {
	client := preparingEnv()
	defer tearDown(client)
	// Empty transaction
	operations := []libovsdb.Operation{}
	err := client.transact(operations, "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateOvsObjCacheByRow(t *testing.T) {
	//client := preparingEnv()
	//defer tearDown(client)
	err := client.updateOvsObjCacheByRow("Bridge", "abcde12345", &libovsdb.Row{})
	if err != nil {
		t.Fatal(err)
	}
	err = client.updateOvsObjCacheByRow("Port", "abcde12345", &libovsdb.Row{})
	if err != nil {
		t.Fatal(err)
	}
	err = client.updateOvsObjCacheByRow("Interface", "abcde12345", &libovsdb.Row{})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRemoveOvsObjCacheByRow(t *testing.T) {
	//client := preparingEnv()
	//defer tearDown(client)
	client.updateOvsObjCacheByRow("Bridge", "abcde12345", &libovsdb.Row{})
	client.updateOvsObjCacheByRow("Port", "abcde12345", &libovsdb.Row{})
	client.updateOvsObjCacheByRow("Interface", "abcde12345", &libovsdb.Row{})
	err := client.removeOvsObjCacheByRow("Bridge", "abcde12345")
	if err != nil {
		t.Fatal(err)
	}
	err = client.removeOvsObjCacheByRow("Port", "abcde12345")
	if err != nil {
		t.Fatal(err)
	}
	err = client.removeOvsObjCacheByRow("Interface", "abcde12345")
	if err != nil {
		t.Fatal(err)
	}
}

func TestPopulateCache(t *testing.T) {
	//updates := libovsdb.TableUpdates{}
}

func TestGetRootUUID(t *testing.T) {
	tmpCache := cache
	cache = make(map[string]map[string]libovsdb.Row, 0)
	cache[defaultOvsDB] = make(map[string]libovsdb.Row)
	cache[defaultOvsDB]["abcdef12345"] = libovsdb.Row{}
	uuid := getRootUUID()
	if uuid != "abcdef12345" {
		t.Fatalf("The uuid %s is incorrect", uuid)
	}
	cache = tmpCache
}
