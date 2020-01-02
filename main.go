package main

import (
	"fmt"
	"log"
	"os"

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

func onPeriphDiscovered(craftyID string, done <-chan bool) func(gatt.Peripheral, *gatt.Advertisement, int) {

	return func(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
		// TODO add selection list or mac input, not connect to the first one
		if p.ID() == craftyID {
			fmt.Printf("Connecting to %s [%s]\n", p.ID(), p.Name())
			p.Device().Connect(p)
		}
	}

}

func onPeriphConnected(craftyID string, done chan bool, commands []string) func(gatt.Peripheral, error) {

	return func(p gatt.Peripheral, err error) {

		if p.ID() != craftyID {
			fmt.Printf("Unexpected device ID [%s] connected instead of [%s]. WTF?", p.ID(), craftyID)
			return
		}

		p.Device().StopScanning()

		services, err := p.DiscoverServices([]gatt.UUID{ble.DataServiceUUID, ble.MetaServiceUUID})
		if err != nil {
			log.Fatalf("Failed to discover services, err :%s\n", err)
			return
		}

		var dataSvc *ble.DataService
		// var metaSvc *gatt.Service

		for _, svc := range services {

			if svc.UUID().Equal(ble.DataServiceUUID) {
				// dataSvc = svc

				if disc, err := ble.DiscoverDataService(p, svc); err == nil {
					data, err := ble.ReadDataServiceCharacteristics(p, disc)
					if err != nil {
						log.Printf("Failed to read metadata from Crafty: %s\n", err)
						continue
					}
					fmt.Println(data)
					dataSvc = disc
				} else {
					log.Printf("Failed to discover Crafty data service: %s\n", err)
					continue
				}

			}

			if svc.UUID().Equal(ble.MetaServiceUUID) {
				// metaSvc = svc
				meta, err := ble.ReadMetadataService(p, svc)
				if err != nil {
					log.Printf("Failed to read metadata from Crafty: %s\n", err)
					continue
				}
				fmt.Println(meta)
			}
		}

		// todo proper cli parsing, not this duct tape mess
		switch commands[0] {
		case "monitor":
			if dataSvc == nil {
				fmt.Printf("Data service is nil - not discovered? wtf?")
				break
			}
			dataSvc.SubscribeBattery(p, func(batteryLevel uint16, err error) {
				fmt.Printf("Battery level: %d%%\n", batteryLevel)
			})
			dataSvc.SubscribeTemp(p, func(currentTemp uint16, err error) {
				fmt.Printf("Current Temp: %d.%d C\n", currentTemp/10, currentTemp%10)
			})
			select {}
		default:
			done <- true
		}

		// defer p.Device().CancelConnection(p)
		// defer fmt.Printf("Disconnected from %s", p.ID())
		// defer p.Device().Scan([]gatt.UUID{ble.DataServiceUUID, ble.MetaServiceUUID}, false)
		// log.Println("Discovering Crafty services")

	}
}

func main() {
	done := make(chan bool, 1)
	craftyID, found := os.LookupEnv("CRAFTY_ID")
	if !found {
		fmt.Println("CRAFTY_ID not set, use ./scanner to find your Crafty.")
		os.Exit(1)
	}

	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	// Register handlers.
	d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered(craftyID, done)),
		gatt.PeripheralConnected(onPeriphConnected(craftyID, done, os.Args[1:])))
	d.Init(onStateChanged)
	select {
	case <-done:
		d.StopScanning()
		return
	}
}
