# owkit

## os (raspbian)

### enable ssh on boot

`touch ssh` in sd-card filesystem

### config.txt

look in `config.txt` file

### run as deamon

Run as service, log to file ex:
`setsid ./owkit >./logfile 2>&1 < /dev/null &`

### alter gpio from cmd

`/sys/class/gpio`

First write no to `export`, then read or set using `gpioXX/value`


