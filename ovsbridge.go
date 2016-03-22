package goovs

import (
	"fmt"

	"github.com/kopwei/libovsdb"
)

// CreateBridge is used to create a ovs bridge
func (client *ovsClient) CreateBridge(brname string) error {
	bridgeUpdateLock.Lock()
	defer bridgeUpdateLock.Unlock()
	bridgeExists, err := client.BridgeExists(brname)
	if err != nil {
		return fmt.Errorf("Failed to retrieve the bridge info")
	} else if bridgeExists {
		return nil
	}

	namedBridgeUUID := "gobridge"
	namedPortUUID := "goport"
	namedInterfaceUUID := "gointerface"

	// intf row to insert
	intf := make(map[string]interface{})
	intf["name"] = brname
	intf["type"] = `internal`

	insertInterfaceOp := libovsdb.Operation{
		Op:       "insert",
		Table:    "Interface",
		Row:      intf,
		UUIDName: namedInterfaceUUID,
	}

	// port row to insert
	port := make(map[string]interface{})
	port["name"] = brname
	port["interfaces"] = libovsdb.UUID{GoUuid: namedInterfaceUUID}

	insertPortOp := libovsdb.Operation{
		Op:       insertOperation,
		Table:    portTableName,
		Row:      port,
		UUIDName: namedPortUUID,
	}

	// bridge row to insert
	bridge := make(map[string]interface{})
	bridge["name"] = brname
	bridge["stp_enable"] = false
	bridge["ports"] = libovsdb.UUID{GoUuid: namedPortUUID}

	// simple insert operation
	insertBridgeOp := libovsdb.Operation{
		Op:       insertOperation,
		Table:    bridgeTableName,
		Row:      bridge,
		UUIDName: namedBridgeUUID,
	}

	// Inserting a Bridge row in Bridge table requires mutating the open_vswitch table
	mutateUUID := []libovsdb.UUID{libovsdb.UUID{GoUuid: namedBridgeUUID}}
	mutateSet, _ := libovsdb.NewOvsSet(mutateUUID)
	mutation := libovsdb.NewMutation("bridges", insertOperation, mutateSet)
	condition := libovsdb.NewCondition("_uuid", "==", libovsdb.UUID{GoUuid: getRootUUID()})

	// simple mutate operation
	mutateOp := libovsdb.Operation{
		Op:        mutateOperation,
		Table:     ovsTableName,
		Mutations: []interface{}{mutation},
		Where:     []interface{}{condition},
	}

	operations := []libovsdb.Operation{insertInterfaceOp, insertPortOp, insertBridgeOp, mutateOp}
	return client.transact(operations, "create bridge")
}

// DeleteBridge is used to delete a ovs bridge
func (client *ovsClient) DeleteBridge(brname string) error {
	bridgeUpdateLock.Lock()
	defer bridgeUpdateLock.Unlock()
	bridgeExists, err := client.BridgeExists(brname)
	if err != nil {
		return fmt.Errorf("Failed to retrieve the bridge info")
	} else if !bridgeExists {
		return nil
	}
	bridgeUUID, err := client.getBridgeUUIDByName(brname)
	if err != nil {
		return err
	}

	delBridgeCondition := libovsdb.NewCondition("name", "==", brname)
	deleteOp := libovsdb.Operation{
		Op:    deleteOperation,
		Table: bridgeTableName,
		Where: []interface{}{delBridgeCondition},
	}

	// Deleting a Bridge row in Bridge table requires mutating the open_vswitch table
	mutateUUID := []libovsdb.UUID{libovsdb.UUID{GoUuid: bridgeUUID}}
	mutateSet, _ := libovsdb.NewOvsSet(mutateUUID)
	mutation := libovsdb.NewMutation("bridges", deleteOperation, mutateSet)
	condition := libovsdb.NewCondition("_uuid", "==", libovsdb.UUID{GoUuid: getRootUUID()})

	// simple mutate operation
	mutateOp := libovsdb.Operation{
		Op:        mutateOperation,
		Table:     ovsTableName,
		Mutations: []interface{}{mutation},
		Where:     []interface{}{condition},
	}

	operations := []libovsdb.Operation{deleteOp, mutateOp}
	return client.transact(operations, "delete bridge")
}

