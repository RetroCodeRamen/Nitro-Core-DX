# Galaxy Force Data Schemas

## Save State Format (Phase B)

```
Offset  Size  Field
0x00    2     magic (0x4746 = "GF")
0x02    1     version
0x03    1     current_mission (0-9)
0x04    2     score (u16)
0x06    1     player_hp
0x07    1     weapon_family (0=Vulcan, 1=Laser, 2=Plasma)
0x08    1     sub_weapon (0=Missile, 1=Bomb, 2=Shield, 3=Drone)
0x09    1     defense_module (0=Armor, 1=Barrier)
0x0A    1     weapon_level (0-4)
0x0B    1     relationship_flags (bit field)
0x0C    1     story_flags (bit field for branching)
0x0D    1     missions_completed (bit field)
0x0E    2     reserved
```

## Level Segment Definition (Phase B/C)

Levels are ordered lists of segments. Each segment type has its own data layout:

### VerticalSegment
```
type: 0
duration: u16 (frames)
scroll_speed: u8
wave_count: u8
waves: WaveDefinition[]
```

### BossSegment
```
type: 1
boss_id: u8
boss_hp: u16
phase_count: u8
phase_thresholds: u8[]
```

### DialogSegment
```
type: 2
dialog_id: u8
choice_count: u8
```

### TunnelSegment (Matrix Mode)
```
type: 3
duration: u16
rotation_speed: i8
tunnel_radius: u8
```

### HorizontalSegment
```
type: 4
duration: u16
scroll_speed: u8
wave_count: u8
```

## Wave Definition
```
spawn_time: u16 (frame offset from segment start)
enemy_type: u8
count: u8
formation: u8 (0=line, 1=V, 2=circle, 3=random)
x_start: u16
y_start: u16
speed: u8
hp: u8
```

## Dialog Entry
```
speaker_id: u8 (0=pilot, 1=wing, 2=command, 3=dominion)
portrait_id: u8
text_length: u8
text: bytes (ASCII)
choice_count: u8
choices: ChoiceEntry[]
```

## Upgrade Definition
```
id: u8
type: u8 (0=weapon, 1=sub, 2=defense, 3=passive)
stat_modifier: i8
cost: u16
prerequisite: u8
```
