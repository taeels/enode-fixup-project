import test from 'node:test';
import assert from 'node:assert/strict';
import vm from 'node:vm';
import { readFile } from 'node:fs/promises';
const code=await readFile(new URL('../static/shared/guest.js',import.meta.url),'utf8');
function load(values,blocked=false){const context={window:{localStorage:{getItem:k=>{if(blocked)throw Error('blocked');return values.get(k)??null;},setItem:(k,v)=>{if(blocked)throw Error('blocked');values.set(k,v);}}}};vm.runInNewContext(code,context);return context.guest;}
for(const existing of ['guest-bright-otter','guest-custom-person'])test('preserves valid guest '+existing,()=>{const g=load(new Map([['enode.guest.id',existing]]));assert.equal(g.guestName(),existing);});
test('corrupt guest is repaired without resetting onboarding',()=>{const values=new Map([['enode.guest.id','<script>bad</script>'],['enode.guest.onboarded','1']]);const g=load(values);assert.match(g.guestName(),/^guest-[a-z]+-[a-z]+$/);assert.equal(g.hasOnboarded(),true);});
test('blocked storage preserves in-memory identity and does not throw',()=>{const g=load(new Map(),true);assert.equal(g.guestName(),g.guestName());g.markOnboarded();assert.equal(g.hasOnboarded(),false);});
