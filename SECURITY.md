# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take security seriously. If you discover a vulnerability, please report it responsibly.

1.  **Do NOT open a public issue.**
2.  Email us directly at [nishalamv@gmail.com](mailto:nishalamv@gmail.com) (or open a Draft Security Advisory on GitHub).
3.  Include a detailed description of the vulnerability and steps to reproduce it.

We aim to acknowledge reports within 48 hours and will provide a timeline for a fix.

## Encryption Standards

-   **Database**: SQLCipher (AES-256 CBC)
-   **Key Derivation**: Argon2id (Time=1, Memory=64MB, Threads=4)
-   **Transport**: libp2p (TLS 1.3 / Noise)

Thank you for helping keep ShellChat secure!
