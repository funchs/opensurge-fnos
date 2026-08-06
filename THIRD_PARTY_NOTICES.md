# Third-Party Notices

OpenSurge is licensed under `GPL-3.0-only`. The independent programs and
libraries listed below retain their upstream licenses.

This fork distributes the gateway as a **Docker image for fnOS (Debian 12,
`linux/amd64`)** rather than a macOS installer package. The image is built by
[`Dockerfile`](Dockerfile); the notices below describe what that image
contains. Upstream's macOS installer placed these notices under
`/Library/Application Support/OpenSurge/share/licenses/` — that path does not
exist in this distribution.

## mihomo

- Version: `1.19.27`
- License: `GPL-3.0-only`
- Distributed form: unmodified upstream `mihomo-linux-amd64-compatible-v1.19.27.gz`,
  downloaded and checksum-verified at image build time, decompressed to
  `/usr/local/bin/mihomo`
- Archive SHA-256:
  `36850c946615f5c712946b62dbbbd06f6941d6d8a7543b315198bcb24ada3ea9`
- Upstream: <https://github.com/MetaCubeX/mihomo>
- Corresponding source:
  <https://github.com/MetaCubeX/mihomo/tree/5184081ac327394d9e15fa5d5f9f4a61e723fd94>
- License text: [`LICENSE`](LICENSE)

## dnsmasq

- Version: `2.90-4~deb12u2` (Dnsmasq 2.90)
- License: `GPL-2.0-only OR GPL-3.0-only`, at the recipient's option
- Distributed form: the unmodified Debian 12 `dnsmasq-base` binary package,
  installed with `apt-get` into the runtime image. Upstream's macOS build
  compiled dnsmasq 2.93 from source; this fork does not.
- Corresponding source: `apt-get source dnsmasq` on Debian 12, or
  <https://sources.debian.org/src/dnsmasq/2.90-4~deb12u2/>
- License texts: [`third_party/licenses/dnsmasq-COPYING`](third_party/licenses/dnsmasq-COPYING)
  and [`LICENSE`](LICENSE)

## Debian 12 base image and system packages

The runtime image derives from `debian:12-slim` and installs the following
Debian binary packages unmodified. Each retains its own license as recorded in
`/usr/share/doc/<package>/copyright` inside the image; corresponding source for
any of them is available via `apt-get source <package>` on Debian 12 or from
<https://sources.debian.org/>.

| Package | Version | License |
|---|---|---|
| `nftables` | `1.0.6-2+deb12u2` | `GPL-2.0-only` |
| `iproute2` | `6.1.0-3` | `GPL-2.0-or-later` |
| `iputils-ping` | `3:20221126-1+deb12u1` | `BSD-3-Clause` and `GPL-2.0-or-later` |
| `procps` | `2:4.0.2-3` | `GPL-2.0-or-later` and `LGPL-2.0-or-later` |
| `ca-certificates` | Debian 12 | `MPL-2.0` and `GPL-2.0-or-later` |

## gopkg.in/yaml.v3

- Version: `3.0.1`
- License: MIT and Apache-2.0, according to the upstream per-file notice
- Upstream: <https://github.com/go-yaml/yaml/tree/v3.0.1>
- License texts: [`third_party/licenses/yaml-v3-LICENSE`](third_party/licenses/yaml-v3-LICENSE)
  and [`third_party/licenses/Apache-2.0.txt`](third_party/licenses/Apache-2.0.txt)

## React, React DOM, and scheduler

- Versions: React `19.2.7`, React DOM `19.2.7`, scheduler `0.27.0`
- License: MIT
- Upstream: <https://github.com/facebook/react>
- License text: [`third_party/licenses/react-MIT.txt`](third_party/licenses/react-MIT.txt)
