package main

import (
    "fmt"
    
    "github.com/kopwei/goovs"
)


func main() {
    client, err := goovs.GetOVSClient("unix", "")
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    err = client.CreateBridge("helloworld")
    if err != nil {
        fmt.Println(err.Error())
        return
    }
}