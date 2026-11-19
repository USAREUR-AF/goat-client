# Figures

`make-figures.py` draws the figure at the top of the root README,
`goat-client`, in a light and a dark version for a `<picture>` element. Every
mode, component and platform in it is checked against the root README, and the
build stops if one of them is no longer stated there.

```
python3 -I docs/figures/make-figures.py           # write the SVG files
python3 -I docs/figures/make-figures.py --check   # exit 1 if a committed SVG is out of date
```

Standard library only. The SVG files are self-contained: no external fonts,
styles or images.
