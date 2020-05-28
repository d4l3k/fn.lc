#!/usr/bin/bash

set -xe

xhost + | exit 0

bind () {
  OUT="$2"
  case `cat /proc/mounts | grep "$OUT" >/dev/null; echo $?` in
    0)
      echo "already mounted $OUT"
      ;;
    1)
      echo "binding $OUT"
      mount --rbind $1 $OUT
      ;;
    *)
      echo "failed to mount $OUT"
      exit 1
      ;;
  esac
}

map () {
  mkdir -p "squashfs-root$1"
  bind $1 "squashfs-root$1"
}

map-file () {
  if [ -z "$2" ]
  then
    OUT="squashfs-root$1"
  else
    OUT="squashfs-root$2"
  fi
  mkdir -p "$(dirname "$OUT")"
  touch "$OUT"
  bind $1 $OUT
}

map /proc
map /dev/dri
map-file /dev/urandom
map-file /dev/zero
map-file /dev/log
map-file /dev/random
map-file /dev/uinput
map /dev/input

chmod +777 squashfs-root/dev/uinput
chmod +777 squashfs-root/dev/input/*
chmod +777 squashfs-root/dev/dri/card0

#ip addr add 192.168.90.100/24 dev lo || true

export PATH=/usr/local/bin:/bin:/sbin:/usr/sbin:/usr/bin:/usr/tesla/UI/bin:/usr/tesla/bin

chroot squashfs-root $*
