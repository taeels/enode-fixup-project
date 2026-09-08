import test from 'node:test';
import assert from 'node:assert/strict';
import { validToken } from '../static/fleet/login.mjs';
test('token validation rejects empty input and header line breaks without rewriting a token',()=>{for(const value of ['', '  ', 'token\nnext', 'token\rnext', null])assert.equal(validToken(value),false);assert.equal(validToken('test-token'),true);});
