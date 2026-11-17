#!/usr/bin/env python3
"""Pixel art for 虎兔大冒险 (frontend/src/games/hutu).

Every sprite is drawn here as rows of palette letters. Running the script checks the row widths,
writes frontend/src/games/hutu/spriteData.ts and, with --preview out.png, renders a contact sheet
(heroes are composed from a head and a body the same way the game does).

    python3 tools/hutu_sprites.py --preview /tmp/hutu-sheet.png
"""

import json
import os
import sys

# shared colours; hero-specific letters (h H k m c C) come from HERO_PALETTES
PALETTE = {
    "o": "#2b1b2e",  # outline / eyes
    "s": "#ffd7b0",  # skin
    "S": "#eaa77e",  # skin shade
    "r": "#ff8a8a",  # blush
    "w": "#fff8ec",  # white
    "W": "#d9cbbd",  # white shade
    "y": "#ffd23f",  # gold trim
    "Y": "#e09a1a",  # gold shade
    "p": "#33427a",  # trousers
    "P": "#232d57",
    "b": "#7a4330",  # shoes
    "g": "#5fbf3a",  # leaf green
    "G": "#2f7d2a",
    "n": "#9a5a32",  # mud brown
    "N": "#6b3a22",
    "l": "#c98a52",  # light brown
    "R": "#e8322e",  # red
    "q": "#a01e2a",  # dark red
    "f": "#ff9d2e",  # flame orange
    "F": "#fff07a",  # flame yellow
    "u": "#8fd3ff",  # light blue
    "U": "#3a7fd0",
    "v": "#bfa7ff",  # lavender
    "x": "#8a8f9c",  # grey
    "X": "#5a5f6e",
    "z": "#cfd4dc",  # light grey
    "t": "#3a2418",  # tiger stripe / dark
    "i": "#ffb7cc",  # pink
    "j": "#2a6d3a",  # bamboo dark
    "J": "#7ed957",  # bamboo light
    "a": "#c2e85a",  # bamboo highlight
}

HERO_PALETTES = {
    "tiger": {"h": "#ff9a2e", "H": "#d4621a", "k": "#3a2418", "m": "#fff8ec", "c": "#e8443a", "C": "#a82a32"},
    "rabbit": {"h": "#f4f0ff", "H": "#bfb3dc", "k": "#ff9fc0", "m": "#ff9fc0", "c": "#3aa0e8", "C": "#2462ad"},
}
# 辣椒 power: the outfit turns into a flame suit
CHILI = {"c": "#ffcf3a", "C": "#e3711a", "y": "#e8282a", "Y": "#a01e2a"}

S = {}


def spr(name, rows):
    w = len(rows[0])
    for i, r in enumerate(rows):
        if len(r) != w:
            raise SystemExit(f"{name}: row {i} is {len(r)} wide, expected {w}: {r!r}")
        for ch in r:
            if ch != "." and ch not in PALETTE and ch not in "hHkmcC":
                raise SystemExit(f"{name}: unknown colour {ch!r}")
    S[name] = rows


# ─────────────────────────── heroes ───────────────────────────
# heads are 16×10; the rabbit's ears add 6 rows above (drawn as a separate piece)

spr("head_tiger", [
    "..oo........oo..",
    ".okho......ohko.",
    ".ohhoooooooohho.",
    "ohhkhhhkkhhhkhho",
    "ohkhhhhkkhhhhkho",
    "oHhhhhhhhhhhhhHo",
    "ommmmmmmmmmmmmmo",
    "oHssosssssssosHo",
    "oHsrossssssorsHo",
    ".oHssssSSssssHo.",
])

spr("head_rabbit", [
    "..okho....ohko..",
    "..ohho....ohho..",
    ".ohhhoooooohhho.",
    "ohhhhhhhhhhhhhho",
    "ohhhhhhkkhhhhhho",
    "oHhhhhhhhhhhhhHo",
    "ommmmmmmmmmmmmmo",
    "oHssosssssssosHo",
    "oHsrossssssorsHo",
    ".oHssssSSssssHo.",
])

spr("ears_rabbit", [
    "...oo......oo...",
    "..ohho....ohho..",
    "..ohko....okho..",
    "..ohko....okho..",
    "..ohko....okho..",
    "..ohko....okho..",
])

