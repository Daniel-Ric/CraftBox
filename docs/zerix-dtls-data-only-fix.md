# Zerix DTLS data-only compatibility — 2026-11-06

Session `session-20261106T063727Z-51272-8b356988bacf` reaches ICE and starts DTLS for `https://zerix.ml:19132`, with remote DTLS role `client`, then fails. The source-picker connection test reports `DTLS Handshake completed and no SRTP Protection Profile was chosen`. HTTP status success does not establish the game connection.

Pion WebRTC v4.2.26 unconditionally rejected the lack of an SRTP profile in `DTLSTransport.completeStart`, even for SCTP-only connections. CraftBox now uses the locally patched version described in [CRAFTBOX_PATCH.md](../third_party/webrtc/CRAFTBOX_PATCH.md). No server-address exception or cipher/fingerprint bypass is used.

Verification:

- A real UDP DTLS test without SRTP failed with the original completion code and passes with the patch.
- The same test exchanges encrypted application data, establishes SCTP, and echoes a DataChannel message end to end.
- Media requested before completion or afterward still fails without SRTP.
- Pion DTLS, invalid-fingerprint, profile-mapping and role tests pass.
- The full NetherNet, discovery and endpoint suites, CraftBox proxy suite, and Gophertunnel NetherNet/login/spawn tests pass.

This verifies the reported transport failure locally. A live Zerix join is still unverified because the execution environment blocks external network connections.

Build: `1.64.0-zerix-dtls-fix-dev`, output `builds/Craft-Box.exe`. The earlier player-count, Structure Studio and other application changes remain included.
