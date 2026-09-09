import test from 'node:test';
import assert from 'node:assert/strict';
import { TourProgress, TOUR_KEY } from '../static/demo/tour.mjs';
test('tour has four steps and writes only its own completion key',()=>{const values=new Map([['enode.guest.onboarded','1']]);const p=new TourProgress({getItem:k=>values.get(k),setItem:(k,v)=>values.set(k,v)});assert.equal(p.completed,false);for(let i=0;i<3;i++)assert.equal(p.next(),false);assert.equal(p.index,3);assert.equal(p.next(),true);assert.equal(values.get(TOUR_KEY),'1');assert.equal(values.get('enode.guest.onboarded'),'1');p.restart();assert.equal(p.index,0);});
test('skip and escape completion survive reentry, unavailable storage stays usable',()=>{const p=new TourProgress({getItem(){throw Error('blocked');},setItem(){throw Error('blocked');}});p.finish();assert.equal(p.completed,true);p.restart();assert.equal(p.index,0);const prior=new TourProgress({getItem:()=> '1'});assert.equal(prior.completed,true);});
