---
title: "swolectl: I wrote an open source controller for my Tonal"
date: 2026-08-10T00:00:00-07:00
---

_Originally posted to [r/tonalgym on Reddit](https://www.reddit.com/r/tonalgym/comments/1vkvfq7/swolectl_i_wrote_a_open_source_controller_for_my/)._

Hey folks, I started a project to provide a completely custom controller for
the Tonal 1. This replaces the Android tablet and directly talks to the motor
controller using the internal USB serial connection.

This is super barebones and not well tested, so you're on your own if it bricks
your system. I figured I should publish it and see if folks were interested in
contributing things such as workouts, programs, etc. The standard Tonal
subscription is good enough for me, so I'm not sure how much I'll actually push
on this.

The code is at [github.com/d4l3k/swolectl](https://github.com/d4l3k/swolectl).

## Architecture

Tonal is effectively two main pieces: an Android 6.0 system and app running on
a MediaTek chip, and a motor controller with its own firmware running on a
Luminary Micro microcontroller. These talk to each other using a simple protocol
over a USB serial connection.

Most of the core weight features are actually implemented by the firmware on
the motor controller and not the app. This makes it very easy to send commands
to enable things like eccentric mode and others.

The protocol and firmware could potentially change down the line, so please be
cautious. I'm using firmware `5.2.18.0`. Tonal seems to flash this at runtime
periodically, so even if they change it, you should be able to revert the
firmware as long as you have a copy of it.

Full details are in the [protocol documentation](https://github.com/d4l3k/swolectl/blob/main/docs/protocol.md).

## Misc Findings

There seem to be a lot more exercise modes than are actually exposed currently.
I'm curious what these do, especially perturbation, but I haven't tested any of
them. I've only tested features that are exposed by the Tonal interface.

| Value | Curve |
| --- | --- |
| 0 | Chains |
| 1 | Bell with eccentric |
| 2 | Reverse chains with eccentric |
| 3 | Chains with eccentric |
| 4 | Flat with eccentric (Smart Flex mapping) |
| 5 | Quadratic ascending with eccentric |
| 6 | Perturbation |
| 7 | Reverse chains |
| 8 | Eccentric reduction |

## Why?

I got accused of lying about rooting my Tonal because I don't want to enable
pirating the Tonal content and deal with all the legal implications. I pay for
a subscription because I like the programs and the product.

I figured I'd make a better, fully local option that can be shared and can't be
patched by Tonal. If all you need is custom workouts and a better way of
managing them, this is perfectly sufficient. I like the programs that Tonal
provides, but it's nice to have a completely offline option if Tonal ever goes
out of business.

This is very much inspired by [Valetudo](https://valetudo.cloud/) for robot
vacuums. Let me know if you try it out. I'm happy to help debug or troubleshoot.
