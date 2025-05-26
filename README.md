# Fade

Experimenting with [Wish](https://github.com/charmbracelet/wish) (part of [Charm](https://charm.sh/)).

We can create a very simple video game using the `▀` as two pixels and coloring the foreground and background based on the wanted color. Animation demo:

![Pokemon GB](https://vhs.charm.sh/vhs-7we4N7EPmQqJZTKMDbsHdD.gif)

I'm wondering if this could be something people would be interested in a pixelated multiplayer game that can be played over SSH.

Questions:
- How to create "rooms" based on demand and periodically merge them?
- Is it possible to play audio over the same SSH connection or does it need another connection?
- Should I allow concurrent connections or should I limit the access based on public key?
