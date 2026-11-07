# CraftBox data-only DTLS patch

Base: Pion WebRTC `v4.2.26`, copied from the pinned Go module. The MIT license and upstream source notices are retained. The root `go.mod` replaces `github.com/pion/webrtc/v4` with this directory.

The production patch is limited to `dtlstransport.go`:

- `completeStart` accepts a completed DTLS handshake without a negotiated SRTP profile when no RTP media was requested. Unknown advertised profiles and missing profiles for requested media remain errors.
- `srtpCanStart` remains false when no profile was negotiated.
- `wantSRTP` rejects a later request for media when the connected DTLS session has no negotiated profile.

NetherNet transports game data over SCTP channels. Peers without RTP media can omit `use_srtp`. DTLS encryption, certificate validation, fingerprint matching, cipher suites and replay protection are unchanged. This does not suppress arbitrary DTLS errors or retry an unauthenticated connection.

`dtls_data_only_test.go` establishes a real local UDP DTLS pair without SRTP, exercises both DTLS completion roles and a full SCTP DataChannel echo, and checks that immediate/later RTP media requests still fail. Upstream invalid-fingerprint and profile-mapping tests are also run. The regular NetherNet and Gophertunnel transport tests cover existing connections that negotiate SRTP.

When upgrading Pion, re-evaluate this patch and retain the regression test until the data-only case is handled upstream.
