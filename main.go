package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pan-maruda/gatt"
	"github.com/pan-maruda/gocrafty/ble"
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

func onPeriphDiscovered(craftySerialNumber string, done <-chan bool) func(gatt.Peripheral, *gatt.Advertisement, int) {

	return func(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
		for _, serviceData := range a.ServiceData {
			if serviceData.UUID.String() == "0052" {
				snFromDevice := string(serviceData.Data)
				if snFromDevice[:8] == craftySerialNumber {
					fmt.Printf("Connecting to %s [%s]\n", p.ID(), p.Name())
					p.Device().StopScanning()
					p.Device().Connect(p)
				}
			}
		}
	}
}

func onPeriphConnected(craftyID string, done chan bool, action func(gatt.Peripheral, *ble.DataService, *ble.SettingsService)) func(gatt.Peripheral, error) {

	return func(p gatt.Peripheral, err error) {

		services, err := p.DiscoverServices([]gatt.UUID{ble.DataServiceUUID, ble.MetaServiceUUID, ble.SettingsServiceUUID})
		if err != nil {
			log.Fatalf("Failed to discover services, err :%s\n", err)
			return
		}

		var dataSvc *ble.DataService
		var settingsSvc *ble.SettingsService

		for _, svc := range services {

			if svc.UUID().Equal(ble.DataServiceUUID) {
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

			if svc.UUID().Equal(ble.SettingsServiceUUID) {
				settings, err := ble.DiscoverSettingsService(p, svc)
				if err != nil {
					log.Printf("Failed to discover settings service on Crafty: %s\n", err)
					continue
				}

				chargeIndicator, err := settings.ChargeIndicatorStatus(p)
				if err != nil {
					log.Printf("Failed to read charging indicator status: %s", err)
				}
				print("Chariging indicator:")
				if chargeIndicator {
					println(" ON")
				} else {
					println(" OFF")
				}
				settingsSvc = settings
			}
		}

		action(p, dataSvc, settingsSvc)
	}
}

func monitor(turnOn *bool) func(p gatt.Peripheral, dataSvc *ble.DataService, settingsSvc *ble.SettingsService) {
	return func(p gatt.Peripheral, dataSvc *ble.DataService, settingsSvc *ble.SettingsService) {
		if dataSvc == nil {
			fmt.Printf("Data service is nil - not discovered? wtf?")
			return
		}
		if *turnOn {
			fmt.Println("Turning Crafty ON...")
			err := dataSvc.TurnOn(p)
			if err != nil {
				fmt.Printf("Failed to send turn on command to Crafty: %s", err)
			}
		}
		dataSvc.SubscribeBattery(p, func(batteryLevel uint16, err error) {
			fmt.Printf("Battery level: %d%%\n", batteryLevel)
		})
		dataSvc.SubscribeTemp(p, func(currentTemp uint16, err error) {
			fmt.Printf("Current Temp: %d.%d C\n", currentTemp/10, currentTemp%10)
		})
		select {}
	}
}

func setValues(temp *int, boost *int, chargeIndicator *string, done chan bool) func(gatt.Peripheral, *ble.DataService, *ble.SettingsService) {
	// todo validate this somewhere else
	return func(p gatt.Peripheral, ds *ble.DataService, ss *ble.SettingsService) {
		if temp != nil && *temp != -1 {
			if *temp > 210 {
				fmt.Println("Temperature cannot exceed 210.")
			}
			if *temp < 0 {
				fmt.Println("Temperature must be positive.")
			}
			fmt.Printf("Setting temperature point to %d\n", *temp)
			ds.SetTemp(p, *temp)
		}

		if boost != nil && *boost != -1 {
			var validBoost int
			if *boost < 0 {
				fmt.Println("Boost must be positive.")
			}
			if *temp+*boost > 210 {
				validBoost = (210 - *temp)
				fmt.Printf("Clamped boost temp to +%d C\n", validBoost)
			} else {
				validBoost = *boost
			}

			if validBoost != 0 {
				fmt.Printf("Setting boost temp to +%d C\n", validBoost)
				ds.SetBoost(p, validBoost)
			}
		}

		if *chargeIndicator != "" {
			switch *chargeIndicator {
			case "ON":
				println("Turning charge indicator ON.")
				ss.SetChargeIndicatorStatus(p, true)
			case "OFF":
				println("Turning charge indicator OFF.")
				ss.SetChargeIndicatorStatus(p, false)
			default:
				fmt.Printf("Unrecognized option [%s] for charge indicator. Must be ON or OFF.\n", *chargeIndicator)
			}
			// todo cleanup
			// fixme why does this need to be read to work more reliably?
			chargeIndicator, err := ss.ChargeIndicatorStatus(p)
			if err != nil {
				log.Printf("Failed to read charging indicator status: %s", err)
			}
			print("Chariging indicator:")
			if chargeIndicator {
				println(" ON")
			} else {
				println(" OFF")
			}
		}
		done <- true

	}
}

func main() {
	snHelpText := "CRAFTY_SN must be set to the device serial number from the bottom label, like [CYxxxxxx]"

	done := make(chan bool, 1)

	// var oneshot = flag.Bool("oneshot", false, "Read data only once (no notifications)")
	var tempFlag = flag.Int("set-temp", -1, "set base vape temperature point")
	var boostTempFlag = flag.Int("set-boost", -1, "set boost value (positive only)")
	var chargeIndicator = flag.String("set-charge-indicator", "", "set charging indicator ON or OFF")
	var turnOnFlag = flag.Bool("turn-on", false, "turn the Crafty ON remotely")
	flag.Parse()
	craftySn, found := os.LookupEnv("CRAFTY_SN")
	if !found || len(craftySn) != 8 {
		fmt.Println(snHelpText)
		os.Exit(1)
	}

	d, err := gatt.NewDevice(
		gatt.LnxMaxConnections(1),
		gatt.LnxDeviceID(-1, true),
	)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	d.Init(onStateChanged)
	if *tempFlag == -1 && *boostTempFlag == -1 && *chargeIndicator == "" {
		d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered(craftySn, done)),
			gatt.PeripheralConnected(onPeriphConnected(craftySn, done, monitor(turnOnFlag))))
	} else {
		d.Handle(gatt.PeripheralDiscovered(onPeriphDiscovered(craftySn, done)),
			gatt.PeripheralConnected(onPeriphConnected(craftySn, done, setValues(tempFlag, boostTempFlag, chargeIndicator, done))))
	}

	// Register handlers.
	select {
	case <-done:
		d.StopScanning()
		return
	}
}
