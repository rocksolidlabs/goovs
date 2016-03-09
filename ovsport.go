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
	intf["options"] = fmt.Sprintf("{peer=%s}", peername)

	return client.createPort(brname, portname, 0, intf)
}

// CreateVethPort ...
func (client *ovsClient) CreateVethPort(brname, portname string) error {
	// intf row to insert
	intf := make(map[string]interface{})
	intf["name"] = portname
	intf["type"] = `system`
	return client.createPort(brname, portname, 0, intf)
}

func (client *ovsClient) createPort(brname, portname string, vlantag int, intf map[string]interface{}) error {
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
	port["interfaces"] = libovsdb.UUID{namedInterfaceUUID}
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
	mutateUUID := []libovsdb.UUID{libovsdb.UUID{namedPortUUID}}
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
	exists, err := client.PortExists(portname)
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
	portDeleteCondition := libovsdb.NewCondition("_uuid", "==", []string{"uuid", portUUID})
	portDeleteOp := libovsdb.Operation{
		Op:    deleteOperation,
		Table: portTableName,
		Where: []interface{}{portDeleteCondition},
	}

	// Inserting a Port row in Port table requires mutating the Bridge table
	mutateUUID := []libovsdb.UUID{libovsdb.UUID{portUUID}}
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

func (client *ovsClient) PortExists(portname string) (bool, error) {
	condition := libovsdb.NewCondition("name", "==", portname)
	return client.selectPortInPortTable(condition)
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
		fmt.Println("The reply is empty, port not found")
		return false, nil
	}
	return true, nil
}

func (client *ovsClient) deleteAllInterfaceOnPort(portname string) error {
	portExists, err := client.PortExists(portname)
	if err != nil {
		return fmt.Errorf("Failed to retrieve the port info")
	} else if !portExists {
		return nil
	}

	interfaceList, err := client.findAllInterfaceUUIDOnPort(portname)
	if err != nil {
		return err
	}
	if len(interfaceList) != 0 {
		for _, interfaceUUID := range interfaceList {
			err = client.RemoveInterfaceFromPort(portname, interfaceUUID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (client *ovsClient) FindAllPortsOnBridge(brname string) ([]string, error) {
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
		return nil, nil
	}
	var portUUIDs []string
	portMap, _ := reply[0].Rows[0]["ports"]

	// Here we pick index 1 since index 0 is string "Set"
	portMapAsSlice := portMap.([]interface{})
	portSet := portMapAsSlice[1]
	v := reflect.ValueOf(portSet)
	if v.Kind() == reflect.String {
		portUUIDs = append(portUUIDs, portSet.(string))
		return portUUIDs, nil
	}

	portSetAsSlice := portSet.([]interface{})
	for _, singleSet := range portSetAsSlice {
		for _, portInfo := range singleSet.([]interface{}) {
			// Portinfo is a string with format: uuidXXXX (where XXX is the UUID of the port)
			portID := portInfo.(string)[4:]
			portUUIDs = append(portUUIDs, portID)
		}
	}
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
	answer := reply[0].Rows[0]["name"].([]interface{})[1].(string)
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
