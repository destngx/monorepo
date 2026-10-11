import {
  BackSide,
  Box3,
  Color,
  DataTexture,
  Mesh,
  MeshToonMaterial,
  NearestFilter,
  RedFormat,
  ShaderMaterial,
  Vector3,
  type ColorRepresentation,
  type Material,
  type Object3D,
  type Texture,
} from 'three';

export type ToonOptions = {
  ink: ColorRepresentation;
  /** Outline width in world units, applied after the object's own transforms. */
  outlineThickness?: number;
};

const OUTLINE_VERTEX = /* glsl */ `
  uniform float thickness;
  void main() {
    // Displace in view space so the hull is equally thick on every mesh, whatever its local scale.
    vec4 viewPosition = modelViewMatrix * vec4(position, 1.0);
    viewPosition.xyz += normalize(normalMatrix * normal) * thickness;
    gl_Position = projectionMatrix * viewPosition;
  }
`;

const OUTLINE_FRAGMENT = /* glsl */ `
  uniform vec3 color;
  void main() {
    gl_FragColor = vec4(color, 1.0);
  }
`;

/** Three hard light bands: shadow, mid, light. This is what makes it look cel-shaded. */
const createToonGradient = (): DataTexture => {
  const texture = new DataTexture(new Uint8Array([70, 160, 255]), 3, 1, RedFormat);
  texture.minFilter = NearestFilter;
  texture.magFilter = NearestFilter;
  texture.needsUpdate = true;
  return texture;
};

type ColouredMaterial = Material & { color?: Color; map?: Texture | null };

const toToonMaterial = (source: ColouredMaterial, gradientMap: DataTexture): MeshToonMaterial =>
  new MeshToonMaterial({
    color: source.color?.clone() ?? new Color('#ffffff'),
    map: source.map ?? null,
    gradientMap,
    transparent: source.transparent,
    opacity: source.opacity,
  });

const createOutline = (mesh: Mesh, ink: Color, thickness: number): Mesh => {
  const outline = new Mesh(
    mesh.geometry,
    new ShaderMaterial({
      uniforms: { color: { value: ink.clone() }, thickness: { value: thickness } },
      vertexShader: OUTLINE_VERTEX,
      fragmentShader: OUTLINE_FRAGMENT,
      side: BackSide,
    }),
  );
  outline.userData.isOutline = true;
  outline.raycast = () => undefined;
  return outline;
};

/**
 * Converts every mesh under `root` to toon shading with an inverted-hull ink outline.
 * Safe to call again (for example on theme change): it only recolours existing outlines.
 */
const isMesh = (object: Object3D): object is Mesh => object instanceof Mesh;

export function toonify(root: Object3D, { ink, outlineThickness = 0.02 }: ToonOptions): void {
  const inkColor = new Color(ink);
  const gradientMap = createToonGradient();
  const meshes: Mesh[] = [];

  root.traverse((child) => {
    if (isMesh(child) && !child.userData.isOutline) meshes.push(child);
  });

  for (const mesh of meshes) {
    const existing = mesh.children.find((child): child is Mesh => isMesh(child) && Boolean(child.userData.isOutline));
    if (existing) {
      ((existing.material as ShaderMaterial).uniforms.color.value as Color).copy(inkColor);
      continue;
    }

    mesh.material = Array.isArray(mesh.material)
      ? mesh.material.map((material) => toToonMaterial(material as ColouredMaterial, gradientMap))
      : toToonMaterial(mesh.material, gradientMap);
    mesh.add(createOutline(mesh, inkColor, outlineThickness));
  }
}

const COLLAPSED_SCALE = 1e-6;

// Read axis lengths from the matrix directly: `Matrix4.decompose` reports a scale of 1 for singular matrices.
const isCollapsed = (object: Object3D): boolean => {
  const e = object.matrixWorld.elements;
  return (
    Math.min(Math.hypot(e[0], e[1], e[2]), Math.hypot(e[4], e[5], e[6]), Math.hypot(e[8], e[9], e[10])) <
    COLLAPSED_SCALE
  );
};

/**
 * Bounds of the meshes that can actually be seen. Meshes collapsed to zero scale (for example particles
 * that an animation scales in later) still count as a point for `Box3.setFromObject`, which would
 * otherwise skew framing towards wherever they happen to be parked.
 */
function visibleBounds(root: Object3D): Box3 {
  const box = new Box3();
  const meshBox = new Box3();
  root.traverse((child) => {
    if (!isMesh(child)) return;
    if (isCollapsed(child)) return;
    child.geometry.computeBoundingBox();
    if (child.geometry.boundingBox) box.union(meshBox.copy(child.geometry.boundingBox).applyMatrix4(child.matrixWorld));
  });
  return box;
}

/**
 * Uniformly scales `root` so the largest visible dimension of `focus` equals `size`, and centres `focus` on the
 * origin. `focus` defaults to `root`; pass a sub-object to frame the subject rather than effects around it.
 */
export function fitToSize(root: Object3D, size: number, focus: Object3D = root): void {
  root.updateMatrixWorld(true);
  const box = visibleBounds(focus);
  if (box.isEmpty()) return;
  const dimensions = box.getSize(new Vector3());
  const center = box.getCenter(new Vector3());
  const largest = Math.max(dimensions.x, dimensions.y, dimensions.z);
  if (largest === 0) return;

  const scale = size / largest;
  root.scale.multiplyScalar(scale);
  root.position.sub(center.multiplyScalar(scale));
}
