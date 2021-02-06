package markdown

// Markdown Parsing ist relativ lokal:
// keine Änderung ganz am Anfang hat Auswirkungen auf Blocks weit hinten
// es genügt, den aktuellen Block-Kontext zu ermitteln
// es genügt nicht, Leerzeilen zu bringen

// Process markdown input

/* https://github.com/russross/blackfriday
 */

// MarkDown EBNF Grammar
//
// https://github.github.com/gfm/
// https://gist.github.com/michaeljaggers/e4b4a94c5caa3ce5788c47f838149b37
// https://www.w3.org/community/markdown/wiki/EbnfGrammar
// https://www.markdownguide.org/basic-syntax
