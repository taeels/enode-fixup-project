#!/usr/bin/python3
"""Install inside the existing isolated VM. Credentials arrive ONLY on stdin.

Input: {email, password, team_id, signing_key}. The signing key is derived by the
Mediator operator, not the model. Source files must be copied to /opt/enode/gallery
before running this installer. Does not enable the public route or restart enode.
"""
import grp
import json
import os
from pathlib import Path
import pwd
import subprocess
import sys


def main():
    if os.getuid() != 0:
        raise SystemExit('Run inside the dedicated VM as root')
    config = json.load(sys.stdin)
    if set(config) != {'email', 'password', 'team_id', 'signing_key'} or not all(isinstance(v, str) and v for v in config.values()):
        raise SystemExit('Invalid broker configuration')
    if len(config['signing_key']) != 64:
        raise SystemExit('Use the purpose-derived SHA256 signing key')
    source = Path('/opt/enode/gallery')
    for name in ('common.py', 'broker.py', 'mcp.py', 'worker.py', 'enode-gallery.service'):
        p = source / name
        if not p.is_file() or p.is_symlink():
            raise SystemExit('Missing source file')
        os.chown(p, 0, 0); p.chmod(0o644)
    source.chmod(0o755); os.chown(source, 0, 0)
    try:
        publisher = pwd.getpwnam('enode-gallery')
    except KeyError:
        subprocess.run(['useradd', '--system', '--user-group', '--no-create-home', '--home-dir', '/var/lib/enode-gallery', '--shell', '/usr/sbin/nologin', 'enode-gallery'], check=True)
        publisher = pwd.getpwnam('enode-gallery')
    grp.getgrnam('enode-worker')
    p = Path('/etc/enode-gallery.json')
    fd = os.open(str(p), os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'w') as f:
        os.fchown(f.fileno(), 0, publisher.pw_gid)
        os.fchmod(f.fileno(), 0o640)
        json.dump(config, f)
    config.clear()
    # Extend only the existing worker egress table; do not replace other rules.
    nft = Path('/etc/nftables.conf')
    content = nft.read_text()
    marker = '    meta skuid 1001 reject'
    if marker not in content:
        raise SystemExit('Expected VM firewall boundary not found')
    rule = f'    meta skuid {publisher.pw_uid} oifname "lo" ip daddr 127.0.0.1 tcp dport 3128 accept\n    meta skuid {publisher.pw_uid} reject'
    if rule not in content:
        updated = content.replace(marker, marker + '\n' + rule)
        check = Path('/run/enode-gallery-firewall.nft'); check.write_text(updated)
        subprocess.run(['nft', '--check', '--file', str(check)], check=True)
        nft.write_text(updated)
        subprocess.run(['nft', '--file', str(nft)], check=True)
        check.unlink()
    unit = Path('/etc/systemd/system/enode-gallery.service')
    unit.write_text((source / unit.name).read_text()); unit.chmod(0o644)
    subprocess.run(['systemctl', 'daemon-reload'], check=True)
    subprocess.run(['systemctl', 'enable', '--now', 'enode-gallery'], check=True)
    print('Gallery broker installed; public route and node advertisement unchanged.')


if __name__ == '__main__':
    main()
