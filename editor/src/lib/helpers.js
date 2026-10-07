export const EMPHASIS_MARKERS = new Set([
  0x2A, // *
  0x5F, // _
  0x7E  // ~
]);

export function isCjkLetter(charCode) {
  if (!charCode || charCode < 0) {
    return false;
  }

  return (
    (charCode >= 0x3400 && charCode <= 0x4DBF) ||  // CJK Unified Ideographs Extension A
    (charCode >= 0x4E00 && charCode <= 0x9FFF) ||  // CJK Unified Ideographs
    (charCode >= 0xF900 && charCode <= 0xFAFF) ||  // CJK Compatibility Ideographs
    (charCode >= 0xFF01 && charCode <= 0xFF60) ||  // Full-width ASCII variants
    (charCode >= 0xFF61 && charCode <= 0xFF9F) ||  // Half-width Katakana
    (charCode >= 0xFFA0 && charCode <= 0xFFDC)     // Full-width Latin letters
  );
}

export function isCjkPunctuation(charCode) {
  if (!charCode || charCode < 0) {
    return false;
  }

  return (
    (charCode >= 0x3000 && charCode <= 0x303F) ||  // CJK 标点符号（。、，；：！？等）
    (charCode >= 0xFF01 && charCode <= 0xFF0F) ||  // 全角标点（！＂＃等）
    (charCode >= 0xFF1A && charCode <= 0xFF20) ||  // 全角标点（：；等）
    (charCode >= 0xFF3B && charCode <= 0xFF40) ||  // 全角标点（［＼等）
    (charCode >= 0xFF5B && charCode <= 0xFF65) ||  // 全角标点（｛｜等）
    (charCode >= 0xFE10 && charCode <= 0xFE1F) ||  // 竖排标点变体
    (charCode >= 0xFE30 && charCode <= 0xFE6F)     // CJK 兼容标点（小写变体）
  );
}

export function withTimeout(promise, ms, message = '操作超时') {
  return Promise.race([
    promise,
    new Promise((_, reject) => setTimeout(() => reject(new Error(message)), ms))
  ]);
}
