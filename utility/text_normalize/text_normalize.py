import unicodedata

def normalizeText(text):
    """
    The one place where text is prepared for the vocabulary, the training labels,
    ASR, and forced alignment. Every caller must use this function; otherwise a
    character that NFC changes is in the vocabulary in one form and in the
    labels or alignment targets in another.
    """
    return unicodedata.normalize("NFC", text.lower())