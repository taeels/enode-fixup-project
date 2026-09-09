"""Credential-owning, single-use publisher. Run as a separate VM user."""
import base64
import grp
import hashlib
import hmac
import json
import os
from pathlib import Path
import re
import socketserver
import sqlite3
import sys
import time

from common import Gallery, ORIGIN, comment_body, project_id

PURPOSE = 'enode-gallery-publish-v1'


def verify(ticket, key, now=None):
    if not isinstance(ticket, str) or len(ticket) > 12000:
        raise ValueError('invalid ticket')
    payload, signature = ticket.split('.')
    expected = hmac.new(key.encode(), payload.encode(), hashlib.sha256).hexdigest()
    if not hmac.compare_digest(signature, expected):
        raise ValueError('invalid signature')
    job = json.loads(base64.urlsafe_b64decode(payload + '=' * (-len(payload) % 4)))
    if set(job) != {'purpose', 'job_id', 'project_id', 'body', 'expires'} or job['purpose'] != PURPOSE:
        raise ValueError('invalid scope')
    now = time.time() if now is None else now
    if type(job['expires']) is not int or not now < job['expires'] <= now + 1800:
        raise ValueError('expired ticket')
    if not re.fullmatch(r'gallery-post-[0-9a-f]{64}', job['job_id']):
        raise ValueError('invalid job')
    project_id(job['project_id'])
    comment_body(job['body'])
    return job


class Publisher:
    def __init__(self, config, database, gallery=None):
        self.config = config
        self.db = sqlite3.connect(database)
        self.db.execute('CREATE TABLE IF NOT EXISTS attempts (id TEXT PRIMARY KEY, intent TEXT NOT NULL, result TEXT NOT NULL)')
        self.gallery = gallery or Gallery()

    def publish(self, ticket):
        job = verify(ticket, self.config['signing_key'])
        identity = json.dumps([job['project_id'], job['body']], ensure_ascii=False)
        existing = self.db.execute('SELECT intent, result FROM attempts WHERE id=?', (job['job_id'],)).fetchone()
        if existing:
            if existing[0] != identity:
                raise ValueError('confirmation changed')
            return json.loads(existing[1])
        # Authentication and read failures happen before any write attempt.
        self.gallery.project(job['project_id'])
        self.gallery.login(self.config)
        before = self.gallery.comments(job['project_id'])
        if before.get('canWrite') is not True or before.get('enabled') is not True:
            raise ValueError('comments unavailable')
        previous_ids = {c.get('id') for c in before.get('comments', [])}
        pending = {'outcome': 'error', 'message': '게시 응답이 미확인입니다. 중복 방지를 위해 자동 재게시하지 않습니다.'}
        # Durable commit BEFORE POST. A crash/timeout can never cause another POST.
        with self.db:
            self.db.execute('INSERT INTO attempts VALUES (?, ?, ?)', (job['job_id'], identity, json.dumps(pending)))
        try:
            self.gallery.post(job['project_id'], job['body'])
        except Exception:
            pass  # Ambiguous outcome: verify using GET only.
        try:
            after = self.gallery.comments(job['project_id'])
            matches = [c for c in after.get('comments', []) if c.get('id') not in previous_ids
                       and c.get('body') == job['body'] and not c.get('deleted')
                       and c.get('mine') is True]
            if len(matches) == 1:
                pending = {'outcome': 'posted', 'message': '확인한 댓글이 갤러리에 게시되었습니다.',
                           'project_id': job['project_id'], 'body': job['body'],
                           'comment_id': matches[0]['id'], 'url': ORIGIN + '/projects/' + job['project_id']}
        except Exception:
            pass
        with self.db:
            self.db.execute('UPDATE attempts SET result=? WHERE id=?', (json.dumps(pending), job['job_id']))
        return pending


class Handler(socketserver.StreamRequestHandler):
    def handle(self):
        self.connection.settimeout(70)
        try:
            line = self.rfile.readline(16001)
            if len(line) > 16000:
                raise ValueError('request too large')
            data = json.loads(line)
            if set(data) != {'ticket'}:
                raise ValueError('invalid request')
            result = self.server.publisher.publish(data['ticket'])
        except Exception:
            result = {'outcome': 'error', 'message': '게시 권한 또는 갤러리 연결을 확인하지 못했습니다.'}
        self.wfile.write(json.dumps(result, ensure_ascii=False).encode() + b'\n')


def main():
    config = json.loads(Path('/etc/enode-gallery.json').read_text())
    socket_path = Path('/run/enode-gallery/broker.sock')
    socket_path.unlink(missing_ok=True)
    with socketserver.UnixStreamServer(str(socket_path), Handler) as server:
        os.chown(socket_path, -1, grp.getgrnam('enode-worker').gr_gid)
        os.chmod(socket_path, 0o660)
        server.publisher = Publisher(config, '/var/lib/enode-gallery/attempts.sqlite3')
        server.serve_forever()


if __name__ == '__main__':
    main()
