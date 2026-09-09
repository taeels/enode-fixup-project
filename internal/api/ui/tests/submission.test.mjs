import test from 'node:test';
import assert from 'node:assert/strict';
import { Submission, PENDING_KEY, requestID } from '../static/demo/submission.mjs';
const ID='11111111-1111-4111-8111-111111111111', ID2='22222222-2222-4222-8222-222222222222';
const options={submitter:'\uBC1D\uC740 \uC218\uB2EC',uuid:()=>ID};
const deferred=()=>{let resolve;const promise=new Promise(r=>resolve=r);return{promise,resolve};};
const reply=(status,body={},headers={})=>new Response(JSON.stringify(body),{status,headers});
const memory=()=>{const values=new Map();return {values,getItem:k=>values.get(k),setItem:(k,v)=>values.set(k,v),removeItem:k=>values.delete(k)};};
test('LAN HTTP fallback uses secure random bytes with UUID v4 and variant bits',()=>{assert.equal(requestID({getRandomValues:bytes=>bytes.fill(255)}),'ffffffff-ffff-4fff-bfff-ffffffffffff');assert.throws(()=>requestID({}));});
test('one intent blocks both scenarios and sends only the three agreed public fields',async()=>{
 const d=deferred(),calls=[],storage=memory();const s=new Submission({...options,storage,fetcher:(url,init)=>{calls.push([url,init]);return d.promise;}});
 const p=s.start('led-toggle');await s.start('welcome-audio');assert.equal(calls.length,1);assert.equal(s.state.phase,'pending');assert.deepEqual(JSON.parse(calls[0][1].body),{scenario_id:'led-toggle',submitter:options.submitter,request_id:ID});assert.deepEqual(calls[0][1].headers,{'Content-Type':'application/json'});
 d.resolve(reply(202,{run_id:'queued-1',state:'QUEUED'}));await p;assert.equal(s.state.phase,'accepted');assert.equal(storage.getItem(PENDING_KEY),undefined);
});
test('uncertain retries preserve request identity and never automatically resend',async()=>{
 const calls=[],storage=memory();const s=new Submission({...options,storage,fetcher:async(_,init)=>{calls.push(init.body);return reply(503);}});
 await s.start('led-toggle');assert.equal(s.state.phase,'uncertain');await s.start('welcome-audio');assert.equal(calls.length,1);await s.retry();assert.equal(calls[0],calls[1]);
 const restored=new Submission({...options,storage,fetcher:()=>{throw Error('must not auto submit');}});assert.equal(restored.state.phase,'uncertain');assert.equal(restored.state.intent.request_id,ID);
});
test('a migrated guest retries the exact legacy intent instead of renaming it',async()=>{
 const storage=memory(),intent={scenario_id:'welcome-audio',submitter:'guest-bright-otter',request_id:ID};
 storage.setItem(PENDING_KEY,JSON.stringify({...intent,started_at:1}));const calls=[];
 const s=new Submission({...options,storage,fetcher:async(_,init)=>{calls.push(JSON.parse(init.body));return reply(503);}});
 assert.equal(s.state.phase,'uncertain');assert.equal(calls.length,0);await s.retry();assert.deepEqual(calls[0],intent);
 s.newIntent();await s.start('led-toggle');assert.equal(calls[1].submitter,options.submitter);
});
test('new submissions require exactly two Hangul words with one space',async()=>{
 for(const name of ['guest-bright-otter',options.submitter.replace(' ','  '),' '+options.submitter,options.submitter+' ',options.submitter+' word','<script>']){
  const s=new Submission({...options,submitter:name,fetcher:()=>assert.fail('invalid name was submitted')});await s.start('led-toggle');assert.equal(s.state.intent,null);
 }
 const s=new Submission({...options,submitter:'\uAC00'.repeat(12)+' '+'\uB098'.repeat(12),fetcher:async()=>reply(503)});await s.start('welcome-audio');assert.equal(s.state.phase,'uncertain');
 const tooLong=new Submission({...options,submitter:'\uAC00'.repeat(13)+' '+'\uB098',fetcher:()=>assert.fail('oversized name was submitted')});await tooLong.start('led-toggle');assert.equal(tooLong.state.intent,null);
});
test('timeout covers response body and late old response cannot change a new intent',async()=>{
 const callbacks=[],d=deferred();let n=0;const timers={setTimeout(fn){callbacks.push(fn);return callbacks.length;},clearTimeout(){}};
 const s=new Submission({...options,uuid:()=>n++?ID2:ID,timers,fetcher:async()=>({status:201,json:()=>d.promise})});const p=s.start('led-toggle');await Promise.resolve();callbacks[0]();await p;assert.equal(s.state.phase,'uncertain');s.newIntent();const p2=s.start('welcome-audio');callbacks[1]();await p2;d.resolve({run_id:'old',state:'RUNNING'});await Promise.resolve();assert.equal(s.state.intent.request_id,ID2);assert.equal(s.state.phase,'uncertain');assert.equal(s.state.run,null);
});
for(const [code,phase] of [[400,'rejected'],[413,'rejected'],[415,'rejected'],[422,'rejected'],[404,'unavailable'],[409,'conflict'],[429,'limited'],[503,'uncertain']])test(`submission ${code} maps to ${phase}`,async()=>{let refresh=0;const s=new Submission({...options,fetcher:async()=>reply(code),onRefresh:()=>refresh++});await s.start('led-toggle');assert.equal(s.state.phase,phase);assert.equal(refresh,code===422?1:0);});
test('write Retry-After prevents manual retry until its deadline',async()=>{let now=0,calls=0;const s=new Submission({...options,now:()=>now,fetcher:async()=>{calls++;return reply(429,{}, {'Retry-After':'3'});}});await s.start('led-toggle');await s.retry();assert.equal(calls,1);now=3000;await s.retry();assert.equal(calls,2);});
test('malformed success stays uncertain; existing FAILED is an acknowledged terminal state',async()=>{
 const s=new Submission({...options,fetcher:async()=>reply(201,{run_id:'missing-state'})});await s.start('led-toggle');assert.equal(s.state.phase,'uncertain');
 const failed=new Submission({...options,fetcher:async()=>reply(200,{run_id:'existing',state:'FAILED'})});await failed.start('led-toggle');assert.equal(failed.state.run.state,'FAILED');assert.match(failed.state.message,/실패/);
});
test('invalid persisted fields are discarded, blocked storage and invalid UUID cannot leak submissions',async()=>{
 const storage=memory();storage.setItem(PENDING_KEY,JSON.stringify({request_id:ID,submitter:'<script>',scenario_id:'led-toggle',started_at:0}));const s=new Submission({...options,storage,uuid:()=> 'bad-id',fetcher:()=>assert.fail('unexpected request')});assert.equal(s.state.phase,'idle');await s.start('led-toggle');assert.equal(s.state.intent,null);
 const blocked=new Submission({...options,storage:{getItem(){throw Error('blocked');},setItem(){throw Error('blocked');},removeItem(){throw Error('blocked');}},fetcher:async()=>reply(503)});await blocked.start('led-toggle');assert.equal(blocked.state.intent.request_id,ID);
});
