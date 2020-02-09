# gocrafty

`gocrafty` controls your S&B Crafty vaporizer.

## how 2 use dis?

Set `CRAFTY_SN` environment variable to your device's serial number 
(you'll find it on the label on the bottom of the vape.)

Without arguments, `gocrafty` will connect to the vape, read some configuration
and current status, print it, then print out every (supported) value that changes,
_aka_ monitoring mode.

Example:
```
# CRAFTY_SN='CYA69420' ./gocrafty           
State: PoweredOn
scanning...
Connecting to 04:20:00:69:69:69 [STORZ&BICKEL]
Current Temp: 24.5 C
Setpoint: 178 C
Boost: +20 C
Battery level: 100%
LED brightness: 11%
Model: [Crafty          ] SN: [CYA6942000] FW: [V02.52] ID [04:20:00:69:69:69]
Chariging indicator: OFF
Current Temp: 24.4 C
  .
  .
  .
Current Temp: 24.4 C
```


To change settings use flags:
```
  -set-boost int
        set boost value (positive only) 
  -set-charge-indicator string
        set charging indicator ON or OFF
  -set-temp int
        set base vape temperature point 
  -turn-on
        turn the Crafty ON remotely
```

After using those options, the command will quit (no monitoring.)


## Notes

This is using a forked version of paypal/gatt because I needed to read service data,
so that I could connect to known S/N similarly to the official, now virtually
unavailable app.

More functionality coming soon, like nicer CLI and more config options.

## Disclaimer

This is (not even) beta quality software, it can have unknown bugs (probably does.)

If you use this, you're doing it on your own risk. Don't blame me if you screw up your vape. 

You have been warned.

