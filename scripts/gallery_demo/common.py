"""Fixed-origin gallery access. No caller supplies an HTTP method or URL."""
import http.cookiejar
import json
import re
from urllib.request import Request, build_opener, ProxyHandler, HTTPCookieProcessor, HTTPRedirectHandler

ORIGIN = 'https://main.d3gkmtkue9o7ly.amplifyapp.com'
PROXY = 'http://127.0.0.1:3128'
PROJECT = re.compile(r'[a-zA-Z0-9_-]{1,64}\Z')
MAX_RESPONSE = 512 * 1024


def project_id(value):
    if not isinstance(value, str) or not PROJECT.fullmatch(value):
        raise ValueError('invalid project')
    return value


def comment_body(value):
    if not isinstance(value, str) or not value.strip() or len(value) > 500 or any(ord(c) < 32 and c not in '\n\t' for c in value):
        raise ValueError('invalid comment')
    return value


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        raise ValueError('redirect blocked')


class Gallery:
    def __init__(self):
        self.jar = http.cookiejar.CookieJar()
        self.opener = build_opener(ProxyHandler({'https': PROXY}), HTTPCookieProcessor(self.jar), NoRedirect())

    def _call(self, path, body=None):
        req = Request(ORIGIN + path, data=None if body is None else json.dumps(body).encode(),
                      headers={'Content-Type': 'application/json', 'Origin': ORIGIN,
                               'User-Agent': 'enode-gallery-demo/1', 'Referer': ORIGIN + '/login'},
                      method='GET' if body is None else 'POST')
        with self.opener.open(req, timeout=20) as response:
            data = response.read(MAX_RESPONSE + 1)
            if len(data) > MAX_RESPONSE:
                raise ValueError('response too large')
            return json.loads(data)

    def project(self, selected):
        selected = project_id(selected)
        p = self._call('/api/projects/' + selected)['project']
        if p.get('id') != selected:
            raise ValueError('project mismatch')
        # Links/media and all unrelated fields never become tool instructions.
        return {k: str(p.get(k) or '')[:8000 if k == 'description' else 300]
                for k in ('id', 'title', 'teamName', 'description')}

    def projects(self):
        rows = self._call('/api/projects')['projects']
        if not isinstance(rows, list) or len(rows) > 200:
            raise ValueError('invalid catalog')
        return [{'id': project_id(p['id']), 'title': str(p.get('title') or '')[:300],
                 'teamName': str(p.get('teamName') or '')[:300]} for p in rows]

    def login(self, credentials):
        self._call('/api/auth/login', {k: credentials[k] for k in ('email', 'password')})
        team = self._call('/api/auth/me').get('team') or {}
        if credentials['team_id'] not in [team.get(k) for k in ('id', 'teamId', 'team_id')]:
            raise ValueError('account mismatch')

    def comments(self, selected):
        return self._call('/api/projects/' + project_id(selected) + '/comments')

    def project_comments(self, selected, offset=0, limit=40):
        selected = project_id(selected)
        if type(offset) is not int or not 0 <= offset <= 2000 or type(limit) is not int or not 1 <= limit <= 50:
            raise ValueError('invalid comment page')
        data = self.comments(selected)
        rows = data.get('comments')
        if not isinstance(rows, list) or len(rows) > 2000 or any(not isinstance(c, dict) for c in rows):
            raise ValueError('invalid comments')
        rows = [c for c in rows if not c.get('deleted')]
        comments = []
        for row in rows[offset:offset + limit]:
            # Public discussion evidence only: never relay account/session fields.
            comment = {key: str(row.get(key) or '')[:4000 if key == 'body' else 300]
                       for key in ('id', 'teamName', 'body', 'parentId', 'createdAt')}
            comment['body_truncated'] = len(str(row.get('body') or '')) > 4000
            comments.append(comment)
        return {'project_id': selected, 'comments': comments, 'total': len(rows), 'offset': offset,
                'next_offset': offset + limit if offset + limit < len(rows) else None}

    def post(self, selected, body):
        return self._call('/api/projects/' + project_id(selected) + '/comments', {'body': comment_body(body)})


def child_setup(parent_pid):
    """Linux VM: a model/MCP child must die when its enode worker is killed."""
    def setup():
        import ctypes
        import os
        import resource
        import signal
        import sys
        resource.setrlimit(resource.RLIMIT_FSIZE, (8 * 1024 * 1024, 8 * 1024 * 1024))
        if sys.platform == 'linux':
            if ctypes.CDLL(None).prctl(1, signal.SIGKILL, 0, 0, 0) != 0:
                os._exit(1)
            if os.getppid() != parent_pid:
                os._exit(1)
    return setup
