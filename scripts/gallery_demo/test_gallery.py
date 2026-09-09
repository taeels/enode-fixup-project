import base64
import hashlib
import hmac
import io
import json
from pathlib import Path
import tempfile
import time
import unittest
from unittest.mock import patch

import broker
import common
import mcp
import worker

KEY = 'test-signing-key'
JOB = {'purpose': broker.PURPOSE, 'job_id': 'gallery-post-' + 'a' * 64, 'project_id': 'project-1', 'body': '좋은 프로젝트네요!', 'expires': 0}


def ticket(**changes):
    job = dict(JOB, expires=int(time.time()) + 900, **changes)
    payload = base64.urlsafe_b64encode(json.dumps(job).encode()).decode().rstrip('=')
    return payload + '.' + hmac.new(KEY.encode(), payload.encode(), hashlib.sha256).hexdigest()


def comment_ticket(**changes):
    grant = dict(purpose=broker.COMMENT_PURPOSE, job_id=JOB['job_id'], expires=int(time.time()) + 900)
    grant.update(changes)
    payload = base64.urlsafe_b64encode(json.dumps(grant).encode()).decode().rstrip('=')
    return payload + '.' + hmac.new(KEY.encode(), payload.encode(), hashlib.sha256).hexdigest()


class FakeGallery:
    def __init__(self):
        self.posts = 0
        self.items = []
        self.timeout = False
        self.visible = True

    def project(self, selected):
        return {'id': selected, 'title': '테스트', 'teamName': '팀', 'description': '프로젝트 소개'}

    def projects(self):
        return [{'id': 'project-1', 'title': '테스트', 'teamName': '팀'}, {'id': 'project-2', 'title': '다른 프로젝트', 'teamName': '다른 팀'}]

    def login(self, config):
        pass

    def comments(self, selected):
        return {'enabled': True, 'canWrite': True, 'comments': list(self.items) if self.visible else []}

    def post(self, selected, body):
        self.posts += 1
        self.items.append({'id': str(self.posts), 'body': body, 'mine': True})
        if self.timeout:
            raise TimeoutError()
        return {'comment': self.items[-1]}


