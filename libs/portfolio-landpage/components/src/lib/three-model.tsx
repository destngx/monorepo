'use client';

import { Suspense, useContext, useEffect, useRef } from 'react';
import { Canvas, useFrame, useLoader } from '@react-three/fiber';
import { OrbitControls, useProgress } from '@react-three/drei';
import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader';
import { AnimationMixer, Mesh } from 'three';
import { GlobalContext } from './context/global';

const PlaneMeshComponent = () => {
  const fileUrl = 'stylized_ww1_plane.glb';
  const mesh = useRef<Mesh>(null);
  const gltf = useLoader(GLTFLoader, fileUrl);

  let mixer: AnimationMixer;
  if (gltf.animations.length) {
    mixer = new AnimationMixer(gltf.scene);
    gltf.animations.forEach((clip) => {
      const action = mixer.clipAction(clip);
      action.play();
    });
  }

  useFrame((state, delta, frame) => {
    if (mixer) mixer?.update(delta);
    if (mesh && mesh.current) mesh.current.rotation.y += 0.01;
  });

  return (
    <mesh scale={[50, 50, 50]} receiveShadow={true} castShadow={true} ref={mesh}>
      <primitive object={gltf.scene} />
    </mesh>
  );
};

const ThreeModel = () => {
  const { progress } = useProgress();
  const { setLoadModelProgress } = useContext(GlobalContext);
  useEffect(() => {
    setLoadModelProgress(progress);
  }, [progress]);
  return (
    <div className={'flex h-40 md:h-[504px]'}>
      <Suspense>
        <Canvas shadows camera={{ position: [90, 1, 5] }}>
          <OrbitControls enableZoom={false} />
          <PlaneMeshComponent />
          <ambientLight color={'white'} intensity={0.8} />
        </Canvas>
      </Suspense>
    </div>
  );
};

export { ThreeModel };
