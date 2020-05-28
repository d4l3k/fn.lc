---
title: "Hacking my Tesla Model 3 - Software Modes"
date: 2020-05-26T20:34:09-07:00
---

*This is a follow up to [Hacking my Tesla Model 3 - Internal API](/post/tesla-model-3-services/).*

As part of reverse engineering the Tesla Model 3 internals, I've been running a
subset of the CID car services to see how they work.

The car computers are using Intel Atom based processors so it's easy to setup a
chroot to launch the services.

I've written two helper scripts to set up the car environment:

1. [chroot.sh](/tesla-model-3-modes/chroot.sh) - runs a command in the Tesla
   chroot environment.
2. [start_car.sh](/tesla-model-3-modes/start_car.sh) - runs the main vehicle
   services: escalator, ecall_client, sim_service, carserver,
   vehicle.
   Other services can be launched via `./chroot.sh /usr/tesla/UI/RunQtCar <service>`

These scripts assume a CID image is extracted at `squashfs-root/`.

## Running QtCar

> WARNING: This runs the Tesla services in a chroot with root permissions. This
> could mess up your system since it has root access to a good chunk of `/dev`
> and `/proc`.

```bash
# add the IP addresses to localhost
sudo ip addr add 192.168.90.0/24 dev lo

# start the background services in tmux
sudo ./start_car.sh

# figure out what device is your touchscreen
sudo evtest

# start the car computer
sudo ./chroot.sh bash
su -c '/usr/tesla/UI/RunQtCar cid --touch evdev:/dev/input/event9 --window 1 --no-map' -s /bin/bash tesla
```

## Modes

These are all the special modes I've found in the Model 3 UI.

### Diagnostic Mode

Enabled by setting `GUI_diagnosticMode` to true.

The only thing it seems to do is to show some of the hidden alerts that are
normally hidden from the driver.

### TDS Mode

Enabled by setting `GUI_tdsMode` to true.

This enables the diagnostic app as well as shows the alerts from diagnostic
mode. You can open and close the diagnostic app by pressing the Tesla logo.

{{% amp-img src="/tesla-model-3-modes/tds-mode.png" %}}
The diagnostic app data viewer. This has stats from all the various car systems.
{{% /amp-img %}}

{{% amp-img src="/tesla-model-3-modes/tds-mode-das.png" %}}
Stats from the DAS.
{{% /amp-img %}}

{{% amp-img src="/tesla-model-3-modes/tds-mode-can.png" %}}
This screen shows stats from the CAN bus messages. You can interactively select
which stats you want to be able to view.
{{% /amp-img %}}

{{% amp-img src="/tesla-model-3-modes/tds-mode-actions.png" %}}
The various diagnostic actions.
{{% /amp-img %}}

{{% amp-img src="/tesla-model-3-modes/tds-mode-settings.png" %}}
Some internal settings. Top speed and park assist are possibly from before speed
limit mode and ultrasonic sensor parking assists were added to the standard UI.
{{% /amp-img %}}


#### Apps

{{% amp-img src="/tesla-model-3-modes/tds-mode-apps.png" %}}
All the apps in the car. The Diagnostic app is the currently open app.
{{% /amp-img %}}

{{% amp-img src="/tesla-model-3-modes/tds-mode-image-viewer.png" %}}
The image viewer. I don't have any images so doesn't show anything.
It might be for viewing calibration images from the DAS?
{{% /amp-img %}}

{{% amp-img src="/tesla-model-3-modes/tds-mode-nav-test.png" %}}
Nav Test seems to be for testing the map and navigation. It supports recording a
GPS path and then replaying it. This app is completely hidden by default but can
enabled by changing the app config file on disk.
{{% /amp-img %}}

{{% amp-img src="/tesla-model-3-modes/tds-mode-nvh.png" %}}
Noise, vibration and handling app. This seems to be for testing the car on
rollers to see what sounds the car makes. It appears to be able to use the cars
internal microphone to record what it sounds like inside the cabin.
{{% /amp-img %}}

{{% amp-img src="/tesla-model-3-modes/tds-mode-wifi.png" %}}
This seems to be for debugging the LTE connection, WIFI and radio (parrot).
{{% /amp-img %}}

{{% amp-img src="/tesla-model-3-modes/tds-mode-sketch-pad-2.png" %}}
The car has both the old sketch pad and the new sketch pad even though only the
new one is accessible by default.
{{% /amp-img %}}

### Developer Mode

Enabled by setting `GUI_developerMode` to true.

It shows all of the alerts on the car even those hidden to diagnostics mode.

Pressing the Tesla logo throws an error in the
console about trying to open the diagnostics app but it being disabled. Enabling
`GUI_tdsMode` with `GUI_developerMode` shows some new tabs.

{{% amp-img src="/tesla-model-3-modes/developer-mode-fonts.png" %}}
The new fonts tab in the diagnostic app. Appears to be just for testing the
different font appearances.
{{% /amp-img %}}

