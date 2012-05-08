// Copyright (C) 2011 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/**
 * @fileoverview
 * Registers a language handler for the Go language.
 * <p>
 * Unlike the minimal lang-go.js, this does more semantic highlighting,
 * similar to the highlighting in the emacs go-mode.el.
 * <p>
 *
 * @author felixz@google.com
 */

(function () {
    /** @const */ var KEYWORDS =
        "break case chan const continue " +
        "default defer else fallthrough for " +
        "func go goto if import " +
        "interface map package range return " +
        "select struct switch type var";

    /** @const */ var CONSTANTS =
        "true false iota nil";

    /** @const */ var TYPES =
        "bool byte complex64 complex128 float32 float64 " +
        "int8 int16 int32 int64 string uint16 uint32 uint64 " +
        "int uint uintptr";

    /** @const */ var BUILTINS =
        "append cap close complex copy imag len " +
        "make new panic print println real recover";

    var shortcuts = [];
    var fallthrus = [];

    shortcuts.push(
        [PR['PR_PLAIN'], /^[\t\n\r \xA0]+/, null, '\t\n\r \xA0']);

    // 'chars' and "strings" can't span lines
    shortcuts.push(
        [PR['PR_STRING'], /^\'(?:[^\\\'\r\n]|\\[^\r\n])*(?:\'|$)/,
         null, "'"]);
    shortcuts.push(
        [PR['PR_STRING'], /^\"(?:[^\\\"\r\n]|\\[^\r\n])*(?:\"|$)/,
         null, '"']);

    // `rawstrings` can span lines and do not treat \ specially
    shortcuts.push(
        [PR['PR_STRING'], /^`[^`]*(?:`|$)/,
         null, '`']);

    // numbers that start with \d
    shortcuts.push(
        [PR['PR_LITERAL'],
         new RegExp(
             '^'
             // A hex number
             + '0x[a-f0-9]+'
             // or an octal or decimal number,
             + '|\\d+(?:\\.\\d*)?'
             // possibly in scientific notation
             + '(?:e[+\\-]?\\d+)?'
             // possibly imaginary
             + 'i?', 'i'),
         null, '0123456789']);

    // numbers that start with '.'
    fallthrus.push(
        [PR['PR_LITERAL'], /^\.\d+(?:e[+\-]?\d+)?i?/i]);

    fallthrus.push(
        [PR['PR_COMMENT'], /^\/\/[^\r\n]*/]);
    fallthrus.push(
        [PR['PR_COMMENT'], /^\/\*[\s\S]*?(?:\*\/|$)/]);

    fallthrus.push(
        [PR['PR_KEYWORD'],
         new RegExp('^(?:' + KEYWORDS.replace(/\s+/g, '|') + ')\\b')]);

    fallthrus.push(
        [PR['PR_KEYWORD'],
         new RegExp('^(?:' + BUILTINS.replace(/\s+/g, '|') + ')\\b')]);

    fallthrus.push(
        [PR['PR_TYPE'],
         new RegExp('^(?:' + TYPES.replace(/\s+/g, '|') + ')\\b')]);

    fallthrus.push(
        [PR['PR_LITERAL'],
         new RegExp('^(?:' + CONSTANTS.replace(/\s+/g, '|') + ')\\b')]);

    fallthrus.push(
        [PR['PR_PUNCTUATION'], /^[+\-*\/%&|^=<>()\[\]{}!:.,;]+/]);

    PR['registerLangHandler'](
        PR['createSimpleLexer'](shortcuts, fallthrus),
        ['go-rich', 'go']);
})();
