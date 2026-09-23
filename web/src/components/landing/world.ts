function context(canvas: HTMLCanvasElement, alpha = true) {
  const result = canvas.getContext("2d", { alpha });
  if (!result) throw new Error("Canvas is unavailable");
  return result;
}

export const clamp = (n: number, a = 0, b = 1) => Math.max(a, Math.min(b, n));
export const mix = (a: number, b: number, t: number) => a + (b - a) * t;
export const smooth = (a: number, b: number, n: number) => {
  const t = clamp((n - a) / (b - a));
  return t * t * (3 - 2 * t);
};
const hash = (i: number) => {
  const n = Math.sin(i * 127.1 + 311.7) * 43758.5453;
  return n - Math.floor(n);
};
const route = [
  [0, 0.67, 0.3],
  [0.08, 0.78, 0.5],
  [0.17, 0.62, 0.28],
  [0.27, 0.57, 0.4],
  [0.37, 0.79, 0.44],
  [0.48, 0.66, 0.46],
  [0.59, 0.52, 0.46],
  [0.69, 0.76, 0.37],
  [0.77, 0.67, 0.32],
  [0.87, 0.72, 0.58],
  [1, 0.73, 0.73],
];

function guide(p: number) {
  let k = 0;
  while (k < route.length - 2 && p > route[k + 1][0]) k++;
  const a = route[Math.max(0, k - 1)],
    b = route[k],
    c = route[k + 1],
    d = route[Math.min(route.length - 1, k + 2)];
  const t = clamp((p - b[0]) / (c[0] - b[0]));
  return [1, 2].map(
    (j) =>
      0.5 *
      (2 * b[j] +
        (-a[j] + c[j]) * t +
        (2 * a[j] - 5 * b[j] + 4 * c[j] - d[j]) * t * t +
        (-a[j] + 3 * b[j] - 3 * c[j] + d[j]) * t * t * t),
  );
}

export function flightPose(
  i: number,
  p: number,
  t: number,
  w: number,
  h: number,
  landing: { x: number; y: number },
) {
  const r = hash(i + 1),
    q = hash(i + 201),
    z = hash(i + 411);
  const delay = r * 0.025;
  const [gx, gy] = guide(clamp(p - delay));
  const phase = t * (0.18 + q * 0.14) + i * 2.399;
  const depth = 0.78 + (0.35 * (Math.sin(t * 0.31 + i * 1.7) + 1)) / 2;
  const spread = Math.min(w, h) * (0.055 + r * 0.2) * depth;
  const approach = smooth(0, 0.07, p);
  const landingMix = smooth(0.89 + r * 0.035, 0.977, p);
  const x =
    gx * w +
    Math.cos(phase) * spread +
    Math.sin(i * 2.3 + p * 18) * spread * 0.9;
  const y =
    gy * h +
    Math.sin(phase * 1.21) * spread * 0.67 +
    Math.cos(i * 1.7 + p * 15) * spread * 0.5 +
    Math.sin(t * 1.4 + i) * 3;
  const glide = smooth(0.6, 0.9, Math.sin(t * 0.63 + i * 0.81));
  return {
    x: mix(x, landing.x, landingMix),
    y: mix(y, landing.y, landingMix),
    size: mix((13 + q * 30) * (w < 600 ? 0.7 : 1) * depth, 2, landingMix),
    angle: mix(
      Math.sin(phase * 0.7) * 0.95 + Math.sin(i + p * 13) * 0.3,
      0.1,
      landingMix,
    ),
    flap: t * (3.2 + z * 1.6) * Math.PI * 2 + i * 2.3,
    glide,
    alpha:
      (0.65 + r * 0.35) * (1 - smooth(0.972, 0.997, p)) * mix(0.7, 1, approach),
    white: i % 4 === 0,
  };
}

