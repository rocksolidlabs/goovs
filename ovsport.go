package goovs

import (
	"fmt"

	"github.com/rocksolidlabs/libovsdb"
)

// OvsPort represents a ovs port structure
type OvsPort struct {
	UUID      string   `json:"_uuid"`
	Name      string   `json:"name"`
	IntfUUIDs []string `json:"interfaces"`
	Tag       float64  `json:"tag"`
}

// ReadFromDBRow is used to initialize the object from a row
func (port *OvsPort) ReadFromDBRow(row *libovsdb.Row) error {
	if port.IntfUUIDs == nil {
		port.IntfUUIDs = make([]string, 0)
	}
	for field, value := range row.Fields {
		switch field {
		case "name":
			port.Name = value.(string)
		case "tag":
			switch value.(type) {
			case float64:
				port.Tag = value.(float64)
				/*
					case libovsdb.OvsSet:
					for _, tags := value.(libovsdb.OvsSet).GoSet {

					}
				*/
			}

		case "interfaces":
			switch value.(type) {
			case libovsdb.UUID:
				port.IntfUUIDs = append(port.IntfUUIDs, value.(libovsdb.UUID).GoUUID)
			case libovsdb.OvsSet:
				for _, uuids := range value.(libovsdb.OvsSet).GoSet {
					port.IntfUUIDs = append(port.IntfUUIDs, uuids.(libovsdb.UUID).GoUUID)
				}
			}
		}
	}
	return nil
}

// CreateInternalPort ...
func (client *ovsClient) CreateInternalPort(brname, portname string, vlantag int) error {
	// intf row to insert
	intf := make(map[string]interface{})
	intf["name"] = portname
	intf["type"] = `internal`

	return client.createPort(brname, portname, vlantag, intf)
}

// CreateInternalPort ...
func (client *ovsClient) CreatePatchPort(brname, portname, peername string) error {
	// intf row to insert
	intf := make(map[string]interface{})
	intf["name"] = portname
	intf["type"] = `patch`
	options := make(map[string]interface{})
	options["peer"] = peername
	intf["options"], _ = libovsdb.NewOvsMap(options)

	return client.createPort(brname, portname, 0, intf)
}

// CreateVethPort ...
func (client *ovsClient) CreateVethPort(brname, portname string, vlantag int) error {
	// intf row to insert
	intf := make(map[string]interface{})
	intf["name"] = portname
	intf["type"] = "system"
	return client.createPort(brname, portname, vlantag, intf)
}

func (client *ovsClient) createPort(brname, portname string, vlantag int, intf map[string]interface{}) error {
	portUpdateLock.Lock()
	defer portUpdateLock.Unlock()
	portExists, err := client.PortExistsOnBridge(portname, brname)
	if err != nil {
		return fmt.Errorf("Failed to retrieve the port info due to %s", err.Error())
	} else if portExists {
		return nil
	}
	namedPortUUID := "goport"
	namedInterfaceUUID := "gointerface"

	insertInterfaceOp := libovsdb.Operation{
		Op:       "insert",
		Table:    "Interface",
		Row:      intf,
		UUIDName: namedInterfaceUUID,
	}

	// port row to insert
	port := make(map[string]interface{})
	port["name"] = portname
	port["interfaces"] = libovsdb.UUID{GoUUID: namedInterfaceUUID}
	if vlantag > 0 && vlantag <= 4095 {
		port["tag"] = vlantag
	}

	insertPortOp := libovsdb.Operation{
		Op:       insertOperation,
		Table:    portTableName,
		Row:      port,
		UUIDName: namedPortUUID,
	}
	// Inserting a Port row in Port table requires mutating the Bridge table
	mutateUUID := []libovsdb.UUID{libovsdb.UUID{GoUUID: namedPortUUID}}
	mutateSet, _ := libovsdb.NewOvsSet(mutateUUID)
	mutation := libovsdb.NewMutation("ports", insertOperation, mutateSet)
	condition := libovsdb.NewCondition("name", "==", brname)

	// simple mutate operation
	mutateOp := libovsdb.Operation{
		Op:        mutateOperation,
		Table:     bridgeTableName,
		Mutations: []interface{}{mutation},
		Where:     []interface{}{condition},
	}

	operations := []libovsdb.Operation{insertInterfaceOp, insertPortOp, mutateOp}
	return client.transact(operations, "create port")
}

func (client *ovsClient) DeletePort(brname, portname string) error {
	exists, err := client.PortExistsOnBridge(portname, brname)
	if err != nil {
		return err
	} else if !exists {
		return nil
	}

	portUUID, err := client.getPortUUIDByName(portname)
	if err != nil {
		return err
	}
	return client.deletePortByUUID(brname, portUUID)
}

