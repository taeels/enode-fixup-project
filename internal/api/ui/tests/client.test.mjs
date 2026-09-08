import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { ObservationClient, retryDelay } from '../static/shared/fleet/client.mjs';
const fixture = async name => JSON.parse(await readFile(new URL(`../testdata/obs-contract/${name}.json`, import.meta.url)));
const response = data => new Response(JSON.stringify(data));
const deferred = () => { let resolve; const promise = new Promise(r => resolve = r); return { promise, resolve }; };
test('public reads omit credentials and detail demand is deduplicated and pruned', async () => {
  const calls = [], nodes = await fixture('nodes-states');
  const c = new ObservationClient({mode:'demo',token:'must-not-send', fetcher:async(path,init)=>{calls.push([path,init]);return response(nodes);}});
  await c.request('nodes'); c.selectedRun = nodes.nodes.find(n=>n.lease).lease.run_id; c.syncDetails();
  assert.equal([...c.resources.keys()].filter(k=>k.startsWith('detail:')).length,3);
  assert.deepEqual(calls[0][1].headers,{}); assert.equal(calls[0][0],'/v1/nodes'); assert.equal(calls[0][1].credentials,'omit');
  c.resources.get('nodes').data.nodes=[];c.selectedRun=null;c.syncDetails();assert.equal(c.resources.size,2);c.stop();
});
test('same resource is never requested concurrently and stale success after stop is ignored',async()=>{
  const d=deferred();let calls=0;const c=new ObservationClient({mode:'fleet',token:'test-token',fetcher:()=>{calls++;return d.promise;}});
  const p=c.request('nodes');await c.request('nodes');assert.equal(calls,1);c.stop();d.resolve(response(await fixture('nodes-empty')));await p;assert.equal(c.resources.size,0);assert.equal(c.token,'');
});
test('late 401 from a pruned selection cannot end the active session',async()=>{
  const d=deferred();let unauthorized=0;const c=new ObservationClient({mode:'fleet',fetcher:()=>d.promise,onUnauthorized:()=>unauthorized++});
  c.selectedRun='old';c.syncDetails();const p=c.request('detail:old');c.selectedRun=null;c.syncDetails();d.resolve(new Response('',{status:401}));await p;assert.equal(unauthorized,0);c.stop();
});
test('current 401 clears protected data before notifying caller',async()=>{
  let calls=0;const c=new ObservationClient({mode:'fleet',token:'test-token',fetcher:async()=>new Response('',{status:401}),onUnauthorized:()=>{calls++;assert.equal(c.resources.size,0);assert.equal(c.token,'');}});
  await c.request('runs');assert.equal(calls,1);
});
test('resource failures do not erase other successful observations and recover independently',async()=>{
  const data=await fixture('nodes-empty');let ok=true,t=0;const c=new ObservationClient({mode:'demo',now:()=>t,fetcher:async()=>ok?response(data):new Response('',{status:503})});
  await c.request('nodes');ok=false;for(t=5000;t<=15000;t+=5000)await c.request('nodes');
  assert.equal(c.resources.get('nodes').failures,3);assert.equal(c.resources.get('nodes').frozenAt,15000);assert.deepEqual(c.resources.get('nodes').data,data);assert.equal(c.resources.get('runs').failures,0);
  ok=true;await c.request('nodes');assert.equal(c.resources.get('nodes').failures,0);assert.equal(c.resources.get('nodes').frozenAt,null);c.stop();
});
test('429 shares the retry boundary across public reads, including manual retry',async()=>{
  let t=0,calls=0;const c=new ObservationClient({mode:'demo',now:()=>t,fetcher:async()=>{calls++;return new Response('',{status:429,headers:{'Retry-After':'2'}});}});
  await c.request('nodes');await c.request('runs');await c.refresh(true);assert.equal(calls,1);t=2000;await c.request('runs');assert.equal(calls,2);c.stop();
  assert.equal(retryDelay('Wed, 09 Sep 2026 00:00:01 GMT',Date.parse('2026-09-09T00:00:00Z')),1000);
});
test('partial detail retains previous requirements but exposes warnings',async()=>{
  let data=await fixture('run-queued-nested');const c=new ObservationClient({mode:'demo',fetcher:async()=>response(data)});c.selectedRun=data.run_id;c.syncDetails();const key=`detail:${data.run_id}`;await c.request(key);
  const before=c.resources.get(key).data.requires;data={run_id:data.run_id,state:'QUEUED',warnings:['unavailable']};await c.request(key);
  assert.deepEqual(c.resources.get(key).previousRequires,before);assert.equal(c.resources.get(key).data.requires,undefined);assert.deepEqual(c.resources.get(key).data.warnings,['unavailable']);c.stop();
});
test('timeout counts once and a late body cannot replace the last observation',async()=>{
  const callbacks=[];const d=deferred();const timers={setTimeout(fn){callbacks.push(fn);return 1;},clearTimeout(){},clearInterval(){}};
  const c=new ObservationClient({mode:'demo',timers,fetcher:async()=>({ok:true,status:200,json:()=>d.promise})});
  const p=c.request('nodes');await Promise.resolve();callbacks[0]();await p;assert.equal(c.resources.get('nodes').failures,1);
  d.resolve(await fixture('nodes-empty'));await Promise.resolve();assert.equal(c.resources.get('nodes').data,null);c.stop();
});
test('invalid contract and malformed JSON remain observation errors',async()=>{
  const c=new ObservationClient({mode:'demo',fetcher:async()=>response({nodes:[]})});await c.request('nodes');assert.match(c.resources.get('nodes').error,/Invalid observation/);assert.equal(c.resources.get('nodes').data,null);c.stop();
});
