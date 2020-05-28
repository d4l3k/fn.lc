#!/bin/bash

set -xe

# var for session name (to avoid repeated occurences)
sn=tesla

tmux new-session -s "$sn" -d "./chroot.sh bash"

run () {
tmux new-window -t "$sn" -n "$1" "$2"
}


run escalator "./chroot.sh /usr/bin/escalator"
run ecall_client "./chroot.sh /usr/tesla/UI/RunQtCar ecall_client"
run sim_service "./chroot.sh /usr/tesla/UI/RunQtCar sim_service"
run carserver "./chroot.sh /usr/tesla/UI/RunQtCar carserver"
run vehicle "./chroot.sh /usr/tesla/UI/RunQtCar vehicle"

tmux -2 attach-session -t "$sn"