# small bodies: 16×6
spr("sb_stand", [
    "...ocyccccyco...",
    "..osccccccccso..",
    "..oSCyyyyyyCSo..",
    "...oppppppppo...",
    "..obbbo..obbbo..",
    "..ooooo..ooooo..",
])
spr("sb_walk1", [
    "...ocyccccyco...",
    "..osccccccccso..",
    "...oCyyyyyyCSo..",
    "...oppppppppo...",
    ".obbbo....obbbo.",
    ".oooo......oooo.",
])
spr("sb_walk2", [
    "...ocyccccyco...",
    "...osccccccso...",
    "...oCyyyyyyCo...",
    "....opppppppo...",
    "....obbbbbbo....",
    "....oooooooo....",
])
spr("sb_walk3", [
    "...ocyccccyco...",
    "..osccccccccso..",
    "..oSCyyyyyyCo...",
    "...oppppppppo...",
    "..obbbo..obbo...",
    "..oooo....ooo...",
])
spr("sb_jump", [
    ".osocyccccycoso.",
    "..ocsccccccsco..",
    "...oCyyyyyyCo...",
    "...oppppppppo...",
    "..obbbo.obbbbo..",
    "..oooo...oooo...",
])
spr("sb_skid", [
    "..socyccccyco...",
    "...occcccccsso..",
    "...oCyyyyyyCo...",
    "...opppppppppo..",
    ".obbbo...obbbo..",
    ".oooo.....ooo...",
])
spr("sb_die", [
    ".so.ocyyyyco.os.",
    ".soocccccccooso.",
    "...oCyyyyyyCo...",
    "...oppppppppo...",
    "...obbo..obbo...",
    "...oooo..oooo...",
])
spr("sb_pole", [
    "...ocyccccycoso.",
    "..osccccccccso..",
    "..oSCyyyyyyCo...",
    "...oppppppppo...",
    "....obbbobbbo...",
    "....ooooooooo...",
])

# big bodies: 16×16 (the head sits on top, so a big hero is 26 tall)
BIG_TOP = [
    "...occyccyycco..",
    "..occccyycccco..",
    ".osccccyyccccso.",
    ".oscCccyycccCso.",
    ".osCcccyyccccso.",
    ".osocccyycccoso.",
    "..ooCccyyccCoo..",
    "...oyyyyyyyyo...",
    "...oYYYYYYYYo...",
]
spr("bb_stand", BIG_TOP + [
    "...opppppppppo..",
    "...oppppoppppo..",
    "...opppo.opppo..",
    "...oPPPo.oPPPo..",
    "..obbbbo.obbbbo.",
    "..obbbbo.obbbbo.",
    "..oooooo.oooooo.",
])
spr("bb_walk1", [
    "...occyccyycco..",
    "..occccyycccco..",
    "..osccccyycccsso",
    "..oscCccyyccCso.",
    "..osCcccyyccco..",
    "..osocccyyccco..",
    "...ooCccyyccCo..",
    "...oyyyyyyyyo...",
    "...oYYYYYYYYo...",
    "..opppppppppppo.",
    ".opppppo.opppppo",
    ".oPPPPo...oPPPPo",
    "obbbbo.....obbbo",
    "obbbbo.....obbbo",
    "oooooo.....ooooo",
    "................",
])
spr("bb_walk2", BIG_TOP + [
    "...opppppppppo..",
    "....opppppppo...",
    "....oppppppo....",
    "....oPPPPPPo....",
    "....obbbbbbbo...",
    "....obbbbbbbo...",
    "....ooooooooo...",
])
spr("bb_walk3", [
    "...occyccyycco..",
    "..occccyycccco..",
    "ossccccyyccccso.",
    ".oscCccyycccCso.",
    "..occcccyycccCso",
    "..occcccyyccccso",
    "...oCcccyyccooo.",
    "...oyyyyyyyyo...",
    "...oYYYYYYYYo...",
    "..oppppppppppo..",
    "..opppppoppppo..",
    "..oPPPPo.oPPPo..",
    ".obbbbo..obbbbo.",
    ".obbbbo..obbbbo.",
    ".oooooo..oooooo.",
    "................",
])
spr("bb_jump", [
    "osocyccyycco.oso",
    "oscccccyycccccso",
    ".occcccyyccccco.",
    "..ocCccyycccCo..",
    "..oCcccyyccccCo.",
    "...occcyyccco...",
    "...oCccyyccCo...",
    "...oyyyyyyyyo...",
    "...oYYYYYYYYo...",
    "..opppppppppppo.",
    "..oppppp.opppppo",
    "..oPPPPo..oPPPPo",
    ".obbbbo...obbbbo",
    ".obbbbo....oooo.",
    ".oooooo.........",
    "................",
])
spr("bb_skid", [
    "...occyccyycco..",
    "..occccyyccccso.",
    "..occcccyyccccso",
    ".soocCccyycccCo.",
    ".socccccyyccccso",
    "..ooocccyyccccso",
    "...oCcccyyccCoo.",
    "...oyyyyyyyyo...",
    "...oYYYYYYYYo...",
    "...oppppppppppo.",
    "..opppppo.ppppo.",
    ".oPPPPo...oPPPo.",
    "obbbbo....obbbbo",
    "obbbbo....obbbbo",
    "oooooo....oooooo",
    "................",
])
spr("bb_crouch", [
    "..occcyccyyccco.",
    ".osccccyycccccso",
    ".oscCccyycccCcso",
    ".ooyyyyyyyyyyyoo",
    ".opppppppppppppo",
    "obbbbbo..obbbbbo",
    "oooooooooooooooo",
])
spr("bb_pole", BIG_TOP[:5] + [
    ".osocccyycccosso",
    "..ooCccyyccCoo..",
    "...oyyyyyyyyo...",
    "...oYYYYYYYYo...",
    "...opppppppppo..",
    "....oppppoppppo.",
    "....oppo..oppo..",
    "....oPPo..oPPo..",
    "....obbbobbbbo..",
    "....obbbobbbbo..",
    "....ooooooooo...",
])

