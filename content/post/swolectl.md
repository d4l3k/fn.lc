---
title: "swolectl: an open source controller for my Tonal"
date: 2026-08-10T00:00:00-07:00
---

_Originally posted to [r/tonalgym on Reddit](https://www.reddit.com/r/tonalgym/comments/1vkvfq7/swolectl_i_wrote_a_open_source_controller_for_my/)._

I built [swolectl](https://github.com/d4l3k/swolectl), an open source controller
for the original Tonal. It replaces the Android tablet and talks directly to the
motor controller over its internal USB serial connection. The result is a fully
local web interface for setting resistance, selecting modes, and viewing live
telemetry—without depending on Tonal's app or cloud services.

This is still experimental and only tested on my hardware, but the code and the
protocol notes are public for anyone interested in testing, debugging, or
building workouts and programs on top of it.

![swolectl's local web interface](/swolectl/ui.png)

## Why build a replacement controller?

I like Tonal's programs and continue to pay for the subscription. This project
isn't intended to pirate Tonal's content or reproduce its service. I wanted a
separate, completely local option for custom workouts—and a way to keep the
hardware useful if the official service ever disappears.

The idea is similar to [Valetudo](https://valetudo.cloud/) for robot vacuums:
replace the cloud-connected control layer while continuing to use the hardware
you own. Because swolectl communicates directly with the motor controller, it
doesn't rely on rooting the Android tablet and can't be removed by an Android
app update.

![A small Linux computer running swolectl beside the Tonal](/swolectl/setup.png)

## How the Tonal is put together

At a high level, the original Tonal contains two mostly independent systems:

1. An Android 6 tablet and the Tonal application running on a MediaTek system.
2. A dedicated motor controller with its own firmware on a Luminary Micro
   microcontroller.

The two communicate using a compact binary protocol over USB CDC ACM. Most of
the important resistance behavior lives in the motor-controller firmware, not
the Android application. Once I decoded the framing, checksums, bring-up
sequence, resistance profiles, and telemetry, a small Python application could
take the tablet's place.

swolectl currently implements the USB serial transport, device discovery,
experimental bring-up, bounded resistance commands, enable and disable
controls, rep counting, cable load telemetry, and a local web control panel. It
also has a receive-only diagnostic mode for inspecting a controller without
sending motor commands.

The web UI supports Basic, Spotter, Drop Sets, Chains, Eccentric, and Smart Flex
modes. One of the more interesting discoveries was that the firmware contains
additional resistance curves which the stock interface does not expose,
including reverse chains, quadratic ascending resistance, eccentric reduction,
and a mode named perturbation. I have documented their identifiers, but have
not tested or assigned behavior to the unknown modes.

## Connecting directly to the controller

Behind the display, the Android computer connects through a white five-pin
connector labeled `tablet`. That connection exposes USB and a wake signal. For
my setup, I disconnect the tablet and attach a Linux host to the motor
controller over USB instead.

![The tablet connector behind the Tonal display](/swolectl/tablet-connector.png)

![The labeled connectors on the Tonal's Android computer](/swolectl/tonal-connectors.png)

Opening the machine requires removing security screws hidden behind the right
arm. Power the unit off before opening it. The tested controller is also
sensitive to startup order: the Tonal must be powered on and allowed to
initialize before USB is connected to the replacement host.

The full connection procedure and current protocol documentation live in the
[swolectl repository](https://github.com/d4l3k/swolectl) and the
[serial protocol specification](https://github.com/d4l3k/swolectl/blob/main/docs/protocol.md).

## A serious safety warning

This software directly commands exercise machinery capable of producing
substantial force. A bug, incorrect command, wiring mistake, or incompatible
firmware could cause unexpected movement, equipment damage, or injury. It may
also void the hardware warranty or brick the controller.

Everything so far has been validated only on a Tonal 1 with motor-controller
board `501-0100_rev004` and firmware `5.2.18.0`. Other Tonal 1 revisions and
firmware versions are unverified; Tonal 2 is unsupported. swolectl fails closed
when it sees an incompatible firmware version, requires explicit opt-in for
motor commands, and applies resistance bounds, but those checks are not a
substitute for a physical emergency disconnect and careful controlled testing.

The project is early, bare-bones, and not affiliated with or supported by
Tonal. If you try it, please read the repository's safety notes first—and if
you'd like to contribute workouts, programs, hardware testing, or protocol
research, [open an issue or pull request](https://github.com/d4l3k/swolectl).
