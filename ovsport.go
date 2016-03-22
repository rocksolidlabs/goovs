package goovs

import (
	"fmt"
	"reflect"
	//"strings"

	"github.com/kopwei/libovsdb"
)

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
	//intf["type"] = `system`
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
	port["interfaces"] = libovsdb.UUID{GoUuid: namedInterfaceUUID}
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
	mutateUUID := []libovsdb.UUID{libovsdb.UUID{GoUuid: namedPortUUID}}
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
	mutateUUID := []libovsdb.UUID{libovsdb.UUID{GoUuid: portUUID}}
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
	condition := libovsdb.NewCondition("_uuid", "==", []string{"uuid", portUUID})
	return client.selectPortInPortTable(condition)
}

func (client *ovsClient) selectPortInPortTable(selectCondition []interface{}) (bool, error) {
	selectOp := libovsdb.Operation{
		Op:    selectOperation,
		Table: portTableName,
		Where: []interface{}{selectCondition},
	}
	operations := []libovsdb.Operation{selectOp}
	reply, _ := client.dbClient.Transact(defaultOvsDB, operations...)

	if len(reply) < len(operations) {
		return false, fmt.Errorf("find port in table failed due to Number of Replies should be at least equal to number of Operations")
	}
	if reply[0].Error != "" {
		return false, fmt.Errorf("find port in table failed due to Transaction Failed due to an error: %v", reply[0].Error)
	}
	if len(reply[0].Rows) == 0 {
		// fmt.Println("The reply is empty, port not found")
		return false, nil
	}
	return true, nil
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
	condition := libovsdb.NewCondition("name", "==", brname)
	selectOp := libovsdb.Operation{
		Op:      selectOperation,
		Table:   bridgeTableName,
		Where:   []interface{}{condition},
		Columns: []string{"ports"},
	}
	operations := []libovsdb.Operation{selectOp}
	reply, _ := client.dbClient.Transact(defaultOvsDB, operations...)
	if len(reply) < len(operations) {
		return nil, fmt.Errorf("Number of Replies should be at least equal to number of Operations")
	}
	if reply[0].Error != "" {
		return nil, fmt.Errorf("Transaction Failed due to an error: %v", reply[0].Error)
	}
	if len(reply[0].Rows) == 0 {
		//fmt.Printf("No ports found on bridge %s\n", brname)
		return nil, nil
	}
	var portUUIDs []string
	portMap, _ := reply[0].Rows[0]["ports"]

	// Here we pick index 1 since index 0 is string "Set"
	portMapAsSlice := portMap.([]interface{})
	portSet := portMapAsSlice[1]
	v := reflect.ValueOf(portSet)
	if v.Kind() == reflect.String {
		//fmt.Printf("One ports %s found on bridge %s\n", portSet.(string), brname)
		portUUIDs = append(portUUIDs, portSet.(string))
		return portUUIDs, nil
	}

	portSetAsSlice := portSet.([]interface{})
	for _, singleSet := range portSetAsSlice {
		for _, portInfo := range singleSet.([]interface{}) {
			if len(portInfo.(string)) > 4 {
				portID := portInfo.(string)
				portUUIDs = append(portUUIDs, portID)
			}
		}
	}
	//fmt.Printf("There are %d ports found on bridge %s and they are %+v\n", len(portUUIDs), brname, portUUIDs)
	return portUUIDs, nil
}

func (client *ovsClient) getPortNameByUUID(portUUID string) (string, error) {
	condition := libovsdb.NewCondition("_uuid", "==", []string{"uuid", portUUID})
	selectOp := libovsdb.Operation{
		Op:      selectOperation,
		Table:   portTableName,
		Where:   []interface{}{condition},
		Columns: []string{"name"},
	}
	operations := []libovsdb.Operation{selectOp}
	reply, _ := client.dbClient.Transact(defaultOvsDB, operations...)
	if len(reply) < len(operations) {
		return "", fmt.Errorf("Find port uuid by name failed due to Number of Replies should be at least equal to number of Operations")
	}
	if reply[0].Error != "" {
		return "", fmt.Errorf("Find port uuid by name transaction Failed due to an error: %v", reply[0].Error)
	}
	if len(reply[0].Rows) == 0 {
		return "", fmt.Errorf("Unable to find the port with uuid %s", portUUID)
	}
	answer := reply[0].Rows[0]["name"].(string)
	//fmt.Printf("Port name %s is found for port with uuid %s\n", answer, portUUID)
	return answer, nil
}

func (client *ovsClient) getPortUUIDByName(portname string) (string, error) {
	condition := libovsdb.NewCondition("name", "==", portname)
	selectOp := libovsdb.Operation{
		Op:    selectOperation,
		Table: portTableName,
		Where: []interface{}{condition},
	}
	operations := []libovsdb.Operation{selectOp}
	reply, _ := client.dbClient.Transact(defaultOvsDB, operations...)
	if len(reply) < len(operations) {
		return "", fmt.Errorf("Find port name by uuid failed due to Number of Replies should be at least equal to number of Operations")
	}
	if reply[0].Error != "" {
		return "", fmt.Errorf("Find port name by uuid transaction Failed due to an error: %v", reply[0].Error)
	}
	if len(reply[0].Rows) == 0 {
		return "", fmt.Errorf("Unable to find the port with name %s", portname)
	}
	answer := reply[0].Rows[0]["_uuid"].([]interface{})[1].(string)
	return answer, nil
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