# ─────────────────────────── enemies ───────────────────────────

spr("mud1", [
    "......ogo.......",
    ".....ogGgo......",
    "......ogo.......",
    "....oonnnoo.....",
    "...onnnnnnnoo...",
    "..onnlnnnnnnno..",
    ".onnlnnnnnnnnno.",
    ".onnwwonnwwonno.",
    ".onnwoonnwoonno.",
    ".oNnnnnnnnnnnNo.",
    ".oNNnnnooonnNNo.",
    "..oNNNNNNNNNNo..",
    "..obbo....obbo..",
    "...oo......oo...",
])
spr("mud2", [
    "........ogo.....",
    ".......ogGgo....",
    "........ogo.....",
    "....oonnnoo.....",
    "...onnnnnnnoo...",
    "..onnlnnnnnnno..",
    ".onnlnnnnnnnnno.",
    ".onnwwonnwwonno.",
    ".onnwoonnwoonno.",
    ".oNnnnnnnnnnnNo.",
    ".oNNnnnooonnNNo.",
    "..oNNNNNNNNNNo..",
    "...obbo..obbo...",
    "....oo....oo....",
])
spr("mud_flat", [
    "......ogo.......",
    "....oonnnnoo....",
    "..onnwwonnwwonn.",
    ".oNNnnnnnnnnNNo.",
    "..oooooooooooo..",
])

spr("hog1", [
    ".....o.o.o......",
    "...ooxoxoxo.....",
    "..oxXxXxXxXo....",
    ".oXxXxXxXxXxo...",
    ".oxXxXxXxXxXxo..",
    "oXxXxXxXxXxXllo.",
    "oxXxXxXxXxXlwolo",
    "oXxXxXxXxXllllNo",
    ".ollllllllllllo.",
    "..obbo....obbo..",
    "...oo......oo...",
])
spr("hog2", [
    ".....o.o.o......",
    "...ooxoxoxo.....",
    "..oxXxXxXxXo....",
    ".oXxXxXxXxXxo...",
    ".oxXxXxXxXxXxo..",
    "oXxXxXxXxXxXllo.",
    "oxXxXxXxXxXlwolo",
    "oXxXxXxXxXllllNo",
    ".ollllllllllllo.",
    "...obbo..obbo...",
    "....oo....oo....",
])
spr("hog_ball", [
    "....o.o.o.o.....",
    "..o.oxoxoxo.o...",
    "...oxXxXxXxXo...",
    ".ooXxXxXxXxXxoo.",
    "..oxXxXxXxXxXo..",
    ".oXxXxXxXxXxXxo.",
    "..oxXxXxXxXxXo..",
    ".ooXxXxXxXxXxoo.",
    "...oxXxXxXxXo...",
    "..o.oxoxoxo.o...",
    "....o.o.o.o.....",
])