func (client *ovsClient) deletePortByUUID(brname, portUUID string) error {
	portUpdateLock.Lock()
	defer portUpdateLock.Unlock()
	portDeleteCondition := libovsdb.NewCondition("_uuid", "==", []string{"uuid", portUUID})
	portDeleteOp := libovsdb.Operation{
		Op:    deleteOperation,
		Table: portTableName,
		Where: []interface{}{portDeleteCondition},
	}

	// Inserting a Port row in Port table requires mutating the Bridge table
	mutateUUID := []libovsdb.UUID{libovsdb.UUID{GoUUID: portUUID}}
	mutateSet, _ := libovsdb.NewOvsSet(mutateUUID)
	mutation := libovsdb.NewMutation("ports", deleteOperation, mutateSet)
	condition := libovsdb.NewCondition("name", "==", brname)

	// simple mutate operation
	mutateOp := libovsdb.Operation{
		Op:        mutateOperation,
		Table:     bridgeTableName,
		Mutations: []interface{}{mutation},
		Where:     []interface{}{condition},
	}

	operations := []libovsdb.Operation{portDeleteOp, mutateOp}
	return client.transact(operations, "delete port")
}

func (client *ovsClient) PortExistsOnBridge(portname, brname string) (bool, error) {
	portUUIDs, err := client.findAllPortUUIDsOnBridge(brname)
	if err != nil {
		return false, nil
	} else if len(portUUIDs) == 0 {
		return false, nil
	}
	for _, portUUID := range portUUIDs {
		portName, err := client.getPortNameByUUID(portUUID)
		if err != nil {
			//fmt.Printf("Failed to get port name by uuid %s\n", portUUID)
			return false, err
		}
		if portName == portname {
			return true, nil
		}
	}
	return false, nil
}

func (client *ovsClient) portExistsByUUID(portUUID string) (bool, error) {
	_, ok := client.portCache[portUUID]
	return ok, nil
}

func (client *ovsClient) FindAllPortsOnBridge(brname string) ([]string, error) {
	portUUIDs, err := client.findAllPortUUIDsOnBridge(brname)
	if err != nil {
		return nil, err
	}
	if len(portUUIDs) == 0 {
		return nil, nil
	}
	portNames := make([]string, len(portUUIDs), len(portUUIDs))
	for index, portUUID := range portUUIDs {
		name, err := client.getPortNameByUUID(portUUID)
		if err != nil {
			return nil, err
		}
		portNames[index] = name
	}
	return portNames, nil
}

func (client *ovsClient) findAllPortUUIDsOnBridge(brname string) ([]string, error) {
	bridgeCacheUpdateLock.RLock()
	defer bridgeCacheUpdateLock.RUnlock()
	for _, bridge := range client.bridgeCache {
		if bridge.Name == brname {
			return bridge.PortUUIDs, nil
		}
	}
	//fmt.Printf("There are %d ports found on bridge %s and they are %+v\n", len(portUUIDs), brname, portUUIDs)
	return nil, fmt.Errorf("Bridge with name %s doesn't exist", brname)
}

func (client *ovsClient) getPortNameByUUID(portUUID string) (string, error) {
	portCacheUpdateLock.RLock()
	defer portCacheUpdateLock.RUnlock()
	port, ok := client.portCache[portUUID]
	if !ok {
		return "", fmt.Errorf("port with uuid %s doesn't exist", portUUID)
	}
	return port.Name, nil
}

func (client *ovsClient) getPortUUIDByName(portname string) (string, error) {
	portCacheUpdateLock.RLock()
	defer portCacheUpdateLock.RUnlock()
	for uuid, ovsport := range client.portCache {
		if ovsport.Name == portname {
			return uuid, nil
		}
	}
	return "", fmt.Errorf("Unable to find the port uuid with name %s", portname)
}

func (client *ovsClient) UpdatePortTagByName(brname, portname string, vlantag int) error {
	if vlantag < 0 || vlantag > 4095 {
		return fmt.Errorf("The vlan tag value is not in valid range")
	}
	portExist, err := client.PortExistsOnBridge(portname, brname)
	if err != nil {
		return err
	}
	if !portExist {
		return fmt.Errorf("The port %s doesn't exists on bridge %s", portname, brname)
	}
	portUUID, err := client.getPortUUIDByName(portname)
	if err != nil {
		return err
	}
	return client.updatePortTagByUUID(portUUID, vlantag)
}

func (client *ovsClient) updatePortTagByUUID(portUUID string, vlantag int) error {
	if vlantag < 0 || vlantag > 4095 {
		return fmt.Errorf("The vlan tag value is not in valid range")
	}
	portUpdateLock.Lock()
	defer portUpdateLock.Unlock()
	portExist, err := client.portExistsByUUID(portUUID)
	if err != nil {
		return err
	}
	if !portExist {
		return fmt.Errorf("The port with UUID %s doesn't exists", portUUID)
	}
	updateCondition := libovsdb.NewCondition("_uuid", "==", []string{"uuid", portUUID})
	// port row to update
	port := make(map[string]interface{})
	port["tag"] = vlantag

	updateOp := libovsdb.Operation{
		Op:    updateOperation,
		Table: portTableName,
		Row:   port,
		Where: []interface{}{updateCondition},
	}
	operations := []libovsdb.Operation{updateOp}
	return client.transact(operations, "update port")
}
