# goovs
The go library used to communicate with Openvswitch and manage Openvswitch bridges and ports

# Description
This is the high level library for Go programs to interact with Openvswitch.

### Features:
* Can connect to local Openvswitch db via Unix socket or remote tcp socket
* Can be used to create/delete bridges, create/delete various types of ports, e.g. Internal port, Veth port or Patch port

### Features which are not ready:
* Openvswitch flow management are not in place since they are not handled by ovsdb

### Environment
Code is tested with Openvswitch 2.0 on Ubuntu14.04

# Examples
### Connect to local Openvswitch DB
```go
	client, err := goovs.GetOVSClient("unix", "")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
```

### Create a bridge
```go
	err := client.CreateBridge(brName)
	if err != nil {
		fmt.Println(err.Error())
	}
```

### Create internal port on the bridge
```go
	err := client.CreateInternalPort(brName, internalPortName, internalPortTag)
	if err != nil {
		fmt.Println(err.Error())
	}
```