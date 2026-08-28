"""Cut the generated engraving sheet into one tinted badge mark per cell."""

from pathlib import Path

from PIL import Image

BADGES = Path(__file__).parent.parent / "src" / "assets" / "art" / "badges"
SHEET = BADGES / "source" / "illarin-badge-glyphs-v1.png"
OUT = BADGES
CELL = 512
COLUMNS = 4
EDGE = 512
MARGIN = 0.07

# Three stops per metal: deepest recess, midtone, struck highlight.
METALS = {
    "iron": ((0x1C, 0x1F, 0x22), (0x6B, 0x71, 0x77), (0xC9, 0xCE, 0xD3)),
    "silver": ((0x31, 0x35, 0x3A), (0xA6, 0xAB, 0xB1), (0xF6, 0xF7, 0xF8)),
    "gold": ((0x38, 0x28, 0x0F), (0xB6, 0x87, 0x31), (0xF7, 0xE6, 0xB4)),
}

MARKS = [
    ("first-light", "iron"),
    ("tenfold", "silver"),
    ("centenary", "gold"),
    ("long-service", "silver"),
    ("field-report", "iron"),
    ("quiet-fix", "silver"),
    ("open-workshop", "iron"),
    ("verified-app-contributor", "gold"),
]


def ramp(metal):
    """Build one 256-entry lookup per channel from a metal's three stops."""
    low, mid, high = METALS[metal]
    table = []
    for channel in range(3):
        entries = []
        for value in range(256):
            if value < 128:
                weight = value / 127
                start, end = low[channel], mid[channel]
            else:
                weight = (value - 128) / 127
                start, end = mid[channel], high[channel]
            entries.append(round(start + (end - start) * weight))
        table.append(entries)
    return table


def cut(sheet, index):
    """Take one cell out of the sheet as greyscale."""
    left = (index % COLUMNS) * CELL
    top = (index // COLUMNS) * CELL
    return sheet.crop((left, top, left + CELL, top + CELL))


def alpha_of(cell):
    """Read the struck relief off the black ground it was generated on."""
    return cell.point(lambda value: min(255, max(0, (value - 8) * 6)))


def square(image, box):
    """Centre one glyph on a square field with the same margin as its siblings."""
    cropped = image.crop(box)
    side = round(max(cropped.size) / (1 - MARGIN * 2))
    field = Image.new(image.mode, (side, side), 0)
    field.paste(
        cropped,
        ((side - cropped.width) // 2, (side - cropped.height) // 2),
    )
    return field.resize((EDGE, EDGE), Image.LANCZOS)


def main():
    sheet = Image.open(SHEET).convert("L")
    for index, (name, metal) in enumerate(MARKS):
        cell = cut(sheet, index)
        alpha = alpha_of(cell)
        box = alpha.getbbox()
        relief = square(cell, box)
        table = ramp(metal)
        channels = [relief.point(entries) for entries in table]
        mark = Image.merge("RGB", channels)
        mark.putalpha(square(alpha, box))
        mark.save(OUT / f"illarin-badge-{name}-v1.png")
        print(name, metal, mark.size)


main()
