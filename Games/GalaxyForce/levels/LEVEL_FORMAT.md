# Level Format

Levels are defined as ordered sequences of segments. Each level file will eventually be
a CoreLX `gamedata` asset. For Phase A, waves are generated procedurally by timer.

## Level 1: Halo Line Defense

```
segments:
  - type: VerticalSegment
    duration: 900
    scroll_speed: 1
    waves:
      - { time: 60,  type: scout,   count: 3, formation: line }
      - { time: 120, type: scout,   count: 4, formation: V }
      - { time: 240, type: fighter, count: 3, formation: line }
      - { time: 360, type: scout,   count: 5, formation: random }
      - { time: 480, type: fighter, count: 4, formation: V }
      - { time: 600, type: heavy,   count: 2, formation: line }
      - { time: 720, type: fighter, count: 6, formation: circle }
      - { time: 840, type: heavy,   count: 3, formation: V }
  - type: BossSegment
    boss: seraph_vanguard
    hp: 30
    phases: [30, 20, 10]
  - type: DialogSegment
    dialog: post_level1
```

## Level 2: Matrix Tunnel (Phase B)

```
segments:
  - type: DialogSegment
    dialog: pre_tunnel
  - type: TunnelSegment
    duration: 600
    rotation_speed: 2
    tunnel_radius: 80
  - type: DialogSegment
    dialog: post_tunnel
```
