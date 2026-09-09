import test from 'node:test';
import assert from 'node:assert/strict';
import vm from 'node:vm';
import { readFile } from 'node:fs/promises';
const code=await readFile(new URL('../static/shared/guest.js',import.meta.url),'utf8');
function load(values,blocked=false){const context={window:{localStorage:{getItem:k=>{if(blocked)throw Error('blocked');return values.get(k)??null;},setItem:(k,v)=>{if(blocked)throw Error('blocked');values.set(k,v);}}}};vm.runInNewContext(code,context);return context.guest;}
const koreanName = '\uBC1D\uC740 \uC218\uB2EC';
test('preserves a Korean two-word guest across page loads',()=>{const values=new Map([['enode.guest.id',koreanName]]);assert.equal(load(values).guestName(),koreanName);assert.equal(load(values).guestName(),koreanName);});
test('migrates a known English name once without resetting onboarding',()=>{const values=new Map([['enode.guest.id','guest-bright-otter'],['enode.guest.onboarded','1']]);const g=load(values);assert.equal(g.guestName(),koreanName);assert.equal(values.get('enode.guest.id'),koreanName);assert.equal(load(values).guestName(),koreanName);assert.equal(g.hasOnboarded(),true);});
test('migrates other legacy names deterministically across separate tabs',()=>{const make=()=>load(new Map([['enode.guest.id','guest-custom-person']])).guestName();assert.equal(make(),make());assert.match(make(),/^[\uAC00-\uD7A3]{1,12} [\uAC00-\uD7A3]{1,12}$/);});
test('corrupt guest is repaired without resetting onboarding',()=>{const values=new Map([['enode.guest.id','<script>bad</script>'],['enode.guest.onboarded','1']]);const g=load(values);assert.match(g.guestName(),/^[\uAC00-\uD7A3]{1,12} [\uAC00-\uD7A3]{1,12}$/);assert.equal(g.hasOnboarded(),true);});
test('blocked storage preserves in-memory identity and does not throw',()=>{const g=load(new Map(),true);assert.equal(g.guestName(),g.guestName());g.markOnboarded();assert.equal(g.hasOnboarded(),false);});
test('readable but unwritable storage keeps the same migrated identity',()=>{const context={window:{localStorage:{getItem:()=> 'guest-bright-otter',setItem:()=>{throw Error('blocked write');}}}};vm.runInNewContext(code,context);assert.equal(context.guest.guestName(),koreanName);assert.equal(context.guest.guestName(),koreanName);});
