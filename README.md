# daghead

RPL DODAG router app for an OpenWSN 6TiSCH network.
Reads incoming data from root mote, and performs routine management.

Since OpenWSN operates RPL in non-storing mode, reads ICMPv6 RPL messages to maintain a
routing table for the network motes.

Also:

  * Prints error notifications from mote to daghead log.
  * Sets the root mote as DODAG root. Assumes root mote is not already DODAG root.
    Presently avoids use of Constrained Join Protocol for network motes by using a
    static network key hardcoded into mote firmware.

## Building and running

See implementation notes in [README-openwsn.md](./README-openwsn.md).


