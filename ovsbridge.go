package goovs

import (
	"fmt"

	"github.com/kopwei/libovsdb"
)

// OvsBridge is the structure represents the ovs bridge
type OvsBridge struct {
	UUID       string   `json:"_uuid"`
	Controller string   `json:"controller"`
	Name       string   `json:"name"`
	PortUUIDs  []string `json:"ports"`
	DatapathID string   `json:"datapath_id"`
}

// ReadFromDBRow is used to initialize the object from a row
func (bridge *OvsBridge) ReadFromDBRow(row *libovsdb.Row) error {
	if bridge.PortUUIDs == nil {
		bridge.PortUUIDs = make([]string, 0)
	}
	for field, value := range row.Fields {
		switch field {
		case "name":
			bridge.Name = value.(string)
		case "datapath_id":
			switch value.(type) {
			case string:
				bridge.DatapathID = value.(string)
			}
		case "ports":
			switch value.(type) {
			case libovsdb.UUID:
				bridge.PortUUIDs = append(bridge.PortUUIDs, value.(libovsdb.UUID).GoUuid)
			case libovsdb.OvsSet:
				for _, uuids := range value.(libovsdb.OvsSet).GoSet {
					bridge.PortUUIDs = append(bridge.PortUUIDs, uuids.(libovsdb.UUID).GoUuid)
				}
			}
		}
	}
	return nil
}

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
	bridgeCacheUpdateLock.RLock()
	defer bridgeCacheUpdateLock.RUnlock()
	for _, br := range client.bridgeCache {
		if br.Name == brname {
			return true, nil
		}
	}
	return false, nil
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
	if brname == "" {
		return "", fmt.Errorf("The bridge name is invalid")
	}
	for uuid, bridge := range client.bridgeCache {
		if bridge.Name == brname {
			return uuid, nil
		}
	}
	return "", fmt.Errorf("The brdige name %s doesn't exist", brname)
}
