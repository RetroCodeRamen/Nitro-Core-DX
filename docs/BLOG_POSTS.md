# Blog Posts

## March 6, 2026 - Weekly Update (Since Feb 27)

This week was a big one for Nitro-Core-DX. We pushed hard on Dev Kit quality, Sprite Lab usability, editor stability, and project planning discipline. We also made a major audio direction call for V1.

### What We Shipped and Improved

- Released and continued iterating after `v0.1.7`, centered on Dev Kit workflow improvements and Sprite Lab quality-of-life upgrades.
- Advanced the IDE structure toward a more professional SDK feel with grouped workflows and cleaner interaction paths.
- Improved Sprite Lab capabilities and ergonomics:
- better color workflow and palette editing
- larger practical sprite workflows
- transparent index handling and visual clarity improvements
- wrapped sprite shift controls (up/down/left/right)
- improved preview behavior to keep correct proportions when resizing
- Continued Tilemap Lab and project asset pipeline work so tools feed compiler workflows more cleanly.
- Added and refined planning artifacts for V1 execution control (`V1_CHARTER`, `V1_ACCEPTANCE`, `V1_RISKS`).

### Biggest Challenges This Week

- The custom code editor stack hit multiple interaction regressions as features accumulated:
- cursor placement inconsistency
- selection behavior failing under real editing use
- input lag under frequent re-highlighting
- right-click crash paths
- Window behavior regressions appeared during UI refactors (native maximize temporarily disabled/greyed out), which hurt the "professional tool" feel.
- We had to stabilize quickly without losing forward momentum on syntax highlighting and Sprite Lab features.

### Design Changes We Made

- We moved away from the fragile hidden-widget editor pattern and started consolidating toward a native, single-ownership editor direction.
- We tightened Dev Kit windowing policy around OS-native behavior (maximize/minimize/restore) and documented it as a release-quality invariant.
- We shifted planning from loose feature growth to explicit gate-based execution to reduce drift and avoid accidental regressions.

### Major Plan Change: Audio Direction for V1

One major strategic change happened this week:

- V1 audio target is now **YM2608**.
- YM2151/OPM-lite is no longer the intended final V1 target.

Why the change: after a retro PC sound deep dive (and a YouTube rabbit hole), the intended sonic character became clearer, and YM2608 better matches the direction we actually want Nitro-Core-DX to sound like at release.

This is now reflected in planning docs as a controlled migration, not an immediate implementation spike.

### Execution Order Locked for V1

We also formalized sequence constraints so scope does not jump around:

1. Finish Sprite Lab polish/stability and Dev Kit hardening.
2. Complete required Tilemap flow.
3. Bring YM2608 chip behavior online and passing conformance tests.
4. Update CoreLX/APU integration for the YM2608 runtime path.
5. Then start Sound Studio implementation.

This ordering is now treated as release-blocking planning guidance, not optional advice.

### What Comes Next

Immediate next focus remains tool and workflow quality:

- Finish Sprite Lab polish and stability pass.
- Continue Dev Kit UX hardening and editor responsiveness improvements.
- Finalize tilemap workflow integration.
- Bring YM2608 conformance tests online, then land CoreLX/APU integration updates.
- Start Sound Studio only after those audio-runtime gates are solid.

Once those are in place, YM2608 work begins with a spec-first approach so we avoid destabilizing the rest of the pipeline.
