/*
Links

Eigenen Text-Processor schreiben, nur für Latin Script, in pur Go?
Alternative zu Harfbuzz; also Latin-Harfbuzz für Arme in Go?
Siehe
https://docs.microsoft.com/en-us/typography/opentype/spec/ttochap1#text-processing-with-opentype-layout-fonts

Text processing with OpenType Layout fonts

A text-processing client follows a standard process to convert the string of characters entered by a user into positioned glyphs. To produce text with OpenType Layout fonts:

* Using the 'cmap' table in the font, the client converts the character codes into a string of glyph indices.

* Using information in the GSUB table, the client modifies the resulting string, substituting positional or vertical glyphs, ligatures, or other alternatives as appropriate.

* Using positioning information in the GPOS table and baseline offset information in the BASE table, the client then positions the glyphs.

* Using design coordinates the client determines device-independent line breaks. Design coordinates are high-resolution and device-independent.

* Using information in the JSTF table, the client justifies the lines, if the user has specified such alignment.

* The operating system rasterizes the line of glyphs and renders the glyphs in device coordinates that correspond to the resolution of the output device.

Throughout this process the text-processing client keeps track of the association between the character codes for the original text and the glyph indices of the final, rendered text. In addition, the client may save language and script information within the text stream to clearly associate character codes with typographical behavior.

Font Tables

https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2

*/
package glypher
