package goovs

import (
	"testing"
)

func preparingEnv() OvsClient {
	client, _ := GetOVSClient("tcp", "ovs:6640")
	return client
}

func tearDown(client OvsClient) {
	client.Disconnect()
}

func TestGetOVSClient(t *testing.T) {
	// TODO
}

func TestDisconnect(t *testing.T) {
	// TODO
}

func TestTransact(t *testing.T) {
	// TODO
}

func TestUpdateOvsObjCacheByRow(t *testing.T) {
	// TODO
}

func TestRemoveOvsObjCacheByRow(t *testing.T) {
	// TODO
}

func TestPopulateCache(t *testing.T) {
	// TODO
}

func TestGetRootUUID(t *testing.T) {
	// TODO
}
