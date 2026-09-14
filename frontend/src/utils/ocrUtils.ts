export interface WordToken {
  id: string;
  text: string;
  trailingSpace: string;
  isSuspicious: boolean;
}

/**
 * Heuristically detects words that are likely OCR misrecognitions or gibberish.
 * e.g., accidental letter-digit mixtures like '1OO.B5', 'O3/12', symbol clutter, etc.
 */
export function isSuspiciousWord(raw: string): boolean {
  const word = raw.trim();
  if (!word || word.length < 2) return false;

  // 1. Mixed letters and digits in non-standard patterns (e.g., "1O0", "B500", "0CR", "l999")
  // Exclude standard identifiers like "v2", "3rd", "1st", "10am", "24px"
  const hasDigit = /\d/.test(word);
  const hasLetter = /[a-zA-Z]/.test(word);

  if (hasDigit && hasLetter) {
    // Common benign suffixes
    if (/^\d+(st|nd|rd|th|am|pm|kg|g|mg|ml|km|m|cm|mm|px|em|rem|%|k|m|b)$/i.test(word)) {
      return false;
    }
    // Version tags like v1, v2.0
    if (/^v\d+(\.\d+)*$/i.test(word)) {
      return false;
    }
    // Model/code patterns like GPT-4, H2O, 4K
    if (/^[A-Z]{1,3}-\d+$/i.test(word) || /^\d+[Kk]$/.test(word)) {
      return false;
    }
    // Any other mixed digit/letter is highly suspicious in scanned OCR text (e.g., $1OO.B5, O3/12)
    return true;
  }

  // 2. OCR punctuation noise (stray bars, backslashes, tildes, carats, random underscores)
  if (/[\\|~^_{}[\]]/.test(word) && !/^https?:\/\//.test(word)) {
    return true;
  }

  // 3. Repeated non-standard symbols (e.g., "..,,", "??!!", "///")
  if (/([^\w\s])\1{2,}/.test(word)) {
    return true;
  }

  // 4. Excessive consonant clustering in Latin text (4+ consecutive consonants without vowel/digit)
  // Excludes common English words like "strength", "lengths", "schmuck"
  const cleanWord = word.replace(/^[^\w]+|[^\w]+$/g, '').toLowerCase();
  if (
    cleanWord.length >= 5 &&
    /[bcdfghjklmnpqrstvwxyz]{5,}/i.test(cleanWord) &&
    !/^(length|strength|catch|match|switch|stretch|sch)/.test(cleanWord)
  ) {
    return true;
  }

  return false;
}

/**
 * Tokenizes a paragraph of text into word tokens while preserving spacing,
 * allowing lossless reconstruction.
 */
export function tokenizeParagraph(paraText: string, blockIndex: number): WordToken[] {
  if (!paraText) return [];

  // Match words and their following whitespace
  const regex = /(\S+)(\s*)/g;
  const tokens: WordToken[] = [];
  let match: RegExpExecArray | null;
  let wordIdx = 0;

  while ((match = regex.exec(paraText)) !== null) {
    const text = match[1];
    const trailingSpace = match[2];
    tokens.push({
      id: `b${blockIndex}-w${wordIdx}`,
      text,
      trailingSpace,
      isSuspicious: isSuspiciousWord(text),
    });
    wordIdx++;
  }

  return tokens;
}

/**
 * Reconstructs the paragraph text from modified word tokens.
 */
export function reconstructParagraph(tokens: WordToken[]): string {
  return tokens.map((t) => t.text + t.trailingSpace).join('');
}

/**
 * Finds the start and end boundary indices of a word given a position in a string.
 * Used for single-click word selection in textareas.
 */
export function findWordBoundaries(text: string, pos: number): { start: number; end: number } {
  if (!text || pos < 0 || pos > text.length) {
    return { start: pos, end: pos };
  }

  // If clicked directly on whitespace, find the word nearest or return current
  const isWordChar = (ch: string) => /\S/.test(ch);

  // If current char is not a word char, check if preceding char is
  let currentPos = pos;
  if (currentPos >= text.length || !isWordChar(text[currentPos])) {
    if (currentPos > 0 && isWordChar(text[currentPos - 1])) {
      currentPos = currentPos - 1;
    } else {
      return { start: pos, end: pos };
    }
  }

  let start = currentPos;
  while (start > 0 && isWordChar(text[start - 1])) {
    start--;
  }

  let end = currentPos;
  while (end < text.length && isWordChar(text[end])) {
    end++;
  }

  return { start, end };
}

