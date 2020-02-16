# owkit

Small application turning Raspberry Pi with 1-wire temperature sensors into temperature logger and (optio) thermostat with other useful functions.

Logging is possible to Influx DB (just set in config) and through http request (POST and GET supported).

Documentation will be updated :) 

## os config

### enable ssh

To enable ssh server before first boot (default is disabled) add file `ssh` in sd-card filesystem:
```
touch ssh
```

### config.txt

Too turn on 1-wire add following line at the end of `config.txt` file:
```
# 1-wire
dtoverlay=w1-gpio
```

Next line will set selected gpio pin (here *21*) in ouput mode and default LOW state
```
# gpio output, def ON (lo) SSR(+) connected to 5V bus, SSR(-) to gpio21
gpio=21=op,dl
```

### download owkit

TODO

### run as deamon

To run as a service and log to file you can use command:
```
setsid owkit > /var/owkit.log 2>&1 < /dev/null &
```

If you want to run *owkit* on startup you can add this line to the `/etc/rc.local` file.


## troubleshooting

Some useful commands and info

### alter gpio from cmd

`/sys/class/gpio`

First write gpio number to `export`, then read or set using `gpioXX/value`

### cross compile go

To build binary for raspbian:
```
GOOS=linux GOARCH=arm GOARM=5 go build
```


## Changelog

### v0.1
First release, working