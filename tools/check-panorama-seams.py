"""Measure matching world rays on adjacent overscanned cube-face renders."""
import json
import math
from pathlib import Path
from PIL import Image

faces = [((1, 0, 0), (0, 1, 0)), ((-1, 0, 0), (0, 1, 0)), ((0, 1, 0), (0, 0, 1)), ((0, -1, 0), (0, 0, -1)), ((0, 0, 1), (0, 1, 0)), ((0, 0, -1), (0, 1, 0))]
images = [Image.open(Path('tmp/studio-lights') / f'face-{i}.png') for i in range(6)]
dot = lambda a, b: sum(x * y for x, y in zip(a, b))
cross = lambda a, b: (a[1]*b[2]-a[2]*b[1], a[2]*b[0]-a[0]*b[2], a[0]*b[1]-a[1]*b[0])

def sample(index, direction):
    forward, up = faces[index]
    d = dot(direction, forward) * math.tan(math.radians(46))
    u = .5 + dot(direction, cross(forward, up)) / (2 * d)
    v = .5 - dot(direction, up) / (2 * d)
    image = images[index]
    x, y = u * image.width - .5, v * image.height - .5
    ix, iy = int(x), int(y)
    color = [0, 0, 0]
    for sy in range(2):
        for sx in range(2):
            weight = (x - ix if sx else 1 - x + ix) * (y - iy if sy else 1 - y + iy)
            pixel = image.getpixel((min(image.width - 1, ix + sx), min(image.height - 1, iy + sy)))
            for c in range(3):
                color[c] += pixel[c] * weight
    return color

result = []
for a in range(6):
    for b in range(a + 1, 6):
        if dot(faces[a][0], faces[b][0]) != 0:
            continue
        axis = cross(faces[a][0], faces[b][0])
        differences = []
        for t in range(-9, 10):
            ray = tuple(faces[a][0][i] + faces[b][0][i] + axis[i] * t / 10 for i in range(3))
            ca, cb = sample(a, ray), sample(b, ray)
            differences.append(max(abs(x - y) for x, y in zip(ca, cb)))
        result.append(dict(faces=[a, b], max=max(differences), mean=sum(differences)/len(differences)))
print(json.dumps(result, indent=2))
