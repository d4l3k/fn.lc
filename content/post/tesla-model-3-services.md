---
title: "Hacking my Tesla Model 3 - Internal API"
date: 2020-05-26T07:34:34-07:00
---

*This is a follow up to [Hacking my Tesla Model 3 - Security Overview](/post/tesla-model-3/).*

This is a technical description of all the internal services I've found and
notes about how they work.

All of these services described are normally unaccessible due to seceth and
firewall rules.

## Hosts

```
192.168.90.100 cid ice
192.168.90.100 ic
192.168.90.102 gw
192.168.90.103 ap ape
192.168.90.104 lb
192.168.90.105 ap-b ape-b
192.168.90.30 tuner
192.168.90.60 modem
```

Tuner isn't present on newer Model 3s as the AM/FM radio has been removed. I'm
not sure what lb is.


## Data Values

Much of the UI and CID/ICE services are based off of "data values". These are
shared KV pairs between most of the services running on the UI. These appear to
be propagated between services and nodes via UDP multicast.

Many of the values seem to be "read only" such as the `VAPI_` values which seem to
come from the Gateway. Some values are propagated back to the Gateway and the
rest of the car such as the various GUI requests (ex: `GUI_blinkerRequest`,
`GUI_pedestrianWarningMuteRequest`).

A lot of these data values are not Model 3 specific and are only present in the
Model S/X and have little to no effect.

Here's the
[list of all the data values](/tesla-model-3-services/data_values.txt)
provided when you query the car. This is only the keys and not the values
themselves since there might be personal/car specific information in there. If
you do have any questions on specific ones feel free to reach out to me.

You can query the DebugService at port 4035 to view and set these values.

### Fetching all Data Values

```bash
$ curl http://cid:4035/Debug/get_data_values?format=csv&show_invalid=true
```

### Setting a Data Value

```bash
$ curl http://cid:4035/set_data_value?name=GUI_trackMode&value=true
```

## CID/ICE Services

See the full list of QT based services on CID/ICE + ports at
[services.cfg](/tesla-model-3-services/services.cfg). This config appears to be
how the various services know how to talk to each other. Port is typically a
HTTP service. DataPort I suspect is for propagating data values between
services but I'm not 100% sure.

### Hermes

Hermes is the secure tunnel that the car uses to talk to Tesla.