func (client *ovsClient) deleteAllPortsOnBridge(brname string) error {
	bridgeExists, err := client.BridgeExists(brname)
	if err != nil {
		return fmt.Errorf("Failed to retrieve the bridge info")
	} else if !bridgeExists {
		return nil
	}

	portList, err := client.findAllPortUUIDsOnBridge(brname)
	if err != nil {
		return err
	}
	if len(portList) != 0 {
		for _, portUUID := range portList {
			err = client.deletePortByUUID(brname, portUUID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// BridgeExists is used to check if a bridge exists or not
func (client *ovsClient) BridgeExists(brname string) (bool, error) {
	// if bridge name is invalid, return false
	if brname == "" {
		return false, fmt.Errorf("The bridge name is invalid")
	}
	condition := libovsdb.NewCondition("name", "==", brname)
	selectOp := libovsdb.Operation{
		Op:    selectOperation,
		Table: bridgeTableName,
		Where: []interface{}{condition},
	}
	operations := []libovsdb.Operation{selectOp}
	reply, _ := client.dbClient.Transact(defaultOvsDB, operations...)

	if len(reply) < len(operations) {
		return false, fmt.Errorf("Number of Replies should be at least equal to number of Operations")
	}
	if reply[0].Error != "" {
		return false, fmt.Errorf("Transaction Failed due to an error: %v", reply[0].Error)
	}
	if len(reply[0].Rows) == 0 {
		return false, nil
	}
	return true, nil
}

func (client *ovsClient) UpdateBridgeController(brname, controller string) error {
	bridgeUpdateLock.Lock()
	defer bridgeUpdateLock.Unlock()
	bridgeExists, err := client.BridgeExists(brname)
	if err != nil {
		return fmt.Errorf("Failed to retrieve the bridge info")
	} else if !bridgeExists {
		return nil
	}

	bridgeUUID, err := client.getBridgeUUIDByName(brname)
	if err != nil {
		return err
	}

	namedControllerUUID := "gocontroller"
	ctrler := make(map[string]interface{})
	ctrler["target"] = controller

	insertControllerOp := libovsdb.Operation{
		Op:       insertOperation,
		Table:    controllerTableName,
		Row:      ctrler,
		UUIDName: namedControllerUUID,
	}
	bridge := make(map[string]interface{})
	bridge["controller"] = libovsdb.UUID{GoUuid: namedControllerUUID}

	updateBrCondition := libovsdb.NewCondition("name", "==", brname)
	updateOp := libovsdb.Operation{
		Op:    updateOperation,
		Table: bridgeTableName,
		Row:   bridge,
		Where: []interface{}{updateBrCondition},
	}

	// Update a Bridge row in Bridge table requires mutating the open_vswitch table
	mutateUUID := []libovsdb.UUID{libovsdb.UUID{GoUuid: namedControllerUUID}}
	mutateSet, _ := libovsdb.NewOvsSet(mutateUUID)
	mutation := libovsdb.NewMutation("controller", insertOperation, mutateSet)
	condition := libovsdb.NewCondition("_uuid", "==", libovsdb.UUID{GoUuid: bridgeUUID})

	// simple mutate operation
	mutateOp := libovsdb.Operation{
		Op:        mutateOperation,
		Table:     bridgeTableName,
		Mutations: []interface{}{mutation},
		Where:     []interface{}{condition},
	}

	operations := []libovsdb.Operation{insertControllerOp, updateOp, mutateOp}
	return client.transact(operations, "update bridge controller")
}

func (client *ovsClient) getBridgeUUIDByName(brname string) (string, error) {
	condition := libovsdb.NewCondition("name", "==", brname)
	selectOp := libovsdb.Operation{
		Op:    selectOperation,
		Table: bridgeTableName,
		Where: []interface{}{condition},
		//Columns: []string{"_uuid"},
	}
	operations := []libovsdb.Operation{selectOp}
	reply, _ := client.dbClient.Transact(defaultOvsDB, operations...)

	if len(reply) < len(operations) {
		return "", fmt.Errorf("get bridge uuid failed due to Number of Replies should be at least equal to number of Operations")
	}
	if reply[0].Error != "" {
		return "", fmt.Errorf("get bridge uuid failed due to Transaction Failed due to an error: %v", reply[0].Error)
	}
	if len(reply[0].Rows) == 0 {
		return "", fmt.Errorf("get bridge uuid failed due to bridge name doesn't exist")
	}
	answer := reply[0].Rows[0]["_uuid"].([]interface{})[1].(string)
	return answer, nil
}
