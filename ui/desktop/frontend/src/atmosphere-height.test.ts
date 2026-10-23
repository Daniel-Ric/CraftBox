import {expect,it} from 'vitest';
import {atmosphereHeight,normalizeAtmosphere} from './atmosphere-settings';
it('places signed offsets against the selected object boundary and preserves large negative input',()=>{
  const bounds={min:-40,max:200};
  expect(atmosphereHeight(-16,'top',bounds)).toBe(184);
  expect(atmosphereHeight(-16,'bottom',bounds)).toBe(-56);
  expect(atmosphereHeight(-16,'center',bounds)).toBe(64);
  expect(atmosphereHeight(-16,'world',bounds)).toBe(-16);
  expect(normalizeAtmosphere({cloudHeight:-1500,fogHeight:-1800}).cloudHeight).toBe(-1500);
  expect(normalizeAtmosphere({cloudHeight:-1500,fogHeight:-1800}).fogHeight).toBe(-1800);
});
