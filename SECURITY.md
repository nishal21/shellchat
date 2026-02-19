# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security seriously. If you discover a vulnerability, please report it responsibly.

1. **You can open a public issue or Send Mail.**
2. Email us directly at [nishalamv@gmail.com](mailto:nishalamv@gmail.com) (or open a Draft Security Advisory on GitHub).
3. Include a detailed description of the vulnerability, affected versions, and steps to reproduce it (ideally with a minimal repro).

We aim to acknowledge reports within 48 hours and will provide a timeline for a fix.

## Encryption Standards

ShellChat's goals are confidentiality and minimal metadata. The high-level stack is:

- Transport: libp2p (Noise protocol for direct peer streams). TLS 1.3 is used where libp2p endpoints or bridges employ it; Noise is the primary end-to-end stream encryption for P2P connections.
- At-Rest (Database and Local Storage): Application-level encryption is used for stored data. We use XChaCha20-Poly1305 (AEAD) for encrypting message payloads and sensitive blobs before writing them to disk. This ensures authenticated encryption with a nonce size that reduces the risk of accidental nonce reuse across sessions.
- Storage Engine: Pure-Go SQLite (modernc.org/sqlite) is used for cross-platform compatibility. Encryption is performed at the application layer (see above) rather than relying on a storage engine-provided encryption extension.
- Key Derivation: Argon2id is used to derive encryption keys from user passphrases. Recommended conservative parameters (may be tuned per release/platform):
  - Time (iterations): 3
  - Memory: 64 MB
  - Parallelism: 4
  These parameters balance brute-force resistance with reasonable cross-platform performance; they may be increased in future releases as platforms allow.
- Secrets Handling: Keys are held in-process and never written to disk unencrypted. When available, OS-provided secure keystores are used to protect persistent secrets (e.g., Keychain on macOS/iOS, Keystore on Android, Windows DPAPI).

## Threat Model & Limitations

- ShellChat aims for end-to-end confidentiality between peers. It does not attempt to prevent endpoint compromise or protect against malware on a user’s device.
- Some metadata (e.g., connection timing, IP addresses visible to peers or NAT/relay services) may be observable by network intermediaries or peers. The project minimizes central metadata retention by design (no central servers).
- Users seeking high anonymity should combine ShellChat with appropriate network-level privacy tools (VPNs, Tor where applicable), understanding trade-offs with P2P connectivity and performance.

## Disclosure & Response

- We will not publicly disclose a reported CVE or advisory until a fix or reasonable mitigation is available, except as required by law.
- Coordinated disclosure timelines will be agreed upon with reporters when practical.
- If you need an encryption key, signing key, or other sensitive artifacts during triage we will provide secure means to exchange secrets (PGP/secure channel) — do not transmit secrets over public or unencrypted channels.

## Contact & Follow-up

- Contact: [nishalamv@gmail.com](mailto:nishalamv@gmail.com)
- For urgent issues, please include “URGENT: ShellChat Security” in the subject line and provide a secure callback method.
- We aim to acknowledge all reports within 48 hours and to provide status updates until the issue is resolved.

Thank you for helping keep ShellChat secure!