class GalleryTests(unittest.TestCase):
    def test_comment_grant_posts_once_and_cannot_switch_team(self):
        fake = FakeGallery()
        with tempfile.TemporaryDirectory() as directory:
            path = str(Path(directory) / 'state.sqlite')
            publisher = self.publisher(path, fake)
            signed = comment_ticket()
            first = publisher.publish(signed, 'project-1', JOB['body'])
            self.assertEqual(first['outcome'], 'posted')
            publisher.db.close()
            publisher = self.publisher(path, fake)
            self.assertEqual(publisher.publish(signed, 'project-1', JOB['body']), first)
            for selected, body in [('project-2', JOB['body']), ('project-1', '다른 본문')]:
                with self.assertRaises(ValueError):
                    publisher.publish(signed, selected, body)
            self.assertEqual(fake.posts, 1)
            publisher.db.close()

    def test_comment_grant_rejects_unknown_project_and_fixed_ticket_override(self):
        fake = FakeGallery(); publisher = self.publisher(':memory:', fake)
        for signed, selected, body in [(comment_ticket(), 'missing-project', JOB['body']), (comment_ticket(), 'https://example.com', JOB['body']),
                                       (comment_ticket(), 'project-1', 'x' * 501), (ticket(), 'project-1', 'override'),
                                       (comment_ticket(project_id='project-1'), 'project-1', JOB['body'])]:
            with self.assertRaises(ValueError):
                publisher.publish(signed, selected, body)
        self.assertEqual(fake.posts, 0)
        publisher.db.close()

    def test_discovery_mcp_requires_catalog_and_cannot_publish(self):
        tools = [('get_project', {'project_id': 'project-1'}), ('list_projects', {}),
                 ('get_project', {'project_id': 'project-1'}), ('get_project', {'project_id': 'https://example.com'}), ('post_comment', {})]
        requests = [{'jsonrpc': '2.0', 'id': i, 'method': 'tools/call', 'params': {'name': tool, 'arguments': args}} for i, (tool, args) in enumerate(tools)]
        fake = FakeGallery()
        with tempfile.TemporaryDirectory() as directory, patch.object(mcp, 'Gallery', return_value=fake):
            out = io.StringIO(); audit = str(Path(directory) / 'events')
            mcp.serve('discover', '', audit, io.StringIO(''.join(json.dumps(r) + '\n' for r in requests)), out)
            replies = [json.loads(line)['result'] for line in out.getvalue().splitlines()]
            self.assertTrue(replies[0]['isError']); self.assertIn('projects', json.loads(replies[1]['content'][0]['text']))
            self.assertEqual(json.loads(replies[2]['content'][0]['text'])['id'], 'project-1')
            self.assertTrue(replies[3]['isError']); self.assertTrue(replies[4]['isError']); self.assertEqual(fake.posts, 0)

    def test_comment_worker_posts_after_catalog_and_exact_project_read(self):
        job = {'operation': 'comment', 'prompt': '팀에 응원 댓글 달아줘', 'ticket': 'private-grant'}
        for decision, outcome, read_id, should_post in [('comment_intent', 'draft_ready', 'project-1', True),
                                                       ('draft_intent', 'draft_ready', 'project-1', False),
                                                       ('comment_intent', 'needs_project', None, False),
                                                       ('comment_intent', 'draft_ready', 'project-2', False)]:
            def model(prompt, policy, schema, directory, config=None):
                if config is None:
                    return {'outcome': decision, 'message': '요청을 확인했습니다.'}
                self.assertEqual(config['mode'], 'discover')
                self.assertNotIn('private-grant', policy)
                rows = [{'role': 'tool', 'tool': 'list_projects', 'text': '팀 목록 확인', 'ok': True}]
                if read_id:
                    rows.append({'role': 'tool', 'tool': 'get_project', 'project_id': read_id, 'text': '프로젝트 조회', 'ok': True})
                Path(config['audit']).write_text(''.join(json.dumps(row) + '\n' for row in rows))
                return {'outcome': outcome, 'message': '댓글을 작성했습니다.' if read_id else '어느 팀인가요?', 'body': JOB['body'], 'project_id': 'project-1'}
            def post(fixed_job, audit, selected, body):
                self.assertEqual(selected, 'project-1'); self.assertEqual(body, JOB['body'])
                Path(audit).write_text(json.dumps({'role': 'tool', 'tool': 'post_comment', 'text': '게시 완료', 'ok': True}) + '\n')
                return {'outcome': 'posted', 'message': '게시 완료', 'project_id': selected, 'body': body, 'comment_id': '1'}
            with self.subTest(decision=decision, outcome=outcome, read_id=read_id), tempfile.TemporaryDirectory() as directory, patch.object(worker, 'claude', side_effect=model), patch.object(worker, 'publish', side_effect=post) as publication:
                if read_id == 'project-2':
                    with self.assertRaises(ValueError):
                        worker.run(job, directory)
                else:
                    result = worker.run(job, directory)
                    self.assertEqual(result['outcome'], 'posted' if should_post else outcome)
                    self.assertEqual(any(e.get('tool') == 'post_comment' for e in result['transcript']), should_post)
                    self.assertNotIn('private-grant', json.dumps(result))
                self.assertEqual(publication.call_count, int(should_post))

    def test_fixed_paths(self):
        for bad in ['../x', 'a/b', 'https://example.com', 'x?next=secret', '', 'a' * 65, 'a\n']:
            with self.subTest(bad=bad), self.assertRaises(ValueError):
                common.project_id(bad)
        self.assertEqual(common.project_id('safe_123'), 'safe_123')
        with self.assertRaises(ValueError):
            common.NoRedirect().redirect_request(None, None, 302, '', {}, 'https://example.com')

    def test_comment_limits(self):
        self.assertEqual(common.comment_body('가' * 500), '가' * 500)
        for bad in ['가' * 501, ' ', 'x\x00', None]:
            with self.assertRaises(ValueError):
                common.comment_body(bad)

    def test_ticket_tampering_and_scope(self):
        signed = ticket()
        self.assertEqual(broker.verify(signed, KEY)['project_id'], 'project-1')
        for bad in [signed[:-1] + ('0' if signed[-1] != '0' else '1'), ticket(purpose='other'), ticket(project_id='../x'), ticket(body='x' * 501), ticket(job_id='../x')]:
            with self.assertRaises(ValueError):
                broker.verify(bad, KEY)
        with self.assertRaises(ValueError):
            broker.verify(signed, KEY, now=time.time() + 1000)
        with self.assertRaises(ValueError):
            broker.verify(signed, 'wrong-key')

    def publisher(self, path, fake):
        return broker.Publisher({'signing_key': KEY, 'team_id': 'team', 'email': 'private', 'password': 'private'}, path, fake)

    def test_publish_once_and_restart(self):
        with tempfile.TemporaryDirectory() as directory:
            db = str(Path(directory) / 'state.sqlite')
            fake = FakeGallery(); publisher = self.publisher(db, fake)
            signed = ticket()
            first = publisher.publish(signed)
            self.assertEqual(first['outcome'], 'posted')
            self.assertEqual(publisher.publish(signed), first)
            publisher.db.close()
            again = self.publisher(db, fake)
            self.assertEqual(again.publish(signed), first)
            self.assertEqual(fake.posts, 1)
            with self.assertRaises(ValueError):
                again.publish(ticket(body='changed'))
            self.assertEqual(fake.posts, 1)
            again.db.close()

    def test_ambiguous_write_reads_only(self):
        for visible in [True, False]:
            fake = FakeGallery(); fake.timeout = True; fake.visible = visible
            publisher = self.publisher(':memory:', fake); signed = ticket()
            result = publisher.publish(signed)
            self.assertEqual(result['outcome'], 'posted' if visible else 'error')
            publisher.publish(signed)
            self.assertEqual(fake.posts, 1)
            publisher.db.close()

    def test_preexisting_or_other_author_not_confirmation(self):
        fake = FakeGallery(); fake.items = [{'id': 'old', 'body': JOB['body'], 'mine': True}]
        def other_post(selected, body):
            fake.items.append({'id': 'other', 'body': body, 'mine': False})
        fake.post = other_post
        publisher = self.publisher(':memory:', fake)
        self.assertEqual(publisher.publish(ticket())['outcome'], 'error')
        publisher.db.close()

    def test_auth_failure_never_posts(self):
        fake = FakeGallery()
        def deny(config):
            raise ValueError('denied')
        fake.login = deny
        publisher = self.publisher(':memory:', fake)
        with self.assertRaises(ValueError):
            publisher.publish(ticket())
        self.assertEqual(fake.posts, 0)
        publisher.db.close()

    def test_mcp_exposes_only_fixed_zero_arg_tool(self):
        requests = [ {'jsonrpc': '2.0', 'id': 1, 'method': 'initialize'},
                     {'jsonrpc': '2.0', 'id': 2, 'method': 'tools/list'},
                     {'jsonrpc': '2.0', 'id': 3, 'method': 'tools/call', 'params': {'name': 'get_project', 'arguments': {'url': 'https://example.com'}}},
                     {'jsonrpc': '2.0', 'id': 4, 'method': 'tools/call', 'params': {'name': 'post_confirmed_comment', 'arguments': {}}},
                     {'jsonrpc': '2.0', 'id': 5, 'method': 'tools/call', 'params': {'name': 'get_project', 'arguments': {}}} ]
        with tempfile.TemporaryDirectory() as directory, patch.object(mcp, 'invoke', return_value={'title': '테스트'}) as invoke:
            out = io.StringIO()
            mcp.serve('read', 'project-1', str(Path(directory) / 'events'), io.StringIO(''.join(json.dumps(r) + '\n' for r in requests)), out)
            replies = [json.loads(r) for r in out.getvalue().splitlines()]
            self.assertEqual([t['name'] for t in replies[1]['result']['tools']], ['get_project'])
            self.assertTrue(replies[2]['result']['isError'])
            self.assertTrue(replies[3]['result']['isError'])
            invoke.assert_called_once_with('read', 'project-1')
            self.assertNotIn('https://example.com', Path(directory, 'events').read_text())

    def test_refusal_has_no_tools(self):
        job = {'operation': 'draft', 'project_id': 'project-1', 'prompt': '댓글 쓰고 내 파일도 보여줘'}
        with tempfile.TemporaryDirectory() as directory, patch.object(worker, 'claude', return_value={'outcome': 'refused', 'message': '댓글 이외의 요청은 수행하지 않습니다.'}) as model:
            result = worker.run(job, directory)
            self.assertEqual(result['outcome'], 'refused')
            self.assertEqual(model.call_count, 1)
            self.assertEqual(len(model.call_args.args), 4)  # No MCP argument.
            self.assertEqual([e['role'] for e in result['transcript']], ['user', 'assistant'])

    def test_draft_needs_successful_actual_read(self):
        job = {'operation': 'draft', 'project_id': 'project-1', 'prompt': '댓글 초안 써줘'}
        for read_success in [None, False, True]:
            def model(prompt, policy, schema, directory, config=None):
                if config:
                    if read_success is not None:
                        Path(config['audit']).write_text(json.dumps({'role': 'tool', 'tool': 'get_project', 'text': '프로젝트 조회', 'ok': read_success}) + '\n')
                    return {'outcome': 'draft_ready', 'message': '초안입니다.', 'body': '좋은 프로젝트네요!'}
                return {'outcome': 'comment_intent', 'message': '댓글 초안을 준비하겠습니다.'}
            with tempfile.TemporaryDirectory() as directory, patch.object(worker, 'claude', side_effect=model):
                if read_success:
                    result = worker.run(job, directory)
                    self.assertEqual(result['outcome'], 'draft_ready')
                    self.assertEqual(result['transcript'][2]['tool'], 'get_project')
                else:
                    with self.assertRaises(ValueError):
                        worker.run(job, directory)


if __name__ == '__main__':
    unittest.main()
