// Bundles three.js (MIT) for the player into public/player/vendor/three.js as a classic script that
// sets window.THREE. The player has no network, so scenes that use S.three load it from the site;
// the runtime only fetches it when a scene's code mentions S.three.
import { build } from 'esbuild'

await build({
  stdin: {
    contents: "export * from 'three'; export { RoomEnvironment } from 'three/examples/jsm/environments/RoomEnvironment.js'; export { RoundedBoxGeometry } from 'three/examples/jsm/geometries/RoundedBoxGeometry.js';",
    resolveDir: new URL('..', import.meta.url).pathname,
  },
  bundle: true,
  format: 'iife',
  globalName: 'THREE',
  minify: true,
  legalComments: 'inline',
  target: 'es2020',
  outfile: new URL('../public/player/vendor/three.js', import.meta.url).pathname,
})
console.log('three.js bundled')