See [Security Overview: Hermes](/post/tesla-model-3/#hermes---talking-to-the-mothership).

### Odin

Odin is a service running on the car for service/support/engineering can use to
debug and control the car.

See [Security Overview: Odin](/post/tesla-model-3/#odin---service-interface).

### DebugService - :4035

This service lets you get/set data values as well as get other metadata about
the car.

```
/Debug/vitals
/Debug/get_data_values?format=csv&show_invalid=true
/set_data_value?name=<>&value=<>
/do_not_sleep?minutes=15&reason=updater
/mothership_bandwidth
/emmc_vitals
/ServiceStatus
/Debug/get_data_values
```

### VehicleService - :4030

Only really seeing things calling it for alerts.

```
/sendAlert
/requestAccPower
/sendAlertForUnexpectedExetingProcess
/alert_kernel_panic
/sendGpuHangAlert
```

### TelemetryService - :4032

```
/send_network_data
/yubikey_added
/yubikey_removed
```

The udev rules on the car send back information about any yubikeys inserted into
the car. I'm not really sure why Tesla has yubikey support. It's possibly for
development purposes or factory purposes for 2FA to provision the car?

### CenterDisplay/QtCar interface - :4070

This is the graphical QT app running on the car display.

```
/display_message
/screenshot
/ensure_power_on
/setStorageRunningLow
/_data_set_value_request_
/update_spool_folder_created
/update_failed?early_failure=true
```

You can display messages on the screen via:

```
$ curl http://192.168.90.100:4070/display_message?message=owned
```

{{% amp-img src="/images/tesla-model-3/model-3-owned.jpg" %}}
Displaying messages on the screen using the internal API. Version 2020.12.11.1
{{% /amp-img %}}

### CarServer - :7654

```
/alerts
/ServiceStatus
/diag_vitals
/get_message_names
/upcoming_calendar_entries
/update_remote_files
/schedule_software_update
/cancel_software_update
```

#### /diag_vitals

This returns a big blob of JSON with diagnostic vital info about the car.

#### /Alerts

```
$ curl http://192.168.90.100:7654/Alerts
{  }
```

#### /ServiceStatus

```
$ curl http://192.168.90.100:7654/ServiceStatus
{"uptime":7598, "requests":68, "avgtime":-1.0, "maxtime":0, "info":""}
```

## APE/ICE Updater

The updaters are used to update the car. There's several variants of them
depending on what the binary is named.

The updater can either run as a CLI or as a service. There's a number of
commands only present as CLI presumably for debugging purposes. A subset of the
CLI commands are present as part of the HTTP.

Some of the more powerful commands are only enabled for HTTP when the car isn't
fused (presumably for development/factory purposes).

When running as a service they have two ports open. One for HTTP and one for
Telnet. You can see more about the telnet/CLI interface at https://github.com/lewurm/blog/issues/5

The updaters are running servers internally at:

* CID: `http://192.168.90.100:20564` and telnet is on `25956`
* APE: `http://192.168.90.103:28496`
* APB: `http://192.168.90.105:28496`

### HTTP Methods

#### /readsig

```
$ curl http://192.168.90.103:28496/readsig
installed_firmware_signature:
<base64 hash>
offline_firmware_signature:
<base64 hash>
```

#### /status

ICE

```
$ curl http://192.168.90.100:20564/status
Executable: /deploy/ice-updater, personality: ice-updater, hash <hex hash>, built for package version: <version>
uptime: <uptime>

/proc/uptime:
<uptime>


current bootdata Contents: <hex data>

Pattern:             0xdeadbeef
Online boot bank:    KERNEL_B
Online fail count:        0
Online dot-model-s size:  1243770944
Offline boot bank:        KERNEL_A
Offline fail count:       0
Offline dot-model-s size: 1209049152
MCU Board Revision:
Fused: 1
Override-version: 0

Online map bank: BANK_A
Online map package size: 5541724224
Online map signature:
<base64 hash>
Offline map bank: BANK_B
Offline map package size: 0
Offline map signature: NULL

Game: purple
Online games bank: BANK_B
Online games package size: 1134821440
Online games signature:
<base64 hash>
Offline games bank: BANK_A
Offline games package size: 1134821440
Offline games signature:
<base64 hash>

Game: teardrop
Online games bank: BANK_B
Online games package size: 1235193920
Online games signature:
<base64  hash>
Offline games bank: BANK_A
Offline games package size: 1235193920
Offline games signature:
<base64 hash>

running_in_recovery_partition = 0
installed_firmware_signature =
<base64 hash>
offline_firmware_signature =
<base64 hash>
staged_update = no
gateway_needs_update = no
updating_maps = yes

END STATUS
```

APE

```
$ curl http://192.168.90.103:28496/status
Welcome to Gadget Updater - <version> (<hex>)
Uptime: <time>
Current State: Waitjob
Session ID: <id>
Update Phase: Idle
Swapped: Online
Staged: false
Fused: true

Online Bank
Bank: KernelA
Size: <size>
Failcount: 0

Offline Bank
Bank: KernelB
Size: <size>
Failcount: 0

Installed
Signature:<base64 hash>
Offline
Signature:<base64 hash>

Job Details
Type: NoActiveJob
Install URL:
Install Path:
Package Signature:
```

#### /install

This sets a URL to download an image from and install it. These images need to
have a valid signature. There's also supposedly a mechanism to prevent
downgrades which unfortunately makes it so we can't downgrade to an known
vulnerable firmware.

Normally this is used by the cid-updater to transfer the APE firmware image from
the CID to the APE to be installed. The APE does appear to have a direct
internet connection so I'm not sure why this is strictly necessary. Might be to
share the same update code with the S/X cars.

```
$ curl http://192.168.90.103:28496/install?http://bar
```

If you run /status again it shows:

```
Job Details
Type: FullDownload
Install URL: http://bar
Install Path: /dev/sda5
Package Signature:
```

#### /reset

I haven't tried this.

#### /gostaged%20status

I believe it returns the update status.

#### /handshake

Does a handshake with the Tesla mothership to check for firmware updates.

#### /service-redeploy

Triggers a service redeploy installation of the firmware already present on car.
