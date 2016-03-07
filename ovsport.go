package goovs

import (
	"fmt"
	"log"
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
func (client *ovsClient) CreateVethPort(brname, portname string, vlantag int) error {
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
		Table:     ovsTableName,
		Mutations: []interface{}{mutation},
		Where:     []interface{}{condition},
	}

	operations := []libovsdb.Operation{insertInterfaceOp, insertPortOp, mutateOp}
	reply, _ := client.dbClient.Transact(defaultOvsDB, operations...)

	if len(reply) < len(operations) {
		return fmt.Errorf("Number of Replies should be at least equal to number of Operations")
	}
	ok := true
	for i, o := range reply {
		if o.Error != "" {
			ok = false
			if i < len(operations) {
				return fmt.Errorf("Transaction Failed due to an error : %s details: %s in %+v", o.Error, o.Details, operations[i])
			}
			return fmt.Errorf("Transaction Failed due to an error :%s", o.Error)
		}
	}
	if ok {
		log.Println("Veth Port Addition Successful : ", reply[0].UUID.GoUuid)
	}

	return nil
}

func (client *ovsClient) DeletePort(brname, portname string) error {
	exists, err := client.PortExists(portname)
	if !exists {
		return nil
	} else if err != nil {
		return fmt.Errorf("Unable to retrieve the port info")
	}

	namedPortUUID := "goport"
	portDeleteCondition := libovsdb.NewCondition("name", "==", portname)

	portDeleteOp := libovsdb.Operation{
		Op:    deleteOperation,
		Table: portTableName,
		Where: []interface{}{portDeleteCondition},
	}

	// Inserting a Port row in Port table requires mutating the Bridge table
	mutateUUID := []libovsdb.UUID{libovsdb.UUID{namedPortUUID}}
	mutateSet, _ := libovsdb.NewOvsSet(mutateUUID)
	mutation := libovsdb.NewMutation("ports", deleteOperation, mutateSet)
	condition := libovsdb.NewCondition("name", "==", brname)

	// simple mutate operation
	mutateOp := libovsdb.Operation{
		Op:        mutateOperation,
		Table:     ovsTableName,
		Mutations: []interface{}{mutation},
		Where:     []interface{}{condition},
	}

	operations := []libovsdb.Operation{portDeleteOp, mutateOp}
	reply, _ := client.dbClient.Transact(defaultOvsDB, operations...)

	if len(reply) < len(operations) {
		return fmt.Errorf("Number of Replies should be at least equal to number of Operations")
	}
	ok := true
	for i, o := range reply {
		if o.Error != "" {
			ok = false
			if i < len(operations) {
				return fmt.Errorf("Transaction Failed due to an error : %s details: %s in %+v", o.Error, o.Details, operations[i])
			}
			return fmt.Errorf("Transaction Failed due to an error :%s", o.Error)
		}
	}
	if ok {
		log.Println("Port Deleting Successful : ", reply[0].UUID.GoUuid)
	}

	return nil
}

func (client *ovsClient) PortExists(portname string) (bool, error) {
	condition := libovsdb.NewCondition("name", "==", portname)
	selectOp := libovsdb.Operation{
		Op:    selectOperation,
		Table: portTableName,
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
	portMap, _ := libovsdb.NewOvsSet(reply[0].Rows[0]["ports"])
	// Here we pick index 1 since index 0 is string "Set"
	portSet := portMap.GoSet[1].([]interface{})

	for _, singleSet := range portSet {
		for _, portInfo := range singleSet.([]interface{}) {
			// Portinfo is a string with format: uuidXXXX (where XXX is the UUID of the port)
			portID := portInfo.(string)[4:]
			portUUIDs = append(portUUIDs, portID)
		}
	}
	return portUUIDs, nil
}
