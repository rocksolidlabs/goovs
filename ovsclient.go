package goovs

import (
    "net"
    "reflect"
    "strconv"
    
    "github.com/kopwei/libovsdb"
)

const (
    defaultTCPHost = "127.0.0.1"
    defaultTCPPort = 6640
    defaultUnixEndpoint = "/var/run/openvswitch/db.sock"
)

const (
    defaultOvsDB = "Open_vSwitch"
)

const (
    ovsTableName = "Open_vSwitch"
    bridgeTableName = "Bridge"
    portTableName = "Port"
    interfaceTableName = "Interface"
    //flowTableName = ""
)

const (
    deleteOperation = "delete"
    insertOperation = "insert"
    mutateOperation = "mutate"
)

// OvsClient is the interface towards outside user
type OvsClient interface {
    CreateBridge(brname string) error
    DeleteBridge(brname string) error
    CreateInternalPort(brname, portname string) error
    CreateVethPort(brname, portname string) error
}

type ovsClient struct {
    dbClient *libovsdb.OvsdbClient
}

var client *ovsClient
var update chan *libovsdb.TableUpdates
var cache map[string]map[string]libovsdb.Row

// GetOVSClient is used for
func GetOVSClient(contype, endpoint string) (OvsClient, error) {
    if client != nil {
        return client, nil
    }
    var dbclient *libovsdb.OvsdbClient
    var err error
    if contype == "tcp" {
        if endpoint == "" {
            dbclient, err = libovsdb.Connect(defaultTCPHost, defaultTCPPort)
        } else {
            host, port, err := net.SplitHostPort(endpoint)
            if err != nil {
                return nil, err
            }
            portInt, _ := strconv.Atoi(port)
            dbclient, err = libovsdb.Connect(host, portInt)
        }
    } else if contype == "unix" {
        if endpoint == "" {
            endpoint = defaultUnixEndpoint
        }
        dbclient, err = libovsdb.ConnectUnix(endpoint)
    }
    if err != nil {
        return nil, err
    }
    //var notifier Notifier
	//dbclient.Register(notifier)
    
    //update = make(chan *libovsdb.TableUpdates)
	cache = make(map[string]map[string]libovsdb.Row)

	initial, _ := dbclient.MonitorAll(defaultOvsDB, "")
	populateCache(*initial)

    client = &ovsClient{dbClient: dbclient}
    return client, nil
}
/*
type Notifier struct {
}

func (n Notifier) Update(context interface{}, tableUpdates libovsdb.TableUpdates) {
	populateCache(tableUpdates)
	update <- &tableUpdates
}
func (n Notifier) Locked([]interface{}) {
}
func (n Notifier) Stolen([]interface{}) {
}
func (n Notifier) Echo([]interface{}) {
}
*/

func populateCache(updates libovsdb.TableUpdates) {
	for table, tableUpdate := range updates.Updates {
		if _, ok := cache[table]; !ok {
			cache[table] = make(map[string]libovsdb.Row)

		}
		for uuid, row := range tableUpdate.Rows {
			empty := libovsdb.Row{}
			if !reflect.DeepEqual(row.New, empty) {
				cache[table][uuid] = row.New
			} else {
				delete(cache[table], uuid)
			}
		}
	}
}


func getRootUUID() string {
    for uuid := range cache[defaultOvsDB] {
        return uuid
    }
    return ""
}