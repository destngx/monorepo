import { BoxGeometry, Color, Group, Mesh, MeshStandardMaterial, MeshToonMaterial, ShaderMaterial } from 'three';
import { describe, expect, it } from 'vitest';
import { fitToSize, toonify } from './toon';

const buildModel = () => {
  const root = new Group();
  const mesh: Mesh = new Mesh(new BoxGeometry(1, 1, 1), new MeshStandardMaterial({ color: '#ff0000' }));
  root.add(mesh);
  return { root, mesh };
};

describe('toonify', () => {
  it('swaps materials for toon materials that keep the original colour', () => {
    const { root, mesh } = buildModel();

    toonify(root, { ink: '#000000' });

    expect(mesh.material).toBeInstanceOf(MeshToonMaterial);
    expect((mesh.material as MeshToonMaterial).color.equals(new Color('#ff0000'))).toBe(true);
  });

  it('adds one ink outline hull per mesh, and is idempotent', () => {
    const { root, mesh } = buildModel();

    toonify(root, { ink: '#000000' });
    toonify(root, { ink: '#000000' });

    const outlines = mesh.children.filter((child) => child.userData.isOutline);
    expect(outlines).toHaveLength(1);
    expect((outlines[0] as Mesh).material).toBeInstanceOf(ShaderMaterial);
  });

  it('can recolour existing outlines when the theme changes', () => {
    const { root, mesh } = buildModel();
    toonify(root, { ink: '#000000' });

    toonify(root, { ink: '#ffffff' });

    const outline = mesh.children.find((child) => child.userData.isOutline) as Mesh;
    const uniforms = (outline.material as ShaderMaterial).uniforms;
    expect((uniforms.color.value as Color).equals(new Color('#ffffff'))).toBe(true);
  });
});

describe('fitToSize', () => {
  it('scales the largest dimension to the target and centres the object', () => {
    const root = new Group();
    const mesh = new Mesh(new BoxGeometry(10, 2, 4));
    mesh.position.set(5, 5, 5);
    root.add(mesh);

    fitToSize(root, 4);

    expect(root.scale.x).toBeCloseTo(0.4);
    expect(root.position.x).toBeCloseTo(-2);
    expect(root.position.y).toBeCloseTo(-2);
  });

  it('ignores collapsed (zero-scale) meshes, such as particles waiting to animate in', () => {
    const root = new Group();
    root.add(new Mesh(new BoxGeometry(2, 2, 2)));
    const particle = new Mesh(new BoxGeometry(1, 1, 1));
    particle.position.set(100, 0, 0);
    particle.scale.setScalar(0);
    root.add(particle);

    fitToSize(root, 4);

    expect(root.scale.x).toBeCloseTo(2);
    expect(root.position.x).toBeCloseTo(0);
  });

  it('frames a focus object while scaling the whole root, so effects around it stay in proportion', () => {
    const root = new Group();
    const body = new Mesh(new BoxGeometry(1, 1, 1));
    body.position.set(3, 0, 0);
    const trail = new Mesh(new BoxGeometry(1, 1, 1));
    trail.position.set(-50, 0, 0);
    root.add(body, trail);

    fitToSize(root, 2, body);

    expect(root.scale.x).toBeCloseTo(2);
    expect(root.position.x).toBeCloseTo(-6);
  });
});