spr("kite1", [
    ".......oo.......",
    "......oRRo......",
    ".....oRwwRo.....",
    "....oRwoowRo....",
    "...oRRwwwwRRo...",
    "..oRRRRRRRRRRo..",
    ".oyyRRRRRRRRyyo.",
    "..oRRRRRRRRRRo..",
    "...oRRqqqqRRo...",
    "....oRqqqqRo....",
    ".....oRqqRo.....",
    "......oRRo......",
    ".......oo.......",
    ".......o........",
    "......o.y.......",
    ".......o..y.....",
])
spr("kite2", [
    ".......oo.......",
    "......oRRo......",
    ".....oRwwRo.....",
    "....oRwoowRo....",
    "...oRRwwwwRRo...",
    "..oRRRRRRRRRRo..",
    ".oyyRRRRRRRRyyo.",
    "..oRRRRRRRRRRo..",
    "...oRRqqqqRRo...",
    "....oRqqqqRo....",
    ".....oRqqRo.....",
    "......oRRo......",
    ".......oo.......",
    "........o.......",
    ".......y.o......",
    ".....y..o.......",
])

spr("rat", [
    "..oo......oo....",
    ".oiio....oiio...",
    ".oilloooollio...",
    "..ollllllllo....",
    ".olllllllllllo..",
    ".olwoollllwoolo.",
    ".ollooollloollo.",
    "olllllliilllllo.",
    "ollllwwoowwllllo",
    ".olllwwwwwwlllo.",
    ".ollllllllllllo.",
    "..olllwwwwlllo..",
    "..olllwwwwlllo..",
    "..olllwwwwlllo..",
    "..olllllllllo...",
    "..olllllllllo...",
])

spr("fish1", [
    "......ooo.......",
    ".....ofFfo......",
    "..oooofffoo..oo.",
    ".offfffffffooffo",
    "offwoffffffffffo",
    "offoofffffffffo.",
    "offfffffffffoffo",
    ".oFFFFFFFFFoo.oo",
    "..oooooooooo....",
])
spr("fish2", [
    "......ooo.......",
    ".....ofFfo....oo",
    "..oooofffoo..ofo",
    ".offfffffffooffo",
    "offwoffffffffffo",
    "offoofffffffffo.",
    "offfffffffffoo..",
    ".oFFFFFFFFFo....",
    "..oooooooooo....",
])

spr("cracker1", [
    "......o.........",
    ".....oyo........",
    "......o.........",
    "....oooooo......",
    "...oRRRRRRo.....",
    "...oyyyyyyo.....",
    "...oRRRRRRo.....",
    "...owoRRwoo.....",
    "...oooRRooo.....",
    "...oRRRRRRo.....",
    "...oRRooRRo.....",
    "...oyyyyyyo.....",
    "...oqqqqqqo.....",
    "...obbo.obbo....",
    "...ooo...ooo....",
])
spr("cracker2", [
    ".......o........",
    "......oyo.......",
    ".......o........",
    "....oooooo......",
    "...oRRRRRRo.....",
    "...oyyyyyyo.....",
    "...oRRRRRRo.....",
    "...owoRRwoo.....",
    "...oooRRooo.....",
    "...oRRRRRRo.....",
    "...oRRooRRo.....",
    "...oyyyyyyo.....",
    "...oqqqqqqo.....",
    "....obbobbo.....",
    "....ooooooo.....",
])
spr("cracker_lit", [
    "......F.........",
    ".....FfF........",
    "......o.........",
    "....oooooo......",
    "...oRRRRRRo.....",
    "...oyyyyyyo.....",
    "...oRRRRRRo.....",
    "...oooRRooo.....",
    "...oRwRRwRo.....",
    "...oRRRRRRo.....",
    "...oRooooRo.....",
    "...oyyyyyyo.....",
    "...oqqqqqqo.....",
    "...oooooooo.....",
])

spr("spark1", [
    "....oFFo....",
    "...oFwwFo...",
    "..oFwwwwFo..",
    ".ofFwwwwFfo.",
    ".ofFFwwFFfo.",
    ".offFFFFffo.",
    "..offffffo..",
    "...oRffRo...",
    "....oRRo....",
    ".....oo.....",
])
spr("spark2", [
    "....oFFo....",
    "...oFwwFo...",
    "..oFwwwwFo..",
    ".ofFwwwwFfo.",
    ".ofFFwwFFfo.",
    ".offFFFFffo.",
    "..offffffo..",
    "...offffo...",
    "...oRooRo...",
    "....o..o....",
])
spr("ember", [
    "..oo..",
    ".oFFo.",
    "oFwwFo",
    "oFwwFo",
    ".oFFo.",
    "..oo..",
])
spr("fire1", [
    "..ooo...",
    ".oFFfo..",
    "oFwwFfo.",
    "oFwwFfRo",
    "ofFFfRo.",
    ".offRo..",
    "..ooo...",
])
spr("fire2", [
    "...ooo..",
    "..ofFFo.",
    ".ofFwwFo",
    "oRfFwwFo",
    ".oRfFFfo",
    "..oRffo.",
    "...ooo..",
])
spr("mudball", [
    "..ooooo..",
    ".onnnnno.",
    "onnlnnnno",
    "onlnnnnNo",
    "onnnnnNNo",
    "onnnnNNNo",
    ".oNNNNNo.",
    "..ooooo..",
])
spr("flame", [
    "......oooo..",
    "...ooofFFfo.",
    ".oofFFFwwFfo",
    "offFFwwwwFfo",
    ".oofFFFwwFfo",
    "...ooofFFfo.",
    "......oooo..",
])

