"""Floyd-Steinberg quantization shared by the Badger image converters."""


def quantize(values, width, height, bits, bounds=None):
    if width < 1 or height < 1 or len(values) != width * height:
        raise ValueError("pixel count must match positive image dimensions")
    if bits not in (2, 4):
        raise ValueError("expected 2bpp or 4bpp grayscale")
    left, top, right, bottom = bounds if bounds is not None else (0, 0, width, height)
    if not (0 <= left < right <= width and 0 <= top < bottom <= height):
        raise ValueError("diffusion bounds must be inside the image")

    maximum = (1 << bits) - 1
    step = 255 // maximum
    per_byte = 8 // bits
    packed = bytearray((width * height + per_byte - 1) // per_byte)
    for y in range(height):
        for x in range(width):
            index = y * width + x
            value = max(0, min(255, values[index]))
            shade = min(maximum, int(value / step + 0.5))
            packed[index // per_byte] |= shade << ((per_byte - 1 - index % per_byte) * bits)
            error = value - shade * step
            for dx, dy, weight in ((1, 0, 7), (-1, 1, 3), (0, 1, 5), (1, 1, 1)):
                nx, ny = x + dx, y + dy
                if left <= nx < right and top <= ny < bottom:
                    values[ny * width + nx] += error * weight / 16
    return bytes(packed)
