package goovs

import (
    "fmt"
    "log"
    
    "github.com/kopwei/libovsdb"
)

// CreateBridge is used to create a ovs bridge
func (client *ovsClient)CreateBridge(brname string) error {
    namedBridgeUUID := "gobridge"
    namedPortUUID := "goport"
    namedInterfaceUUID := "gointerface"
    
    // intf row to insert
	intf := make(map[string]interface{})
	intf["name"] = brname
	intf["type"] = `internal`
    
    insertInterfaceOp := libovsdb.Operation{
        Op: "insert",
        Table: "Interface",
        Row: intf,
        UUIDName: namedInterfaceUUID,
    }
    
    // port row to insert
    port := make(map[string]interface{})
    port["name"] = brname
    port["interfaces"] = libovsdb.UUID{namedInterfaceUUID}
    
    insertPortOp := libovsdb.Operation {
        Op: insertOperation,
        Table: portTableName,
        Row: port,
        UUIDName: namedPortUUID,
    }
    
    // bridge row to insert
    bridge := make(map[string]interface{})
    bridge["name"] = brname
    bridge["stp_enable"] = true
    bridge["ports"] = libovsdb.UUID{namedPortUUID}
    
    // simple insert operation
    insertBridgeOp := libovsdb.Operation{
        Op: insertOperation,
        Table: bridgeTableName,
        Row: bridge,
        UUIDName: namedBridgeUUID,
    }
    
    // Inserting a Bridge row in Bridge table requires mutating the open_vswitch table
    mutateUUID := []libovsdb.UUID{libovsdb.UUID{namedBridgeUUID}}
    mutateSet, _ := libovsdb.NewOvsSet(mutateUUID)
    mutation := libovsdb.NewMutation("bridges", insertOperation, mutateSet)
    condition := libovsdb.NewCondition("_uuid", "==", libovsdb.UUID{getRootUUID()})
    
    // simple mutate operation
    mutateOp := libovsdb.Operation{
        Op: mutateOperation,
        Table: ovsTableName,
        Mutations: [] interface{}{mutation},
        Where: []interface{}{condition},
    }
    
    operations := []libovsdb.Operation{insertInterfaceOp, insertPortOp, insertBridgeOp, mutateOp}
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
        log.Println("Bridge Addition Successful : ", reply[0].UUID.GoUuid)
    }
    
    return nil
}

// DeleteBridge is used to delete a ovs bridge
func (client *ovsClient)DeleteBridge(brname string) error {
    namedBridgeUUID := "gobridge"
    
    delBridgeCondition := libovsdb.NewCondition("name", "==", brname)
    deleteOp := libovsdb.Operation{
        Op: deleteOperation,
        Table: bridgeTableName,
        Where: []interface{}{delBridgeCondition},
    }
    
    // Deleting a Bridge row in Bridge table requires mutating the open_vswitch table
    mutateUUID := []libovsdb.UUID{libovsdb.UUID{namedBridgeUUID}}
    mutateSet, _ := libovsdb.NewOvsSet(mutateUUID)
    mutation := libovsdb.NewMutation("bridges", deleteOperation, mutateSet)
    condition := libovsdb.NewCondition("_uuid", "==", libovsdb.UUID{getRootUUID()})
    
    // simple mutate operation
    mutateOp := libovsdb.Operation {
        Op: mutateOperation,
        Table: ovsTableName,
        Mutations: []interface{}{mutation},
        Where: []interface{}{condition},
    }
    
    operations := []libovsdb.Operation{deleteOp, mutateOp}
    reply, _ := client.dbClient.Transact(defaultOvsDB, operations...)
    
    if len(reply) < len(operations) {
        return fmt.Errorf("Number of Replies should be atleast equal to number of Operations")
    }
    ok := true
    for i, o := range reply {
        if o.Error != "" {
            ok = false
            if i < len(operations) {
                return fmt.Errorf("Transaction Failed due to an error : %s details: %s in %+v", o.Error, o.Details, operations[i])
            }
            return fmt.Errorf("Transaction Failed due to an error :%s details: %s", o.Error, o.Details)
        } 
    }
    if ok {
        log.Println("Bridge Deletion Successful : ", reply[0].UUID.GoUuid)
    }
    
    return nil
}