package main

import (
	"fmt"
	"log"

	"github.com/pan-maruda/gocrafty/ble"
	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

func onStateChanged(d gatt.Device, s gatt.State) {
	fmt.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("scanning...")
		d.Scan([]gatt.UUID{ble.DataServiceUUID, ble.MetaServiceUUID}, false)
		return
	default:
		d.StopScanning()
	}
}

func onPeriphDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	// TODO add selection list or mac input, not connect to the first one
	if a.LocalName == "STORZ&BICKEL" {
		fmt.Printf("\nPeripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
		fmt.Printf("Connecting to %s\n", p.ID())
		p.Device().Connect(p)
	}

}

func onPeriphConnected(p gatt.Peripheral, err error) {
	defer p.Device().CancelConnection(p)

	log.Println("Discovering Crafty services")

	services, err := p.DiscoverServices([]gatt.UUID{ble.DataServiceUUID, ble.MetaServiceUUID})
	if err != nil {
		log.Fatalf("Failed to discover services, err :%s\n", err)
		return
	}

	for _, svc := range services {

		if svc.UUID().Equal(ble.DataServiceUUID) {
			data, err := ble.ReadDataServiceCharacteristics(p, svc)
			if err != nil {
				log.Printf("Failed to read metadata from Crafty: %s\n", err)
				continue
			}
			fmt.Println(data)
		}

		if svc.UUID().Equal(ble.MetaServiceUUID) {
			meta, err := ble.ReadMetadataService(p, svc)
			if err != nil {
				log.Printf("Failed to read metadata from Crafty: %s\n", err)
				continue
			}
			fmt.Println(meta)

		}

	}
}

func main() {
	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	// Register handlers.
	d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered),
		gatt.PeripheralConnected(onPeriphConnected))
	d.Init(onStateChanged)
	select {}
}