{{% amp-img src="/tesla-model-3-modes/developer-mode-factory.png" %}}
The new factory tab in the diagnostic app.
{{% /amp-img %}}


{{% amp-img src="/tesla-model-3-modes/developer-mode-trial-car.png" %}}
Enabling Factory Mode and then turning on "Trial Car". Doesn't appear to do
anything else to the UI. Presumably there's some corresponding changes to the
drive systems.
{{% /amp-img %}}

{{% amp-img src="/tesla-model-3-modes/factory-mode-overrides.png" %}}
Enabling the factory mode overrides.
{{% /amp-img %}}
The factory mode overrides change `GUI_torqueLimitRequest` (default 63) and
`GUI_powerLimitRequest` (default 31).

I'm not sure exactly what setting the
torque limit to 6500 Nm means when the car is listed as having 750 Nm of total
torque.

#### Factory Summon

Factory summon seems to be a special form of summon just used in the factory.

Here's all the data values related to it:

```
GUI_factorySmartSummonEnable,false
GUI_factorySmartSummonStart,false
GUI_factorySummonConveyor,false
GUI_factorySummonHeartbeat,0
DAS_factoryGoalLat,0.000
DAS_factoryGoalLong,0.000
DAS_factorySummonStatus,UnavailableNoQRInstruction
```

`SummonConveyor` seems to imply they're using it as a virtual "conveyor
belt" in the factory. It seems likely once the drive train is in the car they have the
car drive itself down the factory line. `UnavailableNoQRInstruction` implies
that the DAS system is using the cameras on the car to read QR codes that tell
the car where to go and they navigate there using smart summon + GPS
coordinates.

It would be super cool if they actually use smart summon in the factory and
probably makes a lot of sense given QR codes are a lot easier to setup than a
conveyor belt.

### DAS Debug Mode / DAS Developer

Enabled by setting `GUI_dasDebugOn` and `GUI_dasDevMode` to true.

This didn't have any obvious effects. It didn't seem to do anything when I
enabled it. The Autopilot tab in the settings is broken in the chroot and when I
tried enabling it on my car there wasn't anything notable either. Possibly you
need the car to have FSD enabled for the debug mode to show up?

### Game Developer Mode

Enabled by setting `GUI_gameDeveloperMode` to true.

This doesn't have any obvious effects. Possibly enables USB debugging or debug
data from the games? I didn't dig too deep into it.

### Factory Mode

Enabled by setting `GUI_factoryMode` to true.

{{% amp-img src="/tesla-model-3-modes/factory-mode.png" %}}
Factory Mode by itself doesn't seem to add anything to the UI.
{{% /amp-img %}}

By itself it doesn't seem to do much. Enabling `GUI_tdsMode` and
`GUI_developerMode` shows a new
"Factory" tab with some more options (see the Developer Mode section).


### Service Mode

Enabled by setting `GUI_serviceMode` to true.

{{% amp-img src="/tesla-model-3-modes/service-mode.jpg" %}}
The service mode tab. Override Service Limits removes the ~3 mph limit.
{{% /amp-img %}}

Service Mode is what Tesla Service centers and authorized body shops put your
car into when it's being worked on. It adds a ~3 mph speed limit, disables
remote access and disables the dash cams. It also allows for certain operations
such as redeploying the firmware after replacing hardware.

{{% amp-img src="/tesla-model-3-modes/service-mode-system-checks.jpg" %}}
The service mode tab. Override Service Limits removes the ~3 mph limit.
{{% /amp-img %}}

{{% amp-img src="/tesla-model-3-modes/service-mode-actions.jpg" %}}
The service mode tab. Override Service Limits removes the ~3 mph limit.
{{% /amp-img %}}

### Transport Mode

Enabled by setting `GUI_transportMode` to true.

{{% amp-img src="/tesla-model-3-modes/transport-mode.png" %}}
Transport mode.
{{% /amp-img %}}

This mode is pretty minor. I believe it limits the speed the car can drive. It
might also make the car sleep/conserve battery. There's no obvious UI changes.

### Showroom Mode

Enabled by setting `GUI_showroomMode` to true.

No obvious UI changes. Cars in Tesla showrooms have this mode set. It keeps the
car awake and disables certain options such as Pin to Drive.

### Dynotest Mode

You can enter it by entering `dynotest` into the text box that appears after
long pressing the Tesla logo. Setting `GUI_dynotestMode` to true didn't seem to enable it.

This disables traction control for use with dyno testing.

{{% amp-img src="/tesla-model-3-modes/dyno-mode.jpg" %}}
Dyno mode.
{{% /amp-img %}}

### Performance Demo Mode

Enabled by setting `GUI_performanceDemoMode` to true.

This doesn't seem to do anything on the Model 3. Possibly this is a ludicrous
demo for the Model S.

