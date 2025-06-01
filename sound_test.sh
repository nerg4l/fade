#!/usr/bin/env bash

if [ ! -e /tmp/audio_stream ]; then
  mkfifo /tmp/audio_stream
  trap 'rm -f /tmp/audio_stream' EXIT
fi

while read note dur rest < /tmp/audio_stream; do
  if [ ${#note} == 2 ]; then
    play -n synth "$dur" triangle "$note" gain -30 > /dev/null 2> /dev/null &
  fi
done &

ssh -o StrictHostKeychecking=no -p 8122 -t localhost sound on 2> /tmp/audio_stream