export function createWorld(
  canvas: HTMLCanvasElement,
  assets: Record<string, HTMLImageElement>,
) {
  const ctx = context(canvas, false);
  let w = 1,
    h = 1,
    dpr = 1;
  const temp = (width: number, height: number) => {
    const c = document.createElement("canvas");
    c.width = width;
    c.height = height;
    return c;
  };
  const sprite = temp(384, 256);
  const sc = context(sprite);
  sc.imageSmoothingQuality = "high";
  sc.filter = "brightness(1.15)";
  sc.drawImage(assets.butterfly, 0, 0, 384, 256);
  const white = temp(384, 256);
  const wc = context(white);
  wc.filter = "saturate(.15) brightness(1.35)";
  wc.drawImage(sprite, 0, 0);
  const glow = temp(128, 128),
    gc = context(glow);
  const halo = gc.createRadialGradient(64, 64, 0, 64, 64, 64);
  halo.addColorStop(0, "rgba(245,221,255,.68)");
  halo.addColorStop(0.15, "rgba(207,156,255,.38)");
  halo.addColorStop(0.42, "rgba(160,93,247,.12)");
  halo.addColorStop(1, "rgba(133,71,230,0)");
  gc.fillStyle = halo;
  gc.fillRect(0, 0, 128, 128);

  const roomFrame = temp(1920, 1080),
    rc = context(roomFrame);
  const returnFrame = temp(1920, 1080),
    hc = context(returnFrame);
  for (const [target, img] of [
    [rc, assets.room],
    [hc, assets.homecoming],
  ] as const) {
    target.drawImage(img, 0, 0, 1920, 1080);
    target.globalCompositeOperation = "destination-out";
    target.beginPath();
    target.moveTo(916, -10);
    target.lineTo(1630, -10);
    target.lineTo(1631, 613);
    target.lineTo(921, 593);
    target.closePath();
    target.fill();
  }
  const galleryFrame = temp(1920, 1080),
    ac = context(galleryFrame);
  ac.drawImage(assets.gallery, 0, 0, 1920, 1080);
  ac.globalCompositeOperation = "destination-out";
  ac.beginPath();
  ac.moveTo(1690, -10);
  ac.lineTo(1850, -10);
  ac.lineTo(1835, 645);
  ac.lineTo(1545, 630);
  ac.lineTo(1552, 405);
  ac.quadraticCurveTo(1570, 195, 1690, -10);
  ac.closePath();
  ac.fill();
  const mark = temp(440, 300),
    mc = context(mark);
  mc.drawImage(assets.mark, 0, 0, 440, 300);
  const pixels = mc.getImageData(0, 0, 440, 300).data;
  const landings: { x: number; y: number }[] = [];
  for (let i = 0; landings.length < 90 && i < 20000; i++) {
    const x = Math.floor(hash(i + 712) * 440),
      y = Math.floor(hash(i + 1213) * 300);
    if (pixels[(y * 440 + x) * 4 + 3] > 180)
      landings.push({ x: x / 440 - 0.5, y: y / 300 - 0.5 });
  }
  mc.globalCompositeOperation = "source-in";
  mc.fillStyle = "#704098";
  mc.fillRect(0, 0, 440, 300);
  const bridge = temp(1920, 1080),
    bc = context(bridge);
  bc.beginPath();
  bc.moveTo(0, 790);
  bc.lineTo(270, 813);
  bc.lineTo(620, 858);
  bc.lineTo(906, 889);
  bc.lineTo(1234, 939);
  bc.lineTo(1640, 999);
  bc.lineTo(1920, 1060);
  bc.lineTo(1920, 1080);
  bc.lineTo(0, 1080);
  bc.closePath();
  bc.clip();
  bc.drawImage(assets.kingdom, 0, 0, 1920, 1080);
  function resize() {
    w = canvas.clientWidth;
    h = canvas.clientHeight;
    dpr = Math.min(devicePixelRatio, 1.6);
    canvas.width = Math.round(w * dpr);
    canvas.height = Math.round(h * dpr);
  }
  function imageTransform(scale: number, fx = 0.5, fy = 0.5) {
    const s = Math.max(w / 1920, h / 1080) * scale;
    return {
      s,
      x: clamp(w / 2 - fx * 1920 * s, w - 1920 * s, 0),
      y: clamp(h / 2 - fy * 1080 * s, h - 1080 * s, 0),
    };
  }
  function scene(
    image: CanvasImageSource,
    scale = 1,
    fx = 0.5,
    fy = 0.5,
    alpha = 1,
  ) {
    const v = imageTransform(scale, fx, fy);
    ctx.save();
    ctx.globalAlpha = alpha;
    ctx.drawImage(image, v.x, v.y, 1920 * v.s, 1080 * v.s);
    ctx.restore();
    return v;
  }
  function mist(t: number, p: number) {
    ctx.save();
    ctx.globalCompositeOperation = "screen";
    for (let i = 0; i < 5; i++) {
      const x = ((hash(i + 12) * w + t * (3 + i)) % (w * 1.6)) - w * 0.3;
      const y =
        h * (0.48 + hash(i + 20) * 0.42) + Math.sin(p * 9 + i) * h * 0.06;
      const radius = w * (0.24 + hash(i + 19) * 0.17);
      const g = ctx.createRadialGradient(x, y, 0, x, y, radius);
      g.addColorStop(0, "rgba(182,156,216,.034)");
      g.addColorStop(1, "rgba(137,113,172,0)");
      ctx.fillStyle = g;
      ctx.fillRect(x - radius, y - radius, radius * 2, radius * 2);
    }
    ctx.restore();
  }
  function gateway(p: number) {
    const enter = smooth(0.25, 0.5, p),
      through = smooth(0.365, 0.605, p);
    const s = Math.max(w / 1536, h / 1024) * (0.82 / (1 - through * 0.91));
    const cx = mix(w * 1.48, w * 0.51, enter);
    const x = cx - 768 * s,
      y = h * 1.1 - 1024 * s;
    ctx.save();
    ctx.beginPath();
    ctx.moveTo(x + 517 * s, y + 1024 * s);
    ctx.lineTo(x + 517 * s, y + 432 * s);
    ctx.bezierCurveTo(
      x + 525 * s,
      y + 288 * s,
      x + 645 * s,
      y + 198 * s,
      x + 768 * s,
      y + 169 * s,
    );
    ctx.bezierCurveTo(
      x + 924 * s,
      y + 220 * s,
      x + 1000 * s,
      y + 306 * s,
      x + 1019 * s,
      y + 432 * s,
    );
    ctx.lineTo(x + 1019 * s, y + 1024 * s);
    ctx.closePath();
    ctx.clip();
    scene(assets.city, cityScale(p), cityFocus(p), 0.48);
    ctx.restore();
    ctx.drawImage(assets.gateway, x, y, 1536 * s, 1024 * s);
  }
  const cityScale = (p: number) => mix(1.17, 1.34, smooth(0.48, 0.79, p));
  const cityFocus = (p: number) => mix(0.59, 0.63, smooth(0.48, 0.79, p));
  function returnPose(p: number) {
    const pull = smooth(0.76, 0.885, p),
      settle = smooth(0.885, 0.98, p);
    return {
      scale:
        mix(5.5, w < 600 ? 1.02 : 1.07, pull) + settle * (w < 600 ? 0 : 0.3),
      fx:
        mix(0.665, w < 600 ? 0.65 : 0.52, pull) +
        settle * (w < 600 ? 0 : 0.035),
      fy: mix(0.265, 0.49, pull) + settle * 0.07,
    };
  }
  function markPose(p: number) {
    const r = returnPose(p),
      v = imageTransform(r.scale, r.fx, r.fy);
    return {
      x: v.x + 1410 * v.s,
      y: v.y + 869 * v.s,
      width: 80 * v.s,
      height: 55 * v.s,
    };
  }
  function butterfly(b: ReturnType<typeof flightPose>) {
    if (b.alpha < 0.002) return;
    const image = b.white ? white : sprite;
    ctx.save();
    ctx.translate(b.x, b.y);
    ctx.rotate(b.angle);
    ctx.globalCompositeOperation = "screen";
    ctx.globalAlpha = b.alpha * 0.86;
    ctx.drawImage(
      glow,
      -b.size * 1.4,
      -b.size * 1.4,
      b.size * 2.8,
      b.size * 2.8,
    );
    ctx.globalAlpha = b.alpha;
    const fold = mix(
      0.12 + 0.88 * ((Math.sin(b.flap) + 1) / 2) ** 0.72,
      0.8,
      b.glide,
    );
    const right = mix(
      0.12 + 0.88 * ((Math.sin(b.flap + 0.11) + 1) / 2) ** 0.72,
      0.68,
      b.glide,
    );
    const sh = (b.size * 2) / 3;
    ctx.save();
    ctx.transform(fold, Math.cos(b.flap) * 0.13, 0, 1, 0, 0);
    ctx.drawImage(
      image,
      0,
      0,
      189,
      256,
      -b.size / 2,
      -sh * 0.51,
      b.size * 0.493,
      sh,
    );
    ctx.restore();
    ctx.save();
    ctx.transform(right, -Math.cos(b.flap + 0.16) * 0.13, 0, 1, 0, 0);
    ctx.drawImage(
      image,
      195,
      0,
      189,
      256,
      b.size * 0.007,
      -sh * 0.51,
      b.size * 0.493,
      sh,
    );
    ctx.restore();
    ctx.globalCompositeOperation = "source-over";
    ctx.drawImage(
      image,
      187,
      48,
      10,
      149,
      -b.size * 0.013,
      -sh * 0.322,
      b.size * 0.026,
      sh * 0.582,
    );
    ctx.restore();
  }
  function render(progress: number, t: number, arrival = 1) {
    if (w === 0 || h === 0) return;
    // Hold the camera inside the gallery while its works are in view.
    const p =
      progress < 0.62
        ? (progress * 0.755) / 0.62
        : progress < 0.84
          ? 0.755
          : 0.755 + ((progress - 0.84) * 0.245) / 0.16;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.imageSmoothingQuality = "high";
    ctx.fillStyle = "#100e16";
    ctx.fillRect(0, 0, w, h);
    const outside = smooth(0.06, 0.255, p);
    if (p < 0.605) {
      scene(
        assets.distance,
        mix(1.02, 1.14, smooth(0.16, 0.59, p)),
        mix(0.5, 0.59, smooth(0.255, 0.59, p)),
        0.5,
      );
      scene(
        bridge,
        mix(1.03, 2.8, smooth(0.25, 0.605, p)),
        mix(0.5, 0.67, smooth(0.255, 0.59, p)),
        mix(0.5, 0.62, smooth(0.255, 0.59, p)),
      );
      if (p < 0.26)
        scene(
          roomFrame,
          mix(1.025, 5.5, outside),
          mix(w < 600 ? 0.64 : 0.5, 0.665, outside),
          mix(0.5, 0.265, outside),
        );
      if (p > 0.25) gateway(p);
    } else scene(assets.city, cityScale(p), cityFocus(p), 0.48);
    if (progress > 0.605) {
      const reveal = smooth(0.605, 0.692, progress);
      scene(
        galleryFrame,
        mix(10, 1.025, reveal),
        mix(0.88, 0.5, reveal),
        mix(0.295, 0.5, reveal),
      );
    }
    if (p > 0.755) {
      const r = returnPose(p);
      scene(assets.kingdom, 1.15, 0.56, 0.46, smooth(0.82, 0.875, p));
      scene(returnFrame, r.scale, r.fx, r.fy);
    }
    mist(t, p);
    const landing = markPose(p);
    const count = w < 600 ? 42 : 64;
    for (let i = 0; i < count; i++) {
      const dot = landings[i % landings.length];
      const target = {
        x: landing.x + (dot.x - dot.y * 0.23) * landing.width,
        y: landing.y + (dot.y * 0.68 + dot.x * 0.037) * landing.height,
      };
      const pose = flightPose(i, p, t, w, h, target);
      pose.alpha *= arrival;
      const emerge = smooth(0, 2.8, t - hash(i + 319) * 1.1);
      pose.x = mix(w * 0.67, pose.x, emerge);
      pose.y = mix(h * 0.3, pose.y, emerge);
      pose.size *= mix(0.35, 1, emerge);
      if (progress > 0.6 && progress < 0.87) {
        const settle =
          smooth(0.6, 0.68, progress) * (1 - smooth(0.81, 0.87, progress));
        const theta = i * 2.399 + t * 0.14;
        pose.x = mix(pose.x, w * (0.5 + 0.37 * Math.cos(theta)), settle);
        pose.y = mix(pose.y, h * (0.44 + 0.32 * Math.sin(theta)), settle);
        pose.alpha *= 1 - settle * 0.38;
        pose.size *= 1 - settle * 0.28;
      }
      butterfly(pose);
    }
    ctx.save();
    ctx.globalCompositeOperation = "screen";
    for (let i = 0; i < 42; i++) {
      const [gx, gy] = guide(clamp(p - hash(i + 58) * 0.035));
      const x = gx * w + Math.sin(t * 0.17 + i * 2.4) * w * 0.14,
        y = gy * h + Math.cos(t * 0.23 + i * 1.6) * h * 0.21;
      ctx.globalAlpha =
        (0.12 + hash(i + 79) * 0.36) * (1 - smooth(0.9, 0.98, p)) * arrival;
      ctx.fillStyle = i % 3 ? "#c5a2ee" : "#fff5ff";
      ctx.beginPath();
      ctx.arc(x, y, 0.4 + hash(i + 49) * 0.7, 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.restore();
    if (p > 0.955) {
      ctx.save();
      ctx.globalAlpha = smooth(0.955, 0.993, p);
      ctx.translate(landing.x, landing.y);
      ctx.transform(1, 0.037, -0.23, 0.68, 0, 0);
      ctx.shadowColor = "#cc99ff";
      ctx.shadowBlur = 12 * (1 - smooth(0.965, 1, p));
      ctx.globalCompositeOperation = "multiply";
      ctx.drawImage(
        mark,
        -landing.width / 2,
        -landing.height / 2,
        landing.width,
        landing.height,
      );
      ctx.restore();
    }
  }
  resize();
  return { resize, render, markPose };
}
