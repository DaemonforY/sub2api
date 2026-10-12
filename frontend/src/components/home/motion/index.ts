// Everything that needs three.js, behind one dynamic import (see HomeMotion.vue).
import { createStage, type MotionOptions } from './stage'
import { riverScene, type RiverContext } from './river'
import { hiveScene } from './hive'
import { assembleScene, type AssembleContext } from './assemble'
import { teardownScene, type TeardownContext } from './teardown'

export interface MotionContexts {
  river: RiverContext
  teardown: TeardownContext
  hive: undefined
  assemble: AssembleContext
}
export type MotionKind = keyof MotionContexts

export function mountMotion<K extends MotionKind>(kind: K, host: HTMLElement, opts: MotionOptions, ctx: MotionContexts[K]): () => void {
  return createStage(host, opts, (renderer, o) => {
    if (kind === 'river') return riverScene(o, ctx as RiverContext)
    if (kind === 'teardown') return teardownScene(renderer, o, ctx as TeardownContext)
    if (kind === 'assemble') return assembleScene(o, ctx as AssembleContext)
    return hiveScene(o)
  })
}
