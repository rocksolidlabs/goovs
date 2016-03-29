package goovs

import (
    "testing"
)

func preparingEnv() OvsClient{
    client, _ := GetOVSClient("tcp", "ovs:6640")
    return client
}

func tearDown(client OvsClient) {
    client.Disconnect()
}