spr("mudking1", [
    "...........oo..oo...............",
    "..........oyyooyyo..............",
    ".........oyYyyyyYyo.............",
    ".........oyyyRRyyyo.............",
    "........ooyyyyyyyyoo............",
    "......oonnnnnnnnnnnnoo..........",
    ".....onnnnnnnnnnnnnnnnoo........",
    "....onnlllnnnnnnnnnnnnnno.......",
    "...onnlnnnnnnnnnnnnnnnnnno......",
    "..onnlnnnwwwwnnnnwwwwnnnnno.....",
    "..onnnnnwwwwoonnwwwwoonnnno.....",
    ".onnnnnnwwwooonnwwwooonnnnno....",
    ".onnnnnnnwwwwnnnnwwwwnnnnnnno...",
    ".onnnnnnnnnnnnnnnnnnnnnnnnnno...",
    "onnnnnnnnnnoooooooonnnnnnnnnno..",
    "onnnnnnnnnowwowwowwonnnnnnnnno..",
    "onnnnnnnnnoooooooooonnnnnnnnno..",
    "onNnnnnnnnnnnnnnnnnnnnnnnnnNno..",
    "onNNnnnnnnnnnnnnnnnnnnnnnnNNno..",
    ".oNNNnnnnnnnnnnnnnnnnnnnnNNNo...",
    ".oNNNNnnnnnnnnnnnnnnnnnnNNNNo...",
    "onnoNNNNnnnnnnnnnnnnnnNNNNonno..",
    "onnnooNNNNNNNNNNNNNNNNNNoonnno..",
    ".onnno.oNNNNNNNNNNNNNNo.onnno...",
    "..ooo...oooooooooooooo...ooo....",
    "........obbbbo....obbbbo........",
    ".......obbbbbbo..obbbbbbo.......",
    ".......oooooooo..oooooooo.......",
])
spr("mudking2", [
    "...........oo..oo...............",
    "..........oyyooyyo..............",
    ".........oyYyyyyYyo.............",
    ".........oyyyRRyyyo.............",
    "........ooyyyyyyyyoo............",
    "......oonnnnnnnnnnnnoo..........",
    ".....onnnnnnnnnnnnnnnnoo........",
    "....onnlllnnnnnnnnnnnnnno.......",
    "...onnlnnnnnnnnnnnnnnnnnno......",
    "..onnlnnnwwwwnnnnwwwwnnnnno.....",
    "..onnnnnwwoowwnnwwoowwnnnno.....",
    ".onnnnnnwwooowwnwwooowwnnnno....",
    ".onnnnnnnwwwwnnnnwwwwnnnnnnno...",
    ".onnnnnnnnnnnnnnnnnnnnnnnnnno...",
    "onnnnnnnnnnoooooooonnnnnnnnnno..",
    "onnnnnnnnnoRRRRRRRRonnnnnnnnno..",
    "onnnnnnnnnowwowwowwonnnnnnnnno..",
    "onNnnnnnnnnoooooooonnnnnnnnNno..",
    "onNNnnnnnnnnnnnnnnnnnnnnnnNNno..",
    ".oNNNnnnnnnnnnnnnnnnnnnnnNNNo...",
    ".oNNNNnnnnnnnnnnnnnnnnnnNNNNo...",
    "onnoNNNNnnnnnnnnnnnnnnNNNNonno..",
    "onnnooNNNNNNNNNNNNNNNNNNoonnno..",
    ".onnno.oNNNNNNNNNNNNNNo.onnno...",
    "..ooo...oooooooooooooo...ooo....",
    ".........obbbbo..obbbbo.........",
    "........obbbbbbooobbbbbo........",
    "........ooooooo..oooooooo.......",
])

