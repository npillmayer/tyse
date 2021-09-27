# OpenType Features

At the heart of OpenType layout lie the **features** of a font. From the OpenType spec:

> The OpenType Layout tables provide typographic information for properly positioning and substituting
> glyphs, operations that are required for accurate typography in many language environments.
> OpenType Layout data is organized by script, language system, typographic feature, and lookup.
> 
> A language system defines features, which are typographic rules for using glyphs to represent a language.
> Sample features are a 'vert' feature that substitutes vertical glyphs in Japanese, a 'liga' feature for
> using ligatures in place of separate glyphs, and a 'mark' feature that positions diacritical marks with
> respect to base glyphs in Arabic. In the absence of language-specific rules, default
> language system features apply to the entire script. For instance, a default language system feature for
> the Arabic script substitutes initial, medial, and final glyph forms based on a glyph’s position in a
> word.

Features relay the heavy lifting to **Lookup**s and lookup-subtables, which come in various flavours
and formats. Package `otlayout` relies on package `ot` to hide some of the nifty details of the
lookup-idiosyncracies, but their application is still somewhat complicated. `otlayout` trys to offer a
uniform API for feature application, alleviating clients from having to consider the details of
all registered OT features.

