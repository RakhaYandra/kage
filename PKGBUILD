# Maintainer: RakhaYandra
pkgname=kage
pkgver=0.2.0
pkgrel=1
pkgdesc="Configuration state manager: snapshot, diff, restore Linux configs"
arch=('x86_64' 'aarch64')
url="https://github.com/RakhaYandra/kage"
license=('MIT')
depends=('git' 'glibc')
optdepends=('systemd: scheduled snapshots via user timer')
makedepends=('go>=1.27')
source=("$pkgname-$pkgver.tar.gz::$url/archive/v$pkgver.tar.gz")
sha256sums=('SKIP')

build() {
  cd "$pkgname-$pkgver"
  export CGO_ENABLED=0
  go build -trimpath \
    -ldflags "-s -w -X github.com/RakhaYandra/kage/internal/cli.version=$pkgver" \
    -o bin/kage ./cmd/kage
}

package() {
  cd "$pkgname-$pkgver"
  install -Dm755 bin/kage "$pkgdir/usr/bin/kage"
  install -Dm644 systemd/kage-snapshot.service "$pkgdir/usr/lib/systemd/user/kage-snapshot.service"
  install -Dm644 systemd/kage-snapshot.timer "$pkgdir/usr/lib/systemd/user/kage-snapshot.timer"
  install -Dm644 README.md "$pkgdir/usr/share/doc/$pkgname/README.md"
}