spr("nian1", [
    "......oo..................oo............",
    ".....oyyo................oyyo...........",
    ".....oyYyo..............oyYyo...........",
    "......oyYyoo..........ooyYyo............",
    ".......oyyYyoooooooooooyYyyo............",
    "........ooRRRRRRRRRRRRRRRRoo............",
    "......ooRRRRRRRRRRRRRRRRRRRRoo..........",
    ".....oRRRyyyRRRRRRRRRRRRyyyRRRo.........",
    "....oRRRyywwyRRRRRRRRRRywwyyRRRo........",
    "...oRRRRywooyRRRRRRRRRRyoowyRRRRo.......",
    "...oRRRRyyooyRRRRyyRRRRyooyyRRRRo.......",
    "..oRRRRRRyyyRRRRyyyyRRRRyyyRRRRRRo......",
    "..oRRRRRRRRRRRRRyyyyRRRRRRRRRRRRRo......",
    ".oyyRRRRRRRRRRRRRyyRRRRRRRRRRRRRyyo.....",
    ".oyyyRRRRRRRooooooooooooRRRRRRRyyyo.....",
    ".oyyyRRRRRRowwowwwwwwowwoRRRRRRyyyo.....",
    "..oyyRRRRRRoqqqqqqqqqqqqoRRRRRRyyo......",
    "..oyyRRRRRRowwowwwwwwowwoRRRRRRyyo......",
    "...oyRRRRRRRooooooooooooRRRRRRRyo.......",
    "...ooRRRRRRRRRRRRRRRRRRRRRRRRRRoo.......",
    "....oqRRRRRRRRRRRRRRRRRRRRRRRRqo........",
    "....oqqRRRyyyRRRRRRRRRRyyyRRRqqo........",
    "...oqqqqRRRRRRRRRRRRRRRRRRRRqqqqo.......",
    "..oqqooqqqqRRRRRRRRRRRRRRqqqqooqqo......",
    "..oqo..ooqqqqqqqqqqqqqqqqqqoo..oqo......",
    "..ooo....ooqqqqqqqqqqqqqqoo....ooo......",
    ".........oyyyyo......oyyyyo.............",
    "........oyyYYyyo....oyyYYyyo............",
    "........owowowo......owowowo............",
    "........ooooooo......ooooooo............",
])
spr("nian2", [
    "......oo..................oo............",
    ".....oyyo................oyyo...........",
    ".....oyYyo..............oyYyo...........",
    "......oyYyoo..........ooyYyo............",
    ".......oyyYyoooooooooooyYyyo............",
    "........ooRRRRRRRRRRRRRRRRoo............",
    "......ooRRRRRRRRRRRRRRRRRRRRoo..........",
    ".....oRRRyyyRRRRRRRRRRRRyyyRRRo.........",
    "....oRRRyywwyRRRRRRRRRRywwyyRRRo........",
    "...oRRRRywooyRRRRRRRRRRyoowyRRRRo.......",
    "...oRRRRyyooyRRRRyyRRRRyooyyRRRRo.......",
    "..oRRRRRRyyyRRRRyyyyRRRRyyyRRRRRRo......",
    "..oRRRRRRRRRRRRRyyyyRRRRRRRRRRRRRo......",
    ".oyyRRRRRRRRRRRRRyyRRRRRRRRRRRRRyyo.....",
    ".oyyyRRRRRRRooooooooooooRRRRRRRyyyo.....",
    ".oyyyRRRRRRowwowwwwwwowwoRRRRRRyyyo.....",
    "..oyyRRRRRRoqqqqqqqqqqqqoRRRRRRyyo......",
    "..oyyRRRRRRowwowwwwwwowwoRRRRRRyyo......",
    "...oyRRRRRRRooooooooooooRRRRRRRyo.......",
    "...ooRRRRRRRRRRRRRRRRRRRRRRRRRRoo.......",
    "....oqRRRRRRRRRRRRRRRRRRRRRRRRqo........",
    "....oqqRRRyyyRRRRRRRRRRyyyRRRqqo........",
    "...oqqqqRRRRRRRRRRRRRRRRRRRRqqqqo.......",
    "..oqqooqqqqRRRRRRRRRRRRRRqqqqooqqo......",
    "..oqo..ooqqqqqqqqqqqqqqqqqqoo..oqo......",
    "..ooo....ooqqqqqqqqqqqqqqoo....ooo......",
    "......oyyyyo..............oyyyyo........",
    ".....oyyYYyyo............oyyYYyyo.......",
    ".....owowowo..............owowowo.......",
    ".....ooooooo..............ooooooo.......",
])

# ─────────────────────────── items ───────────────────────────

