import { apiUpload } from './api'

const MAX_SIDE = 2048

// Large or unusual pictures are scaled down in the browser first (also turns HEIC into JPEG where
// the browser can open it); the server checks and re-encodes them anyway.
async function shrink(file) {
  if (file.size < 3 * 1024 * 1024 && /^image\/(jpeg|png|webp|gif)$/.test(file.type)) return file
  try {
    const bmp = await createImageBitmap(file)
    const k = Math.min(1, MAX_SIDE / Math.max(bmp.width, bmp.height))
    const canvas = document.createElement('canvas')
    canvas.width = Math.round(bmp.width * k)
    canvas.height = Math.round(bmp.height * k)
    canvas.getContext('2d').drawImage(bmp, 0, 0, canvas.width, canvas.height)
    const blob = await new Promise((r) => canvas.toBlob(r, 'image/jpeg', 0.9))
    return blob ? new File([blob], file.name.replace(/\.\w+$/, '') + '.jpg', { type: 'image/jpeg' }) : file
  } catch {
    return file
  }
}

export const isImageFile = (f) => f.type.startsWith('image/') || /\.(heic|heif)$/i.test(f.name)

/** Uploads a picture for a project (POST /images); returns { token, width, height, size }. */
export async function stageImage(file, onProgress) {
  const form = new FormData()
  form.append('file', await shrink(file))
  return apiUpload('/images', form, { onProgress })
}
