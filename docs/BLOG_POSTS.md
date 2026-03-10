# Blog Posts

## March 9, 2026 - YM2608 Runtime Bring-Up Update

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