spr("bun", [
    ".......oo.......",
    "......owwo......",
    "....oowWwwoo....",
    "...owwWwwWwwo...",
    "..owwwWwWwwwwo..",
    ".owwwwwWwwwwwwo.",
    ".owwwwwwwwwwwwo.",
    "owwwwwwwwwwwwwwo",
    "owwwwrwwwwwrwwwo",
    "owwwwwwoowwwwwwo",
    "oWwwwwwwwwwwwwWo",
    ".oWWwwwwwwwwWWo.",
    "..ooWWWWWWWWoo..",
    "....oooooooo....",
])
spr("chili", [
    "......oo..oo",
    ".....ogGoogo",
    "....ogGGGgo.",
    "...oRRgGgo..",
    "..oRwRRRRo..",
    ".oRwRRRRRo..",
    ".oRRRRRRqo..",
    "oRRRRRRqo...",
    "oRRRRRqqo...",
    "oRRRRqqo....",
    ".oRRqqo.....",
    ".oRqqo......",
    "..oqo.......",
    "...o........",
])
spr("bell", [
    ".......oo.......",
    "......oyyo......",
    "......oyyo......",
    "....ooyyyyoo....",
    "...oyywyyyyYo...",
    "..oyywyyyyyyYo..",
    "..oywyyyyyyyYo..",
    "..oywyyyyyyyYo..",
    ".oyyyyyyyyyyyYo.",
    ".oyYYYYYYYYYYYo.",
    "oyyyyyyyyyyyyyYo",
    "oYYYYYYYYYYYYYYo",
    ".oooooyyyyooooo.",
    "......oYYo......",
    ".......oo.......",
])
spr("haw", [
    ".......oo...",
    "......oRRo..",
    ".....oRwRRo.",
    ".....oRRRqo.",
    "......oqqo..",
    "......oRRo..",
    ".....oRwRRo.",
    ".....oRRRqo.",
    "......oqqo..",
    "......oRRo..",
    ".....oRwRRo.",
    ".....oRRRqo.",
    "......oqqo..",
    ".......ol...",
    ".......ol...",
    ".......oo...",
])
spr("lamp", [
    "......oooo......",
    ".......oo.......",
    "....oooooooo....",
    "...oyyyyyyyyo...",
    "..oRRRRRRRRRRo..",
    ".oRRwRRRRRRRqRo.",
    ".oRwRRyyyyRRqRo.",
    "oRRwRyFFFFyRRqRo",
    "oRRRRyFwwFyRRqRo",
    "oRRRRyFFFFyRRqRo",
    ".oRRRRyyyyRRqRo.",
    ".oRRRRRRRRRRqRo.",
    "..oRRRRRRRRRRo..",
    "...oyyyyyyyyo...",
    "....oooooooo....",
    ".......yy.......",
    "......yRRy......",
    ".......yy.......",
])
for i, rows in enumerate([
    [
        "....oooooo......".replace("......", "..")[:12],
    ]
]):
    pass

spr("coin1", [
    "...oooooo...",
    "..oyyyyyyo..",
    ".oyyYYYYyyo.",
    "oyyYooooYyyo",
    "oyyYo..oYyYo",
    "oyyYo..oYyYo",
    "oyyYooooYyYo",
    ".oyyYYYYyYo.",
    "..oyyyyyYo..",
    "...oooooo...",
])
spr("coin2", [
    "....oooo....",
    "...oyyyyo...",
    "..oyyYYyyo..",
    "..oyYooYyo..",
    "..oyYo.oYo..",
    "..oyYo.oYo..",
    "..oyYooYyo..",
    "..oyyYYyYo..",
    "...oyyyYo...",
    "....oooo....",
])
spr("coin3", [
    ".....oo.....",
    "....oyYo....",
    "....oyYo....",
    "....oyYo....",
    "....oyYo....",
    "....oyYo....",
    "....oyYo....",
    "....oyYo....",
    "....oyYo....",
    ".....oo.....",
])

# ─────────────────────────── tiles (16×16) ───────────────────────────

