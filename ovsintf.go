package goovs

import (
	"fmt"
	"reflect"

	"github.com/kopwei/libovsdb"
)

// OvsInterface is the structure represents an interface row
type OvsInterface struct {
	UUID    string            `json:"_uuid"`
	Name    string            `json:"name"`
	Options map[string]string `json:"options"`
	Type    string            `json:"type"`
}

// ReadFromDBRow is used to initialize the object from a row
func (intf *OvsInterface) ReadFromDBRow(row *libovsdb.Row) error {
	if intf.Options == nil {
		intf.Options = make(map[string]string)
	}
	for field, value := range row.Fields {
		switch field {
		case "name":
			intf.Name = value.(string)
		case "type":
			intf.Type = value.(string)
		case "options":
			for key, opt := range value.(libovsdb.OvsMap).GoMap {
				intf.Options[key.(string)] = opt.(string)
			}
		}
	}
	return nil
}

// AddInternalInterfaceOnPort ...
func (client *ovsClient) AddInternalInterfaceOnPort(portname string) error {
	// intf row to insert
	intf := make(map[string]interface{})
	intf["name"] = portname
	intf["type"] = `internal`

	return client.addInterfaceOnPort(portname, intf)
}

// AddVethInterfaceOnPort ...
func (client *ovsClient) AddVethInterfaceOnPort(portname string) error {
	// intf row to insert
	intf := make(map[string]interface{})
	intf["name"] = portname
	intf["type"] = `system`

	return client.addInterfaceOnPort(portname, intf)
}

// AddPeerInterfaceOnPort ...
func (client *ovsClient) AddPeerInterfaceOnPort(portname, peername string) error {
	// intf row to insert
	intf := make(map[string]interface{})
	intf["name"] = portname
	intf["type"] = `peer`
	intf["options"] = fmt.Sprintf("{peer=%s}", peername)

	return client.addInterfaceOnPort(portname, intf)
}

func (client *ovsClient) addInterfaceOnPort(portName string, intf map[string]interface{}) error {
	intfUpdateLock.Lock()
	defer intfUpdateLock.Unlock()
	namedInterfaceUUID := "gointerface"
	insertInterfaceOp := libovsdb.Operation{
		Op:       insertOperation,
		Table:    interfaceTableName,
		Row:      intf,
		UUIDName: namedInterfaceUUID,
	}
	// Inserting a Port row in Port table requires mutating the Bridge table
	mutateUUID := []libovsdb.UUID{libovsdb.UUID{GoUuid: namedInterfaceUUID}}
	mutateSet, _ := libovsdb.NewOvsSet(mutateUUID)
	mutation := libovsdb.NewMutation("interfaces", insertOperation, mutateSet)
	condition := libovsdb.NewCondition("name", "==", portName)

	// simple mutate operation
	mutateOp := libovsdb.Operation{
		Op:        mutateOperation,
		Table:     portTableName,
		Mutations: []interface{}{mutation},
		Where:     []interface{}{condition},
	}

	operations := []libovsdb.Operation{insertInterfaceOp, mutateOp}
	return client.transact(operations, "add interface")
}

func (client *ovsClient) RemoveInterfaceFromPort(portname, interfaceUUID string) error {
	intfUpdateLock.Lock()
	defer intfUpdateLock.Unlock()
	namedInterfaceUUID := "gointerface"
	interfaceDeleteCondition := libovsdb.NewCondition("_uuid", "==", []string{"uuid", interfaceUUID})
	interfaceDeleteOp := libovsdb.Operation{
		Op:    deleteOperation,
		Table: interfaceTableName,
		Where: []interface{}{interfaceDeleteCondition},
	}

	// Inserting a Port row in Port table requires mutating the Bridge table
	mutateUUID := []libovsdb.UUID{libovsdb.UUID{GoUuid: namedInterfaceUUID}}
	mutateSet, _ := libovsdb.NewOvsSet(mutateUUID)
	mutation := libovsdb.NewMutation("interfaces", deleteOperation, mutateSet)
	condition := libovsdb.NewCondition("name", "==", portname)

	// simple mutate operation
	mutateOp := libovsdb.Operation{
		Op:        mutateOperation,
		Table:     portTableName,
		Mutations: []interface{}{mutation},
		Where:     []interface{}{condition},
	}

	operations := []libovsdb.Operation{interfaceDeleteOp, mutateOp}
	return client.transact(operations, "remove interface")
}

func (client *ovsClient) interfaceUUIDExists(interfaceUUID string) (bool, error) {
	condition := libovsdb.NewCondition("_uuid", "==", []string{"uuid", interfaceUUID})
	selectOp := libovsdb.Operation{
		Op:    selectOperation,
		Table: interfaceTableName,
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
		//fmt.Println("The reply is empty, interface not found")
		return false, nil
	}
	return true, nil
}

func (client *ovsClient) findAllInterfaceUUIDOnPort(portname string) ([]string, error) {
	condition := libovsdb.NewCondition("name", "==", portname)
	selectOp := libovsdb.Operation{
		Op:      selectOperation,
		Table:   portTableName,
		Where:   []interface{}{condition},
		Columns: []string{"interfaces"},
	}
	operations := []libovsdb.Operation{selectOp}
	reply, _ := client.dbClient.Transact(defaultOvsDB, operations...)
	if len(reply) < len(operations) {
		return nil, fmt.Errorf("Get interface from port failed due to Number of Replies should be at least equal to number of Operations")
	}
	if reply[0].Error != "" {
		return nil, fmt.Errorf("Get interface from port failed due to Transaction Failed due to an error: %v", reply[0].Error)
	}
	if len(reply[0].Rows) == 0 {
		return nil, nil
	}
	var interfaceUUIDs []string
	interfaceMap, _ := reply[0].Rows[0]["interfaces"]
	interfaceMapAsSlice := interfaceMap.([]interface{})
	// Here we pick index 1 since index 0 is string "Set"
	interfaceSet := interfaceMapAsSlice[1]
	v := reflect.ValueOf(interfaceSet)
	if v.Kind() == reflect.String {
		interfaceUUIDs = append(interfaceUUIDs, interfaceSet.(string))
		return interfaceUUIDs, nil
	}

	interfaceSetAsSlice := interfaceSet.([]interface{})
	for _, singleSet := range interfaceSetAsSlice {
		for _, interfaceInfo := range singleSet.([]interface{}) {
			// Portinfo is a string with format: uuidXXXX (where XXX is the UUID of the port)
			interfaceID := interfaceInfo.(string)[4:]
			interfaceUUIDs = append(interfaceUUIDs, interfaceID)
		}
	}
	return interfaceUUIDs, nil
}
