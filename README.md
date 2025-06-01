# Fade

Experimenting with [Wish](https://github.com/charmbracelet/wish) (part of [Charm](https://charm.sh/)).

We can create a very simple video game using the `▀` as two pixels and coloring the foreground and background based on the wanted color. Animation demo:

![Pokemon GB](https://vhs.charm.sh/vhs-7we4N7EPmQqJZTKMDbsHdD.gif)

I'm wondering if this could be something people would be interested in a pixelated multiplayer game that can be played over SSH.

Questions:
- How to create "rooms" based on demand and periodically merge them?
- Is it possible to play audio over the same SSH connection or does it need another connection?
- Should I allow concurrent connections or should I limit the access based on public key?

## Sound

I researched a bit about how to add sound. It is tricky, because it has to be supported by SSH and available in the underlying ssh library. After a bit of research and trial-and-error I think I found a way to use stderr as an audio channel.

When `sound on` is used as a command, the program will send notes and duration through stderr in the format of `<note> <duration> \n`. Example: `G4 0.3 \n`. The received data can be played using the `play` command of [SoX - Sound eXchange](https://sourceforge.net/projects/sox/).

Example script for enabling sound:

```
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

ssh -o StrictHostKeychecking=no -t fade.nergal.xyz sound on 2> /tmp/audio_stream
```
