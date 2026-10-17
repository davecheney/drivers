"""Put a radial starburst behind the transparent Tainigo logo and encode 2bpp."""

from io import BytesIO
from pathlib import Path
import runpy
import xml.etree.ElementTree as ET

import resvg_py


HERE = Path(__file__).resolve().parent
SVG = "http://www.w3.org/2000/svg"
INKSCAPE = "http://www.inkscape.org/namespaces/inkscape"
XLINK = "http://www.w3.org/1999/xlink"
for prefix, namespace in (
    ("", SVG),
    ("inkscape", INKSCAPE),
    ("sodipodi", "http://sodipodi.sourceforge.net/DTD/sodipodi-0.dtd"),
    ("xlink", XLINK),
):
    ET.register_namespace(prefix, namespace)

convert_four = runpy.run_path(
    str(HERE.parent / "grayscale-slideshow/convert.py")
)["convert"]


def compose(source):
    root = ET.parse(source).getroot()
    if root.tag != f"{{{SVG}}}svg" or root.get("viewBox") != "0 0 60 33":
        raise ValueError("expected the 60x33 transparent Tainigo SVG")
    root.set("width", "296")
    root.set("height", "128")
    root.set("viewBox", "-8.15625 0 76.3125 33")
    for path in root.iter(f"{{{SVG}}}path"):
        style = path.get("style", "")
        path.set("style", style.replace("fill:#00459c", "fill:#000000"))

    defs = root.find(f"{{{SVG}}}defs")
    if defs is None:
        raise ValueError("Tainigo SVG is missing its defs element")
    ET.SubElement(defs, f"{{{SVG}}}path", {
        "id": "sunburst-ray",
        "d": "M 0,0 100,-13.16525 100,13.16525 Z",
    })
    burst = ET.Element(f"{{{SVG}}}g", {
        "id": "sunburst",
        f"{{{INKSCAPE}}}groupmode": "layer",
        f"{{{INKSCAPE}}}label": "sunburst",
    })
    ET.SubElement(burst, f"{{{SVG}}}rect", {
        "x": "-8.15625", "y": "0", "width": "76.3125", "height": "33",
        "fill": "#ffffff",
    })
    rays = ET.SubElement(burst, f"{{{SVG}}}g", {
        "transform": "translate(30,16.5)", "fill": "#aaaaaa",
    })
    for angle in range(0, 360, 30):
        ET.SubElement(rays, f"{{{SVG}}}use", {
            f"{{{XLINK}}}href": "#sunburst-ray", "transform": f"rotate({angle})",
        })
    root.insert(list(root).index(defs) + 1, burst)
    ET.indent(root, space="  ")
    return ET.tostring(root, encoding="unicode", xml_declaration=True) + "\n"


def main():
    assets = HERE / "assets"
    svg = compose(assets / "tainigo_nobleed_nobg.svg")
    png = resvg_py.svg_to_bytes(
        svg_string=svg, width=1184, height=512, dpi=96, background="white",
    )
    data = convert_four(BytesIO(png), 1.0)
    (assets / "tainigo.svg").write_text(svg, encoding="utf-8")
    (assets / "tainigo.bin").write_bytes(data)
    print(f"tainigo.bin: {len(data)} bytes, four shades, Floyd-Steinberg")


if __name__ == "__main__":
    main()
