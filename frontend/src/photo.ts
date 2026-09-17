const maxSide = 1600
const quality = 0.85
const outputType = 'image/jpeg'

export async function preparePhoto(file: File): Promise<File> {
  try {
    const image = await decodeSmall(file)

    const canvas = document.createElement('canvas')
    canvas.width = image.width
    canvas.height = image.height
    const context = canvas.getContext('2d')
    if (context === null) {
      image.close()
      return file
    }
    context.drawImage(image, 0, 0)
    image.close()

    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, outputType, quality))
    // Маленький снимок после пережатия может оказаться тяжелее: тогда он и не нужен.
    if (blob === null || blob.size >= file.size) {
      return file
    }
    return new File([blob], 'photo.jpg', { type: outputType })
  } catch {
    return file
  }
}

async function decodeSmall(file: File): Promise<ImageBitmap> {
  const options: ImageBitmapOptions = { imageOrientation: 'from-image', resizeQuality: 'high' }

  const size = await readSize(file)
  const scale = size === null ? 1 : Math.min(1, maxSide / Math.max(size.width, size.height))
  if (size !== null && scale < 1) {
    if (size.width >= size.height) {
      options.resizeWidth = Math.round(size.width * scale)
    } else {
      options.resizeHeight = Math.round(size.height * scale)
    }
  }

  return createImageBitmap(file, options)
}

function readSize(file: File): Promise<{ width: number; height: number } | null> {
  return new Promise((resolve) => {
    const url = URL.createObjectURL(file)
    const image = new Image()
    image.onload = () => {
      URL.revokeObjectURL(url)
      resolve({ width: image.naturalWidth, height: image.naturalHeight })
    }
    image.onerror = () => {
      URL.revokeObjectURL(url)
      resolve(null)
    }
    image.src = url
  })
}
