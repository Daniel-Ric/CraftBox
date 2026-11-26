package offlinerender

const blenderScript = `import bpy
import bmesh
import base64
import json
import math
import os
import sys
from mathutils import Matrix, Quaternion, Vector

model, config, output, image_format = sys.argv[sys.argv.index('--') + 1:]
with open(config, encoding='utf-8') as file:
    settings = json.load(file)

def vector(point):
    return Vector((point[0], -point[2], point[1]))

def rgb(color):
    values = [int(color[i:i+2], 16) / 255 for i in (1, 3, 5)]
    return tuple(v / 12.92 if v <= 0.04045 else ((v + 0.055) / 1.055) ** 2.4 for v in values)

def aim(obj, position, target):
    obj.location = vector(position)
    direction = vector(target) - obj.location
    if direction.length_squared > 0:
        obj.rotation_euler = direction.to_track_quat('-Z', 'Y').to_euler()

def light_object(name, kind):
    data = bpy.data.lights.new(name, kind)
    obj = bpy.data.objects.new(name, data)
    scene.collection.objects.link(obj)
    return obj, data

def gobo(source, angle, index, nodes, links):
    path = os.path.join(os.getcwd(), 'projection-%d.png' % index)
    with open(path, 'wb') as file:
        file.write(base64.b64decode(source.split(',', 1)[1], validate=True))
    image = bpy.data.images.load(path)
    texture = nodes.new('ShaderNodeTexImage')
    texture.image = image
    texture.extension = 'CLIP'
    coordinates = nodes.new('ShaderNodeTexCoord')
    separate = nodes.new('ShaderNodeSeparateXYZ')
    links.new(coordinates.outputs['Normal'], separate.inputs[0])
    scale = 0.5 / math.tan(math.radians(angle))
    combine = nodes.new('ShaderNodeCombineXYZ')
    for axis, sign in (('X', -1), ('Y', -1)):
        divide = nodes.new('ShaderNodeMath')
        divide.operation = 'DIVIDE'
        links.new(separate.outputs[axis], divide.inputs[0])
        links.new(separate.outputs['Z'], divide.inputs[1])
        transform = nodes.new('ShaderNodeMath')
        transform.operation = 'MULTIPLY_ADD'
        links.new(divide.outputs[0], transform.inputs[0])
        transform.inputs[1].default_value = scale * sign
        transform.inputs[2].default_value = 0.5
        links.new(transform.outputs[0], combine.inputs[axis])
    links.new(combine.outputs[0], texture.inputs['Vector'])
    return texture.outputs['Color']

def light_shader(data, spec, index):
    source = spec.get('gobo')
    reach = spec.get('range', 0)
    if not source and reach <= 0:
        return
    data.use_nodes = True
    nodes = data.node_tree.nodes
    links = data.node_tree.links
    emission = next(node for node in nodes if node.type == 'EMISSION')
    color = (*rgb(spec['color']), 1)
    data.color = (1, 1, 1)
    emission.inputs['Color'].default_value = color
    if source and spec['type'] == 'spot':
        mix = nodes.new('ShaderNodeMixRGB')
        mix.blend_type = 'MULTIPLY'
        mix.inputs[0].default_value = 1
        mix.inputs[1].default_value = color
        links.new(gobo(source, spec['angle'], index, nodes, links), mix.inputs[2])
        links.new(mix.outputs[0], emission.inputs['Color'])
    if reach > 0:
        path = nodes.new('ShaderNodeLightPath')
        falloff = nodes.new('ShaderNodeMapRange')
        falloff.interpolation_type = 'SMOOTHERSTEP'
        falloff.inputs['From Min'].default_value = reach * 0.75
        falloff.inputs['From Max'].default_value = reach
        falloff.inputs['To Min'].default_value = 1
        falloff.inputs['To Max'].default_value = 0
        links.new(path.outputs['Ray Length'], falloff.inputs['Value'])
        multiplier = nodes.new('ShaderNodeMath')
        multiplier.operation = 'MULTIPLY'
        multiplier.inputs[1].default_value = emission.inputs['Strength'].default_value
        links.new(falloff.outputs[0], multiplier.inputs[0])
        links.new(multiplier.outputs[0], emission.inputs['Strength'])

bpy.ops.object.select_all(action='SELECT')
bpy.ops.object.delete(use_global=False)
bpy.ops.import_scene.gltf(filepath=model)
scene = bpy.context.scene
meshes = [obj for obj in scene.objects if obj.type == 'MESH']

for material in bpy.data.materials:
    if not material.use_nodes or 'clay' not in material:
        continue
    surface = next((node for node in material.node_tree.nodes if node.type == 'BSDF_PRINCIPLED'), None)
    if surface:
        color = surface.inputs['Base Color']
        for link in list(color.links):
            material.node_tree.links.remove(link)
        color.default_value = (*rgb(material['clay']), 1)

for obj in meshes:
    if not settings['planes']:
        break
    mesh = bmesh.new()
    mesh.from_mesh(obj.data)
    world = obj.matrix_world
    for plane in settings['planes']:
        normal = vector(plane['normal'])
        position = world.inverted() @ (-plane['offset'] * normal)
        direction = world.to_3x3().transposed() @ normal
        bmesh.ops.bisect_plane(mesh, geom=list(mesh.verts) + list(mesh.edges) + list(mesh.faces),
                             dist=0.000001, plane_co=position, plane_no=direction, clear_inner=True)
    mesh.to_mesh(obj.data)
    mesh.free()
    obj.data.update()

camera_data = bpy.data.cameras.new('CraftBox camera')
camera = bpy.data.objects.new('CraftBox camera', camera_data)
scene.collection.objects.link(camera)
scene.camera = camera
camera.location = vector(settings['position'])
x, y, z, w = settings['rotation']
camera.rotation_mode = 'QUATERNION'
camera.rotation_quaternion = Matrix.Rotation(math.pi / 2, 3, 'X').to_quaternion() @ Quaternion((w, x, y, z))
camera_data.clip_start = settings['near']
camera_data.clip_end = settings['far']
camera_data.shift_y = settings.get('verticalShift', 0)
if 'orthoScale' in settings:
    camera_data.type = 'ORTHO'
    camera_data.ortho_scale = settings['orthoScale'] * max(1, settings['height'] / settings['width'])
else:
    camera_data.sensor_fit = 'VERTICAL'
    camera_data.angle_y = math.radians(settings['fov'])
    if settings.get('dof'):
        camera_data.dof.use_dof = True
        camera_data.dof.focus_distance = settings['dof']['distance']
        camera_data.dof.aperture_fstop = settings['dof']['aperture']

sun = settings['sun']
sun_obj, sun_data = light_object('Sun', 'SUN')
aim(sun_obj, sun['position'], sun['target'])
sun_data.color = rgb(sun['color'])
sun_data.energy = sun['power']
sun_data.angle = min(math.pi, math.radians(sun['softness']))

linked = {}
for index, spec in enumerate(settings['studio']):
    obj, data = light_object(spec['name'], spec['type'].upper())
    aim(obj, spec['position'], spec['target'])
    data.color = rgb(spec['color'])
    data.energy = spec['power'] / 683
    data.use_shadow = spec['shadows'] if spec['type'] != 'area' else True
    if spec['type'] == 'spot':
        data.spot_size = math.radians(spec['angle'] * 2)
        data.spot_blend = spec['penumbra']
    if spec['type'] == 'area':
        data.shape = 'RECTANGLE'
        data.size = spec['width']
        data.size_y = spec['height']
    light_shader(data, spec, index)
    receivers = spec.get('receivers') or ([spec['receiver']] if spec.get('receiver') else [])
    if receivers:
        excluded = spec.get('receiverMode') == 'exclude'
        link_id = (tuple(receivers), excluded)
        if link_id not in linked:
            collection = bpy.data.collections.new('Light receiver ' + spec['name'])
            for mesh_obj in meshes:
                if (mesh_obj.get('receiver') in receivers) != excluded:
                    collection.objects.link(mesh_obj)
            linked[link_id] = collection
        collection = linked[link_id]
        if len(collection.objects):
            obj.light_linking.receiver_collection = collection
        else:
            data.energy = 0

scene.world = bpy.data.worlds.new('CraftBox world')
scene.world.use_nodes = True
world = scene.world.node_tree
ambient = world.nodes.get('Background')
sky_colors = {'daylight': '#a9cbff', 'dawn': '#b8a9dc', 'sunset': '#efae91', 'night': '#354579'}
ambient.inputs['Color'].default_value = (*rgb(sky_colors.get(settings['sky'], '#ffffff')), 1)
ambient.inputs['Strength'].default_value = settings['ambient']
backdrop = world.nodes.new('ShaderNodeBackground')
backdrop.inputs['Color'].default_value = (*rgb(sky_colors.get(settings['sky'], settings['background'])), 1)
if settings.get('skyImage'):
    sky_path = os.path.join(os.getcwd(), 'environment.exr')
    with open(sky_path, 'wb') as file:
        file.write(base64.b64decode(settings['skyImage'], validate=True))
    env = world.nodes.new('ShaderNodeTexEnvironment')
    env.image = bpy.data.images.load(sky_path)
    coordinates = world.nodes.new('ShaderNodeTexCoord')
    rotation = world.nodes.new('ShaderNodeMapping')
    rotation.vector_type = 'VECTOR'
    rotation.inputs['Rotation'].default_value = (math.radians(settings['skyElevation']), 0, -math.radians(settings['skyRotation']))
    world.links.new(coordinates.outputs['Generated'], rotation.inputs['Vector'])
    world.links.new(rotation.outputs[0], env.inputs['Vector'])
    world.links.new(env.outputs['Color'], ambient.inputs['Color'])
    world.links.new(env.outputs['Color'], backdrop.inputs['Color'])
    ambient.inputs['Strength'].default_value = settings['environment']
    backdrop.inputs['Strength'].default_value = settings['skyBrightness']
path = world.nodes.new('ShaderNodeLightPath')
mix = world.nodes.new('ShaderNodeMixShader')
world.links.new(path.outputs['Is Camera Ray'], mix.inputs[0])
world.links.new(ambient.outputs['Background'], mix.inputs[1])
world.links.new(backdrop.outputs[0], mix.inputs[2])
world.links.new(mix.outputs[0], world.nodes.get('World Output').inputs['Surface'])

fog = settings.get('fog')
if fog:
    low = vector(fog['min'])
    high = vector(fog['max'])
    center = (low + high) / 2
    extent = (high - low) * 0.5
    volume_material = bpy.data.materials.new('CraftBox fog')
    volume_material.use_nodes = True
    nodes = volume_material.node_tree.nodes
    nodes.clear()
    volume = nodes.new('ShaderNodeVolumePrincipled')
    volume.inputs['Density'].default_value = fog['density'] / max(8, abs(extent.z))
    volume.inputs['Color'].default_value = (*rgb(fog['color']), 1)
    volume.inputs['Anisotropy'].default_value = fog['anisotropy']
    volume_material.node_tree.links.new(volume.outputs[0], nodes.new('ShaderNodeOutputMaterial').inputs['Volume'])
    bpy.ops.mesh.primitive_cube_add(size=2, location=center)
    atmosphere = bpy.context.object
    atmosphere.name = 'CraftBox fog'
    atmosphere.scale = tuple(max(4, abs(value) * 1.25) for value in extent)
    atmosphere.data.materials.append(volume_material)

scene.render.engine = 'CYCLES'
scene.cycles.samples = settings['samples']
scene.cycles.use_denoising = True
scene.render.resolution_x = settings['width']
scene.render.resolution_y = settings['height']
scene.render.resolution_percentage = 100
scene.render.film_transparent = settings['transparent']
scene.view_settings.exposure = math.log2(max(0.0001, settings['exposure']))
scene.render.image_settings.color_mode = 'RGBA'
if image_format == 'exr':
    image_settings = scene.render.image_settings
    if hasattr(image_settings, 'media_type'):
        image_settings.media_type = 'MULTI_LAYER_IMAGE'
    else:
        image_settings.file_format = 'OPEN_EXR_MULTILAYER'
    image_settings.color_depth = '32'
    image_settings.exr_codec = 'ZIP'
    if hasattr(image_settings, 'use_exr_interleave'):
        image_settings.use_exr_interleave = True
    layer = bpy.context.view_layer
    for name in ('z', 'normal', 'emit', 'cryptomatte_object', 'cryptomatte_material', 'diffuse_color', 'diffuse_direct', 'diffuse_indirect'):
        setattr(layer, 'use_pass_' + name, True)
    terrain_collection = bpy.data.collections.new('Terrain sources')
    shadow_collection = bpy.data.collections.new('Shadow catchers')
    scene.collection.children.link(terrain_collection)
    scene.collection.children.link(shadow_collection)
    for obj in meshes:
        catcher = obj.copy()
        catcher.is_shadow_catcher = True
        shadow_collection.objects.link(catcher)
        for collection in list(obj.users_collection):
            if collection not in linked.values():
                collection.objects.unlink(obj)
        terrain_collection.objects.link(obj)
        for (receivers, excluded), collection in linked.items():
            if (obj.get('receiver') in receivers) != excluded:
                collection.objects.link(catcher)
    layer.layer_collection.children[shadow_collection.name].exclude = True
    shadow_layer = scene.view_layers.new('Shadows')
    shadow_layer.layer_collection.children[terrain_collection.name].exclude = True
    shadow_layer.cycles.use_pass_shadow_catcher = True
else:
    scene.render.image_settings.file_format = 'PNG'
scene.render.use_file_extension = False
scene.render.filepath = output
bpy.ops.render.render(write_still=True)
`
