# cricket

A simple process which reads systemsinformation on linux and prints it out
periodically. It will collect and print information on cpu, memory, disk, and
network.

## Usage

```
# cricket
...
~ INFO -- cpu stats (diff) -- cpuIOWait="0" cpuIRQ="0" cpuIdle="186" cpuNice="0" cpuSoftIRQ="0" cpuSystem="8" cpuUser="6"
~ INFO -- net stats (diff) -- dev="eth0" rcvBytes="0" rcvDrop="0" rcvErrs="0" rcvPackets="0" txBytes="0" txDrop="0" txErrs="0" txPackets="0"
~ INFO -- net stats (diff) -- dev="wlan0" rcvBytes="52" rcvDrop="1" rcvErrs="0" rcvPackets="1" txBytes="0" txDrop="0" txErrs="0" txPackets="0"
~ INFO -- net stats (diff) -- dev="lo" rcvBytes="0" rcvDrop="0" rcvErrs="0" rcvPackets="0" txBytes="0" txDrop="0" txErrs="0" txPackets="0"
~ INFO -- net stats (diff) -- dev="tun0" rcvBytes="0" rcvDrop="0" rcvErrs="0" rcvPackets="0" txBytes="0" txDrop="0" txErrs="0" txPackets="0"
~ INFO -- mem stats -- memAvailKB="2102500" memTotalKB="3943816" memUsedKB="1841316" memUsedPer="87"
~ INFO -- disk usage stats -- bytesAvail="98042679296" bytesTotal="206285275136" bytesUsed="97740271616" fs="/dev/sda2" mountPoint="/"
~ INFO -- disk usage stats -- bytesAvail="64243712" bytesTotal="104634368" bytesUsed="40390656" fs="/dev/sda1" mountPoint="/boot"
~ INFO -- disk io stats (diff) -- fs="/dev/sda1" ioMillis="0" readMillis="0" readSectors="0" readsCompleted="0" readsMerged="0" weightedIOMillis="0" writeMillis="0" writesCompleted="0" writesMerged="0" writtenSectors="0"
~ INFO -- disk io stats (diff) -- fs="/dev/sda2" ioMillis="0" readMillis="0" readSectors="0" readsCompleted="0" readsMerged="0" weightedIOMillis="0" writeMillis="0" writesCompleted="0" writesMerged="0" writtenSectors="0"
...
```

The above will be printed out periodically (with numeric values differing each
time of course). The interval at which each type of stat is collected can be
changed through command-line parameters.

## Stats

Here's some description on each stat, how they are collected, and what they
mean:

* `cpu stats (diff)` - Displays the number of "jiffies" (a length of time
  defined in the kernel) the cpu has spent doing each type of action since the
  last `cpu stats (diff)` message.

  See the */proc/stat* section on the [proc man page][proc] for more details.

* `net stats (diff)` - Displays the number of events, with one event type per
  field, which have occured since the last `net stats (diff)` message. One
  message is displayed per interface. It is *not* necessary to restart cricket to
  account for interfaces being added and removed.

  See the */proc/net/dev* section on the [proc man page][proc] for more details.

* `mem stats` - Displays the current state of system memory usage.

  See the */proc/meminfo* section on the [proc man page][proc] for more details.

* `disk usage stats` - Displays the current state of the disk usage on the
  system, one message per partition. This uses the `df` command to obtain its
  info, and will only display for devices which can be found in `/dev/` (this
  will exclude tmpfs, for example). It is *not* necessary to restart cricket to
  account for interfaces being added and removed.

* `disk io stats (diff)` - Displays the number of events, with on event type per
  field, which have occurred since the last `disk io stats (diff)` message. One
  message is displayed per partition. It is *not* necessary to restart cricket
  to account for interfaces being added and removed.

  See the */proc/diskstats* section on the [proc man page][proc] for more details.

* `ping result` - Displays the average ping time to a host. By default this stat
  will not be used, but it can be enabled using `--ping-hosts`. It require's
  superuser to work, or you can use `setcap cap_net_raw=+ep` on the binary.

[proc]: http://man7.org/linux/man-pages/man5/proc.5.html
