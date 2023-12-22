'use client';

import { Suspense, useContext, useEffect, useRef } from 'react';
import { Canvas, useFrame, useLoader } from '@react-three/fiber';
import { OrbitControls, useProgress } from '@react-three/drei';
import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader';
import { AnimationMixer, Mesh } from 'three';
import { GlobalContext } from './context/global';

const BoxMesh = (props: JSX.IntrinsicElements['mesh']) => {
  return (
    <mesh {...props}>
      <tetrahedronGeometry args={[1, 1]} />
      <meshToonMaterial wireframe={Math.random() < 0.7} color="cyan" />
    </mesh>
  );
};
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
    if (mesh && mesh.current) {
      mesh.current.rotation.y += -0.02;
    }
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
        <Canvas
          className={
            'bg-[linear-gradient(to_right,#808088_1px,transparent_1px),linear-gradient(to_bottom,#808080_1px,transparent_1px)] bg-[size:14px_24px]'
          }
          shadows
          camera={{ position: [90, 1, 5] }}
        >
          <OrbitControls autoRotate={true} enableZoom={false} />
          <PlaneMeshComponent />
          <BoxMesh scale={[10, 10, 10]} position={[-25, 35, 0]} />
          <BoxMesh rotation={[1, Math.PI / 2, 0]} scale={[10, 10, 10]} position={[25, 35, 0]} />
          <BoxMesh rotation={[0, Math.PI / 3, 0]} scale={[10, 10, 10]} position={[25, -25, 0]} />
          <BoxMesh rotation={[0, Math.PI / 2, 1]} scale={[10, 10, 10]} position={[-25, -25, 0]} />
          <BoxMesh rotation={[0, Math.PI / 2, -1]} scale={[10, 10, 10]} position={[-45, 0, 0]} />
          <BoxMesh rotation={[-2, Math.PI / 2, 1]} scale={[10, 10, 10]} position={[45, 0, 0]} />
          <BoxMesh rotation={[-2, Math.PI / 2, 1]} scale={[10, 10, 10]} position={[0, 35, 0]} />
          <BoxMesh rotation={[-2, Math.PI / 2, 1]} scale={[10, 10, 10]} position={[0, -35, 0]} />
          <pointLight color={'white'} position={[10, 10, 10]} intensity={1} />
          <ambientLight color={'white'} intensity={0.8} />
        </Canvas>
      </Suspense>
    </div>
  );
};

export { ThreeModel };
