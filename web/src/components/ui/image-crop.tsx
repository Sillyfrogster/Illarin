"use client";

import {
  type CSSProperties,
  createContext,
  type ReactNode,
  type RefObject,
  type SyntheticEvent,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import ReactCrop, {
  centerCrop,
  makeAspectCrop,
  type PercentCrop,
  type PixelCrop,
} from "react-image-crop";
import { cn } from "@/lib/cn";
import "react-image-crop/dist/ReactCrop.css";

const CROP_VARIABLES = {
  "--rc-border-color": "var(--v-over)",
  "--rc-focus-color": "var(--v-accent)",
  "--rc-drag-handle-bg-colour": "var(--v-action)",
} as CSSProperties;

const OUTPUT_TYPES = new Set(["image/jpeg", "image/webp", "image/png"]);

function centredCrop(width: number, height: number, aspect: number) {
  return centerCrop(
    makeAspectCrop({ unit: "%", width: 90 }, aspect, width, height),
    width,
    height,
  );
}

async function cutOut(
  image: HTMLImageElement,
  crop: PixelCrop,
  file: File,
): Promise<File> {
  const scaleX = image.naturalWidth / image.width;
  const scaleY = image.naturalHeight / image.height;
  const canvas = document.createElement("canvas");
  canvas.width = Math.round(crop.width * scaleX);
  canvas.height = Math.round(crop.height * scaleY);
  const context = canvas.getContext("2d");
  if (!context) throw new Error("The picture could not be cropped.");
  context.drawImage(
    image,
    crop.x * scaleX,
    crop.y * scaleY,
    canvas.width,
    canvas.height,
    0,
    0,
    canvas.width,
    canvas.height,
  );
  const type = OUTPUT_TYPES.has(file.type) ? file.type : "image/png";
  const blob = await new Promise<Blob | null>((resolve) =>
    canvas.toBlob(resolve, type, 0.92),
  );
  if (!blob) throw new Error("The picture could not be cropped.");
  const stem = file.name.replace(/\.[^.]+$/, "") || "picture";
  return new File([blob], `${stem}.${type.slice(6).replace("jpeg", "jpg")}`, {
    type,
  });
}

type Cropping = {
  aspect: number;
  crop: PercentCrop | undefined;
  cut: () => Promise<File>;
  image: RefObject<HTMLImageElement | null>;
  onChange: (crop: PercentCrop) => void;
  onComplete: (crop: PixelCrop) => void;
  onLoad: (event: SyntheticEvent<HTMLImageElement>) => void;
  reset: () => void;
  src: string;
};

const CropContext = createContext<Cropping | null>(null);

export function useImageCrop(): Cropping {
  const context = useContext(CropContext);
  if (!context) throw new Error("Image crop parts belong inside ImageCrop.");
  return context;
}

/** ImageCrop reads one file and holds the crop the person draws on it. */
export function ImageCrop({
  aspect,
  children,
  file,
}: {
  aspect: number;
  children: ReactNode;
  file: File;
}) {
  const image = useRef<HTMLImageElement | null>(null);
  const [src, setSrc] = useState("");
  const [crop, setCrop] = useState<PercentCrop>();
  const [initial, setInitial] = useState<PercentCrop>();
  const [completed, setCompleted] = useState<PixelCrop | null>(null);

  useEffect(() => {
    const url = URL.createObjectURL(file);
    setSrc(url);
    return () => URL.revokeObjectURL(url);
  }, [file]);

  const onLoad = useCallback(
    (event: SyntheticEvent<HTMLImageElement>) => {
      const { width, height } = event.currentTarget;
      const centred = centredCrop(width, height, aspect);
      setCrop(centred);
      setInitial(centred);
    },
    [aspect],
  );

  const cut = useCallback(async () => {
    if (!image.current) throw new Error("The picture has not loaded yet.");
    const chosen =
      completed ??
      pixels(
        crop ?? centredCrop(image.current.width, image.current.height, aspect),
        image.current,
      );
    return cutOut(image.current, chosen, file);
  }, [aspect, completed, crop, file]);

  const reset = useCallback(() => {
    setCrop(initial);
    setCompleted(null);
  }, [initial]);

  return (
    <CropContext.Provider
      value={{
        aspect,
        crop,
        cut,
        image,
        onChange: setCrop,
        onComplete: setCompleted,
        onLoad,
        reset,
        src,
      }}
    >
      {children}
    </CropContext.Provider>
  );
}

function pixels(crop: PercentCrop, image: HTMLImageElement): PixelCrop {
  return {
    unit: "px",
    x: (crop.x / 100) * image.width,
    y: (crop.y / 100) * image.height,
    width: (crop.width / 100) * image.width,
    height: (crop.height / 100) * image.height,
  };
}

/** ImageCropContent draws the picture with the crop frame over it. */
export function ImageCropContent({ className }: { className?: string }) {
  const { aspect, crop, image, onChange, onComplete, onLoad, src } =
    useImageCrop();

  return (
    <ReactCrop
      aspect={aspect}
      className={cn("max-h-[60dvh] max-w-full rounded-control", className)}
      crop={crop}
      keepSelection
      onChange={(_, percent) => onChange(percent)}
      onComplete={(pixel) => onComplete(pixel)}
      ruleOfThirds
      style={CROP_VARIABLES}
    >
      {src ? (
        <img
          alt="The picture to crop"
          className="max-h-[60dvh]"
          onLoad={onLoad}
          ref={image}
          src={src}
        />
      ) : null}
    </ReactCrop>
  );
}
