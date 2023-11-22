import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader';
import { Mesh, Scene } from 'three';

export function loadGLTFModel(scene: Scene, glbPath: string, options = { isReceiveShadow: true, isCastShadow: true }) {
  const { isReceiveShadow, isCastShadow } = options;
  return new Promise((resolve, reject) => {
    const loader = new GLTFLoader();

    loader.load(
      glbPath,
      (gltf) => {
        const obj = gltf.scene;
        obj.name = 'dog';
        obj.position.y = 0;
        obj.position.x = 0;
        obj.receiveShadow = isReceiveShadow;
        obj.castShadow = isCastShadow;
        scene.add(obj);

        obj.traverse(function (child) {
          if ((child as Mesh).isMesh) {
            child.castShadow = isCastShadow;
            child.receiveShadow = isReceiveShadow;
          }
        });
        resolve(obj);
      },
      undefined,
      function (error) {
        reject(error);
      },
    );
  });
}
