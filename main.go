package main

import (
        "fmt"
        "log"
        "tinygo.org/x/bluetooth"
)

func main() {
        adapter := bluetooth.DefaultAdapter
        if err := adapter.Enable(); err != nil {
                log.Fatal("enable bluetooth: ", err)
        }
        fmt.Println("scanning... (Ctrl+C to stop)")

        err := adapter.Scan(func(a *bluetooth.Adapter, d bluetooth.ScanResult) {
                fmt.Printf("%-30s %s %4d dBm\n", d.LocalName(), d.Address, d.RSSI)
        })
        if err != nil {
                log.Fatal("scan: ", err)
        }
}