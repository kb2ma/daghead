# Using OpenWSN in RIOT

OpenWSN was integrated into RIOT as a package in 2020-06 with PR 13824. This document summarizes setup that worked for me.

See documentation:

  * `tests/pkg_openwsn/README.md`
  * `pkg/openwsn/doc.txt`
  * comments in `dist/tools/openvisualizer.inc.mk`

Using samr21-xpro for all nodes. Also removed use of channel hopping in flash commands.

See section below on eliding join mode.

## Root node

Flash with regular USB cable using command below.

```
   SERIAL=ATML2127031800001334 OPENSERIAL_BAUD=19200 USEMODULE=openwsn_serial BOARD=samr21-xpro CFLAGS="-DIEEE802154E_SINGLE_CHANNEL=26" make flash -j4
```

Run with USB TTL serial converter cable (3.3V logic).

For my cables:

|Wire       |samr21 pin          |
|-----------|--------------------|
|5V (red)   |5V0 IN on PWR header|
|GND (black)|GND on PWR header   |
|RXD (white)|PA22                |
|TXD (green)|PA23                |

After flashing or powering up, use OpenVisualizer. See that section below.


## Other nodes

Flash and run with regular USB cable. See command below. When running, the RIOT terminal menu includes good diagnostic information.

```
BOARD=samr21-xpro CFLAGS="-DIEEE802154E_SINGLE_CHANNEL=26" make all -j4
BOARD=samr21-xpro SERIAL=ATML2127031800001700 make flash-only
```

## OpenVisualizer

RIOT does include a make file in dist/tools/openvisualizer, but I found it easier to just build/run OpenVisualizer from a separate openwsn-sw checkout.

### Setup

Must run in Python 2.7. This is becoming difficult in Ubuntu. In 2020.04, must install python2 pkg in apt. Also must install pip2 and virtualenv manually. I installed both of these using sudo. For pip: `python2 get-pip.py`. For virtualenv, `pip2 install virtualenv`.

Must use fjmolinas branch of OpenVisualizer. Use:

```
git clone -b develop_SW-318-RIOT https://github.com/fjmolinas/openvisualizer.git
```

Must actually install OpenVisualizer. Use `pip2 install -e .` without sudo, from openvisualizer directory. Notice that the `-e` parameter somehow allows you to run/use an openwsn-sw development directory. Requires further investigation.

### Running

Set up a `run` directory. Copy all `openvisualizer/config/*conf` files here.

Assuming root node is powered up from USB TTL serial converter, run:

```
$ sudo su
# . ../share/venv/bin/activate
# openv-server --fw-path "/home/kbee/dev/riot/repo/build/pkg/openwsn" \
   --port-mask "/dev/ttyUSB0" --baudrate 19200 --opentun --root "/dev/ttyUSB0" \
   --lconf "logging.conf"
```
Not sure that 'sudo' is required, but probably.

## Eliding Join mode

OpenWSN includes a secure join mode for motes, Constrained Join Protocol (CoJP). This mode requires CoAP and an involved encryption mechanism, and would require significant effort to implement for daghead, so, we have elided its use for now. See below for the updates to individual projects.

The goal of CoJP is transmission and sharing of a random 16-byte key used to encrypt link layer beacons (I think). So, to elide use of CoJP we hard code a static key in the DODAG root (border router) and in the network motes. We also ensure the mote does not send the CoAP POST to retrieve the key.

| Project        | Branch              | Notes                         |
|----------------|---------------------|-------------------------------|
| openwsn-fw     | elide-cjoin         |                               |
| openvisualizer | elide-cjoin         |                               |
| riot           | openwsn/elide-cjoin | Only to use openwsn-fw branch |



