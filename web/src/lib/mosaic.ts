export type MosaicPicture = {
  height: number;
  id: string;
  name?: string;
  src: string;
  width: number;
};

export type MosaicRow = {
  height: number;
  justified: boolean;
  pictures: MosaicPicture[];
};

export const MOSAIC_GAP_PX = 8;

/** How far a short last row may stretch past the height the creator chose. */
const SHORT_ROW_CEILING = 1.35;

/** mosaicRows packs pictures into rows that fill the width at their own shapes. */
export function mosaicRows(
  pictures: readonly MosaicPicture[],
  { rowHeight, width }: { rowHeight: number; width: number },
): MosaicRow[] {
  if (pictures.length === 0) return [];
  if (width <= 0) {
    return [{ height: rowHeight, justified: false, pictures: [...pictures] }];
  }

  const rows: MosaicRow[] = [];
  let run: MosaicPicture[] = [];
  let ratios = 0;

  for (const picture of pictures) {
    run.push(picture);
    ratios += aspect(picture);
    if (rowFits(run.length, ratios, width) <= rowHeight) {
      rows.push({
        height: rowFits(run.length, ratios, width),
        justified: true,
        pictures: run,
      });
      run = [];
      ratios = 0;
    }
  }

  if (run.length > 0) {
    const stretched = rowFits(run.length, ratios, width);
    const justified = stretched <= rowHeight * SHORT_ROW_CEILING;
    rows.push({
      height: justified ? stretched : rowHeight,
      justified,
      pictures: run,
    });
  }

  return rows;
}

export function pictureWidth(picture: MosaicPicture, height: number): number {
  return aspect(picture) * height;
}

function rowFits(count: number, ratios: number, width: number): number {
  return (width - MOSAIC_GAP_PX * (count - 1)) / ratios;
}

function aspect(picture: MosaicPicture): number {
  if (picture.width <= 0 || picture.height <= 0) return 1;
  return picture.width / picture.height;
}
