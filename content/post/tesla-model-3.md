---
title: "Hacking my Tesla Model 3 - Security Overview"
date: 2020-05-12T23:12:03-07:00
---

*See the follow up at [Hacking my Tesla Model 3 - Internal API](/post/tesla-model-3-services/).*

I recently got a Tesla Model 3 and since I'm a huge nerd I've been spending a
lot of time poking at the systems and trying to reverse engineer/figure out how
to root my car.

I work on Machine Learning infrastructure so I'd love to be able to take a deep
look at how autopilot/FSD works under the hood and what it can actually do
beyond what limited information the UI shows. I know some people have managed to
get a copy of this.

{{% amp-img src="/images/tesla-model-3/model-3-owned.jpg" %}}
Displaying messages on the screen using the internal API. Version 2020.12.11.1
{{% /amp-img %}}

## Existing Research

A lot of the existing knowledge about the internal systems are specific to the
older Model S cars since their security is pretty non-existent. The Model 3 (and
presumably the newer Model S/X/Y) has numerous layers of security measures. The
high level architecture is fairly similar but has been hardened a lot.

### Model 3

* [lewurm's blog posts about his Model 3](https://github.com/lewurm/blog/issues)

### Model S/X
* [green's analysis from his older Model S](https://twitter.com/greentheonly)
* [Lunar's Model S MCU1 info dumps/wiki](https://github.com/Lunars/tesla)
* [Reverse Engineering the Tesla Firmware Update Process](https://www.pentestpartners.com/security-blog/reverse-engineering-the-tesla-firmware-update-process/)
* [freedomEV for Model S MCU1](https://github.com/jnuyens/freedomev)

#### Tencent Keen Security Lab

* [Free-Fall: Hacking Tesla From Wireless To CAN BUS](https://www.blackhat.com/docs/us-17/thursday/us-17-Nie-Free-Fall-Hacking-Tesla-From-Wireless-To-CAN-Bus-wp.pdf)
* [Over-The-Air: How We Remotely Compromised The Gateway, BCM, and Autopilot ECUs Of Tesla Cars](https://i.blackhat.com/us-18/Thu-August-9/us-18-Liu-Over-The-Air-How-We-Remotely-Compromised-The-Gateway-Bcm-And-Autopilot-Ecus-Of-Tesla-Cars-wp.pdf)

## Tesla Security Researcher Program

Before I touched my car at all, I registered as part of the Tesla bug bounty
program and my car is a research-registered vehicle. If you're interested in
poking at your car at all, I'd highly recommend registering as Tesla will try to
fix it if you brick your car.

> If, through your good-faith security research, you (a pre-approved, good-faith
> security researcher) cause a software issue that requires your
> research-registered vehicle to be updated or "reflashed," as an act of
> goodwill, Tesla shall make reasonable efforts to update or "reflash" Tesla
> software on the research-registered vehicle by over-the-air update, offering
> assistance at a service center to restore the vehicle's software using our
> standard service tools, or other actions we deem appropriate.

https://www.tesla.com/about/security

## Internal Layout of the Car

All of the higher level components are connected via an internal Ethernet
switch. These include:

* cid/ice - this is the computer that controls the display and all of the media
  systems such as sound.
  * 192.168.90.100
* autopilot primary and secondary computers.
  * 192.168.90.103 - ap/ape
  * 192.168.90.105 - ap-b/ape-b
* Gateway - this is primarily UDP server that controls the switch, vehicle
  config and proxies requests between the ethernet side (cid/autopilot) and the
  * 192.168.90.102
  CAN BUS to the motor controllers and sensors.
* Modem - this is the LTE modem
  * 192.168.90.60
* Tuner - this is for the AM/FM radio. Not present on the newer Model 3 cars
  including mine. Not having an AM/FM radio does seem like a safety issue so I
  was surprised to see it was removed.
  * 192.168.90.30

## seceth - Secure Ethernet TCAM

The internal car network appears to be using a Marvel 88EA6321 as a switch. This
is an automative gigabit switch.

Most of the connections are using 100BASE-T1 which is a 2 wire PHY for ethernet.
The autopilot computers, modem, tuner, gateway, CID all use 100Base-T1. There's
two standard ethernet ports. One is located on the CID motherboard and has a
standard ethernet jack. The other is located in the driver side footwell and has
a [custom connector](https://teslaownersonline.com/threads/ethernet-port-in-driver-footwell.15045/).

### DSA

The switch appears to be using something called [Distributed Switch
Architecture](https://www.kernel.org/doc/Documentation/networking/dsa/dsa.txt)
and TCAM.

DSA allows the switch to be controlled by a separate processor. In the
Model 3, I believe the Gateway controls it. I haven't seen any references to the
Linux dsa subsystem in the CID.

### TCAM

TCAM is a special type of memory that can do very fast lookups/filters in a
sincel cycle. This allows for the Gateway to specify packet filters for the
switch to apply. By default the ethernet port in the driver side footwell is
disabled by these rules. The diagnostic jack on the CID motherboard can only
access port 8080 (Odin) and 22 (SSH) on the CID.

There is a way to disable the secure ethernet but this seems to be only
accessible via Odin by Tesla engineering and possibly service.

There's apparently a daily changed code that unlocks the diagnostic
port/service mode. Service likely has to get this from Tesla via Toolbox.

## Hermes - Talking to the Mothership

The older Model S cars use a persistent OpenVPN connection to communicate with the
"mothership" as Tesla refers to it. All communication with Tesla go through this
VPN connection so there's no way to sniff any of the updates.

Instead of using OpenVPN, the Model 3 runs a proxy service called Hermes. Hermes
is a relatively simple service that can proxy unauthenticated requests on the
CID to the mothership. Presumably maintaining persistent OpenVPN connections on
500,000+ cars wasn't scalable so they switched to a lower overhead solution.

Hermes also allows Tesla to make requests to the car itself and fetch logs from
it. Presumably this is how Tesla can enable features such as Full Self-Driving
over the air without a full software update as well as do remote service.

### Certificates

Every car is issued unique client certificates for Hermes/OpenVPN and they're
periodically rotated. This makes it quite hard to do things like grabbing
firmware images or inspect Tesla's backend since you first have to get root
access to a car.

These certificates live under `/var/lib/car_creds/car.{crt,key}`.

```
# Phone Home connects to devices over Hermes based on the
# Hermes certificate CN.
...
#     subject=
#     CN=BANGELOM300000001
#     OU=Tesla Motors
#     O=Tesla
#     L=Palo Alto
#     ST=California
#     C=US
```

Each car is issued a specific common name that's only accessible internally to
make it harder for attackers to try and fake a cert. This is relevant for SSH as
we'll see later.


### Binaries

There's a bunch of different hermes binaries. They all seem to be written in
*Go* :). It's nice to see my favorite programming language running in my car.

```
$ ls opt/hermes/
hermes_client*     hermes_fileupload*  hermes_historylogs*  hermes_teleforce*
hermes_eventlogs*  hermes_grablogs*    hermes_proxy*

$ file /opt/hermes/hermes_client
opt/hermes/hermes_client: sticky ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=JRZRLflVY89A6p67rwkt/nb9KmeWMLadrBGvRVujH/aJPtciQz8Xldpa7VcVy_/XzIY9KY7sZI0KdwLYOK5, stripped
```

It's pretty easy to see what OSS libraries they're using in the binary by using
`strings hermes_client | rg vendor/`. Maybe I'll make a follow up post analyzing
Hermes itself.

## Odin - Service Interface

Odin is a python 3 service running on every car. It's used for various
maintenance actions on the car such as calibrating the radar and the cameras. If
you connect to the internal car network you can access it at
http://192.168.90.100:8080.

There's a screenshot of this interface at https://github.com/lewurm/blog/issues/4

If you try to run any of the actions on Odin it just throws an error.

### Odin Authentication

```
{error: "Token 2.0 not found."}
```

I dug into the source code.

*Tesla uses signed certificates for everything.*

From a security perspective this is amazing. :) From a "I want to get root on my
car" perspective it's awful. :(

Each token contains a security level. These levels grant access to different
Odin commands. This allows different tiers of service the minimum permissions
they need to do their job.

These are broken into `principals` and `remote_execution_permissions`.
Presumably `principals` requires physical access via the diagnostic ethernet
port.

The `principals` levels listed in the Odin tasks are:

* tbx-internal
* tbx-external
* tbx-technical-specialist
* tbx-engineering
* tbx-service

These seem to be mostly internal car tests likely used during manufacturing.
The only time the non internal/external principals show up is for
`PROC_ICE_X_LOGS-UPLOADER` and `ICE_DEASSOCIATE_PRODUCT_ID`. The second is
engineering only and appears to wipe the vehicle VIN and car config.

The `remote_execution_permission` levels listed in the Odin tasks are:
* tbx-service
* tbx-service-infotainment
* tbx-technical-specialist
* tbx-service-engineering
* tbx-engineering
* tbx-mothership

Things like `TEST-BASH_ICE_X_SEARCH-UI-ALERTS` can be accessed by `tbx-service`,
`tbx-service-engineering` and `tbx-mothership`.

Things like `PROC_ICE_X_SET-VEHICLE-CONFIG` can only be accessed by
`tbx-mothership`.

The token's are signed by an intermediate certificate. This intermediate
certificate public key is included as part of the token and signed by Tesla's
root CA. From my understanding this follows standard security practices of web
CAs to prevent the root certificate from being compromised.

### Odin Networks

Odin is implemented in a pretty interesting way. There's a list of `tasks` and
`networks`. The tasks are high level actions that can be executed by someone
with specific permissions.

The `lib` files are "networks" that appear to be a domain specific language/UI
program just for creating service tasks.

The networks are very close to JSON but stored in `.py` files.

Here's an excerpt of one:
```py
network = {
...
    "get_success": {
	"default": {"datatype": "Bool", "value": False},
	"position": {"y": 265.22259521484375, "x": 108.96072387695312},
	"variable": {"value": "success"},
	"value": {"datatype": "Bool"},
	"type": "networks.Get",
    },
    "IfThen": {
	"position": {"y": 340.1793670654297, "x": 297.02069091796875},
	"expr": {"datatype": "Bool", "connection": "get_success.value"},
	"if_true": {"connection": "exit.exit"},
	"type": "control.IfThen",
	"if_false": {"connection": "capturemetric.capture"},
    },
...
}
```

Each network is structured as a series of nodes with types describing what they
do. The nodes can consume inputs from other nodes via "connection"s. The actual
logic of each node type is implemented in standard python.

The `position` field seems to indicate that these networks are created via a UI
tool.

### Toolbox

Tesla's service tool is called Toolbox. There seems to be two versions.

1) A program you can download and runs under windows: https://toolbox.teslamotors.com/
2) And a newer web based tool: https://toolbox.tesla.com/

Looking at the source code of the web based tool we see references to the auth
tokens as well as the task names. Presumably this toolbox interface is the front
end to the Odin server that runs on each car.

There's some Russian guy who will supposedly sell you a cracked version of
Toolbox for \$5000. Looking at how Odin is implemented I assume that cracked
version only works on older Model S/X cars since the Model 3 requires signed
certs from Tesla.

## Fused vs Unfused

There's a number of security measures based off of the Intel SOC's efuse. This
is a bit built into the processor that can only be written once. During
manufacturing, after provisioning the car the efuse is set to "fuse" the car and
prevent any unauthorized modification to the system.

Development cars are in an unfused state as to allow easy debugging. When the
car is unfused all of the firewall rules are disabled, a different set of SSH
keys are used and Odin authentication is disabled.

I've seen at least one "unfused" car computer listed on eBay. I'd be interested
to know how they obtained it. It would be interesting to buy one and see if you
could upload the standard car firmware to it and run it in an unfused/hackable
mode.

I've heard from a friend who used to work at Intel that the fuses are supposed
to be only be write once but it's sometimes possible to write them several times
and get them into a "broken" state where they'll return the wrong value. The
fuser does appear to write the same value 10 times so Tesla might have already
mitigated that.

## SSH

### Authentication

Model S used to have a SSH key on the CID/APE that could SSH into each other.
They also had password auth enabled so you could just use the default password
to get root access. This is no longer the case.

As I mentioned before, Tesla uses signed certifcates for everything and this
includes SSH. To SSH into the car you need an SSH certificate for that car
signed by the Tesla CA or one of their recovery keys. To ensure that one leaked
cert won't be reused elsewhere the keys include a "principle" for that specific
car.

```
PubkeyAuthentication yes
AuthorizedKeysFile /etc/ssh/authorized_keys_prod

# Support SSH certificate-based authentication.  Certificates must be signed
# by the TrustedUserCAKeys and must contain the authorized principal string
# that is returned by AuthorizedPrincipalsCommand.
TrustedUserCAKeys /etc/ssh/ssh_ca_developers_prod.pub
AuthorizedPrincipalsCommand /sbin/authorized_principal
AuthorizedPrincipalsCommandUser root
```

There's a few backup keys that can be used to SSH in but the key lengths seem
suitably long and presumably in cold storage somewhere as a last resort if all
of their CA infrastructure explodes.

#### /sbin/authorized_principle

This script parses the Hermes certificate to fetch the common name for the car.
It ensures that the SSH cert used has the principle `tesla:motors:vehicle:$CN`
so certs can't be reused from one car to another.

If there's no Hermes cert it falls back to `tesla:motors:vehicle:$VIN`.

If there's no VIN it requires `tesla:motors:vehicle:unprovisioned`. Presumably
these last two are used during development or as a last resort during
manufacturing.

### Protocols & Ciphers

As of version 2020.12.11.1 the car is using a version of OpenSSH and OpenSSL
from 21 April 2020. It doesn't appear there's any known vulnerabilities there.

Tesla has gotten a lot better at using up to date software. A number of the
previous exploits on the Model S were simple due to ancient software versions.

```
alarm@tesla ~> ssh -v 192.168.90.100
OpenSSH_8.2p1, OpenSSL 1.1.1g  21 Apr 2020
debug1: Reading configuration data /etc/ssh/ssh_config
debug1: Connecting to 192.168.90.100 [192.168.90.100] port 22.
debug1: Connection established.
debug1: identity file /home/alarm/.ssh/id_rsa type 0
debug1: identity file /home/alarm/.ssh/id_rsa-cert type -1
debug1: identity file /home/alarm/.ssh/id_dsa type -1
debug1: identity file /home/alarm/.ssh/id_dsa-cert type -1
debug1: identity file /home/alarm/.ssh/id_ecdsa type -1
debug1: identity file /home/alarm/.ssh/id_ecdsa-cert type -1
debug1: identity file /home/alarm/.ssh/id_ecdsa_sk type -1
debug1: identity file /home/alarm/.ssh/id_ecdsa_sk-cert type -1
debug1: identity file /home/alarm/.ssh/id_ed25519 type -1
debug1: identity file /home/alarm/.ssh/id_ed25519-cert type -1
debug1: identity file /home/alarm/.ssh/id_ed25519_sk type -1
debug1: identity file /home/alarm/.ssh/id_ed25519_sk-cert type -1
debug1: identity file /home/alarm/.ssh/id_xmss type -1
debug1: identity file /home/alarm/.ssh/id_xmss-cert type -1
debug1: Local version string SSH-2.0-OpenSSH_8.2
debug1: Remote protocol version 2.0, remote software version OpenSSH_7.9
debug1: match: OpenSSH_7.9 pat OpenSSH* compat 0x04000000
debug1: Authenticating to 192.168.90.100:22 as 'alarm'
debug1: SSH2_MSG_KEXINIT sent
debug1: SSH2_MSG_KEXINIT received
debug1: kex: algorithm: curve25519-sha256
debug1: kex: host key algorithm: ecdsa-sha2-nistp256
debug1: kex: server->client cipher: chacha20-poly1305@openssh.com MAC:
<implicit> compression: none
debug1: kex: client->server cipher: chacha20-poly1305@openssh.com MAC:
<implicit> compression: none
debug1: expecting SSH2_MSG_KEX_ECDH_REPLY
debug1: Server host key: ecdsa-sha2-nistp256
SHA256:g2LMKjlsobIXVimHcaP58JLahYrhyzoqJevYMq0LTuQ
debug1: Host '192.168.90.100' is known and matches the ECDSA host key.
debug1: Found key in /home/alarm/.ssh/known_hosts:4
debug1: rekey out after 134217728 blocks
debug1: SSH2_MSG_NEWKEYS sent
debug1: expecting SSH2_MSG_NEWKEYS
debug1: SSH2_MSG_NEWKEYS received
debug1: rekey in after 134217728 blocks
debug1: Will attempt key: /home/alarm/.ssh/id_rsa RSA
SHA256:C6m79wZNJKGfQxEHWp2MunUjssfKgYq4FNZQ6ncrPZ8
debug1: Will attempt key: /home/alarm/.ssh/id_dsa
debug1: Will attempt key: /home/alarm/.ssh/id_ecdsa
debug1: Will attempt key: /home/alarm/.ssh/id_ecdsa_sk
debug1: Will attempt key: /home/alarm/.ssh/id_ed25519
debug1: Will attempt key: /home/alarm/.ssh/id_ed25519_sk
debug1: Will attempt key: /home/alarm/.ssh/id_xmss
debug1: SSH2_MSG_EXT_INFO received
debug1: kex_input_ext_info:
server-sig-algs=<ssh-ed25519,ssh-rsa,rsa-sha2-256,rsa-sha2-512,ssh-dss,ecdsa-sha2-nistp256,ecdsa-sha2-nistp384,ecdsa-sha2-nistp521>
debug1: SSH2_MSG_SERVICE_ACCEPT received
debug1: Authentications that can continue: publickey
debug1: Next authentication method: publickey
debug1: Offering public key: /home/alarm/.ssh/id_rsa RSA
SHA256:C6m79wZNJKGfQxEHWp2MunUjssfKgYq4FNZQ6ncrPZ8
debug1: Authentications that can continue: publickey
debug1: Trying private key: /home/alarm/.ssh/id_dsa
debug1: Trying private key: /home/alarm/.ssh/id_ecdsa
debug1: Trying private key: /home/alarm/.ssh/id_ecdsa_sk
debug1: Trying private key: /home/alarm/.ssh/id_ed25519
debug1: Trying private key: /home/alarm/.ssh/id_ed25519_sk
debug1: Trying private key: /home/alarm/.ssh/id_xmss
debug1: No more authentication methods to try.
alarm@192.168.90.100: Permission denied (publickey).
```

## Disk / Firmware

### dm-verity

The root filesystem for the CID is mounted read-only to prevent any changes to
the running code. There's a few partitions for user data such as Spotify logins,
various configs, map data, etc but those are all mounted non-executable.

The root filesystem is also verified by the dm-verity kernel module which hashes
the filesystem on boot. This means it's nearly impossible to gain root access by
modifying the filesystem.

### Kernel / Secure Boot

I don't know a lot about the Intel SOC that's being used but it does support
some form of secure boot. I have no way of checking whether it's enabled but I
wouldn't be surprised if it was. If it's not enabled it should be possible to
modify the kernel to disable dm-verity and boot an unsigned image.

## Updater

All of the firmware blobs deployed to the various controllers around the car are
signed by Tesla. The updater checks the signature before updating to ensure
nothing weird is going on. This means we can't MITM the updater to install a
modified firmware.

If you can bypass the seceth rules you can talk directly to the updater and
manually give it an image to install but it has to be signed by Tesla. From one
of the Keen Security Lab papers they mentioned that Tesla has since added a
security measure to prevent the updater from installing an older version of the
software. This pretty much eliminates any hope of downgrading to a more
vulnerable version of the firmware.

## CAN Bus

There are a number of CAN bus connections in the car that can be accessed. CAN
bus is unencrypted so we can pull a fair amount of internal data from them.
There's been a number of projects to reverse engineer the message meanings.

There's a [couple of off the shelf harnesses/diagnostic
tools](http://store.evtv.me/proddetail.php?prod=TeslaModel3CANKit) you can use
to read them.

I reached out to the EVTV Motor Verks guys and they told me if the car
detects any injected/malicious CAN bus messages the entire car shuts down. I
haven't tried injecting messages on this so I'm not sure how extensive these
protections are.

## Services & AppArmor

Almost all of the various services in the car have AppArmor enabled and are
running as non-privileged users.

Spotify is running under the spotify user as a service. There doesn't seem to be
any way to deploy new sandboxed apps onto the system. I thought there would be
something similar to Androids APKs for something like Spotify but it's just a Qt
app.

## iptables / Firewall Rules

There's extensive iptables rules restricting all network communication. The
firewall rules are specified on a per user basis which I hadn't seen before.
This means things like the modem are restricted so they can only be accessed by the modem controller
and the updater.

There are forwarding rules so the Autopilot computer can talk directly to the
internet but only outgoing connections are allowed. It's a bit scary that the
computer driving the car has a direct internet connection.

```bash
# Setup Internet sharing for ape
iptables -A FORWARD -i eth0 -o eth0.2 -s $APE_LIST -d 192.168.20.0/24 -j DROP # disallow forwarding to modem device
for i in eth0.2 wlan0 ; do
    iptables -A FORWARD -i eth0 -o $i -s $APE_LIST -j ACCEPT
    iptables -A FORWARD -i $i -o eth0 -d $APE_LIST -m state --state RELATED,ESTABLISHED -j ACCEPT
    iptables -t nat -A POSTROUTING -o $i -j MASQUERADE -s $APE_LIST
done
echo 1 > /proc/sys/net/ipv4/ip_forward
```


## Escalator

There's a service running on the car called `escalator`. This is a service that
allows specific requests, from specific processes/users to run as root. On the
Model S there was just a hardcoded root password that processes could call, but
now all elevated permissions run through a single point.

If you manage to get a shell on the car, this would be a good place to look for
vulnerabilities to get root.

## Internal Car APIs

There's a number of internal car APIs accessible by unauthenticated HTTP. The
firewall rules mostly block these from being accessed externally as well as by
processes that aren't supposed to.

I was able to access some of these and I'll make a follow up post about some of
the things I found. :)