BOX_BASE = [
    "oooooooooooooooo",
    "olllllllllllllNo",
    "olyyyyyyyyyyyyNo",
    "olyNNNNNNNNNNyNo",
    "olyNooooooooNyNo",
    "olyNoyyyyyyoNyNo",
    "olyNoyoooyoNNyNo",
    "olyNoyo..yoNNyNo",
    "olyNoyo..yoNNyNo",
    "olyNoyoooyoNNyNo",
    "olyNoyyyyyyoNyNo",
    "olyNooooooooNyNo",
    "olyNNNNNNNNNNyNo",
    "olyyyyyyyyyyyyNo",
    "oNNNNNNNNNNNNNNo",
    "oooooooooooooooo",
]
spr("box1", BOX_BASE)
spr("box2", [r.replace("y", "F") if 2 <= i <= 13 else r for i, r in enumerate(BOX_BASE)])
spr("box3", [r.replace("y", "Y") if 2 <= i <= 13 else r for i, r in enumerate(BOX_BASE)])
spr("used", [
    "oooooooooooooooo",
    "oxzzzzzzzzzzzzXo",
    "ozxxxxxxxxxxxxXo",
    "ozxoxxxxxxxxoxXo",
    "ozxxxxxxxxxxxxXo",
    "ozxxxxxxxxxxxxXo",
    "ozxxxxxxxxxxxxXo",
    "ozxxxxxxxxxxxxXo",
    "ozxxxxxxxxxxxxXo",
    "ozxxxxxxxxxxxxXo",
    "ozxxxxxxxxxxxxXo",
    "ozxxxxxxxxxxxxXo",
    "ozxoxxxxxxxxoxXo",
    "ozxxxxxxxxxxxxXo",
    "oXXXXXXXXXXXXXXo",
    "oooooooooooooooo",
])

json_out = {"palette": PALETTE, "heroes": HERO_PALETTES, "chili": CHILI, "sprites": S}


def write_ts(path):
    lines = [
        "// Generated by tools/hutu_sprites.py — edit the pixel art there, then run the script.",
        "/* eslint-disable */",
        f"export const PALETTE: Record<string, string> = {json.dumps(PALETTE)}",
        f"export const HERO_PALETTES: Record<string, Record<string, string>> = {json.dumps(HERO_PALETTES)}",
        f"export const CHILI_PALETTE: Record<string, string> = {json.dumps(CHILI)}",
        "export const SPRITES: Record<string, string[]> = {",
    ]
    for k, rows in S.items():
        lines.append(f"  {k}: [")
        for r in rows:
            lines.append(f"    '{r}',")
        lines.append("  ],")
    lines.append("}")
    with open(path, "w") as f:
        f.write("\n".join(lines) + "\n")


def hexrgb(h):
    h = h.lstrip("#")
    return tuple(int(h[i:i + 2], 16) for i in (0, 2, 4)) + (255,)


def preview(path):
    from PIL import Image

    def render(rows, pal):
        im = Image.new("RGBA", (len(rows[0]), len(rows)), (0, 0, 0, 0))
        for y, r in enumerate(rows):
            for x, ch in enumerate(r):
                if ch == ".":
                    continue
                col = pal.get(ch) or PALETTE.get(ch)
                im.putpixel((x, y), hexrgb(col))
        return im

    items = []
    for hero in ("tiger", "rabbit"):
        for chili in (False, True):
            pal = dict(PALETTE)
            pal.update(HERO_PALETTES[hero])
            if chili:
                pal.update(CHILI)
            head = S["head_" + hero]
            ears = S["ears_rabbit"] if hero == "rabbit" else []
            for body in [k for k in S if k.startswith("sb_")] + [k for k in S if k.startswith("bb_")]:
                if chili and body.startswith("sb_"):
                    continue
                rows = ears + head + S[body]
                items.append(render(rows, pal))
    for k, rows in S.items():
        if k.startswith(("head_", "ears_", "sb_", "bb_")):
            continue
        items.append(render(rows, PALETTE))
    scale = 5
    pad = 4
    cols = 12
    cw = max(i.width for i in items) + pad
    ch = max(i.height for i in items) + pad
    rows_n = (len(items) + cols - 1) // cols
    sheet = Image.new("RGBA", (cols * cw, rows_n * ch), (120, 170, 220, 255))
    for n, im in enumerate(items):
        x = (n % cols) * cw
        y = (n // cols) * ch + (ch - pad - im.height)
        sheet.paste(im, (x, y), im)
    sheet = sheet.resize((sheet.width * scale, sheet.height * scale), Image.NEAREST)
    sheet.save(path)
    print("preview", path, sheet.size)


if __name__ == "__main__":
    here = os.path.dirname(os.path.abspath(__file__))
    out = os.path.join(here, "..", "frontend", "src", "games", "hutu", "spriteData.ts")
    write_ts(out)
    print("wrote", os.path.normpath(out), len(S), "sprites")
    if "--preview" in sys.argv:
        preview(sys.argv[sys.argv.index("--preview") + 1])
