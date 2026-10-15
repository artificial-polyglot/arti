import subprocess
import os
import numpy
from sqlite_utility import *

#
# This module is probably not useful for arti as a QA product.
# but it here for interest.
# As of Oct 14, 2026, it will not run unless pyctcdecode and nltk are installed.
#

def create_labels(processor):
    vocab = processor.tokenizer.get_vocab()
    sorted_vocab = dict(sorted(vocab.items(), key=lambda item: item[1]))
    labels = list(sorted_vocab.keys())
    if "<1>" not in labels:
        labels.append("<1>") # not sure why here
    if "#" not in labels:
        labels.append("#") # not sure why here
    return labels

def create_ref_words(db_path):
    database = SqliteUtility(db_path)
    query = "SELECT distinct lower(word) FROM words WHERE ttype='W' ORDER BY lower(word)"
    words = database.select(query, ())
    database.close()
    ref_words = [row[0] for row in words]
    return ref_words

def create_lm_corpus(db_path, corpus_file):
    database = SqliteUtility(db_path)
    query = "SELECT script_text FROM scripts ORDER BY script_id"
    rows = database.select(query, ())
    database.close()
    with open(corpus_file, 'w', encoding='utf-8') as f:
        for row in rows:
            f.write(row[0].lower() + '\n')

def create_decoder(type, processor, db_path, directory="."):
    if type == "greedy":
        return None
    import nltk
    from pyctcdecode import build_ctcdecoder
    if type == "simple":
        decoder = build_ctcdecoder(
            labels = create_labels(processor),
            kenlm_model = None,
        )
    elif type == "hotwords":
        decoder = build_ctcdecoder(
            labels = create_labels(processor),
            kenlm_model = None,
            hotwords = create_ref_words(db_path),
            hotword_weight = 20.0  # how strongly to bias toward reference words
        )
    elif type == "kenlm":
        SCRIPT_FILE = "script.txt"
        MODEL_FILE = "model.arpa"
        MODEL_BIN = "model.bin"
        scriptFile = os.path.join(directory, SCRIPT_FILE)
        modelFile = os.path.join(directory, MODEL_FILE)
        modelBin = os.path.join(directory, MODEL_BIN)
        create_lm_corpus(db_path, scriptFile)
        subprocess.run(["kenlm/build/bin/lmplz", "-o", "5",
                            "--text", scriptFile, "--arpa", modelFile], check=True)
        subprocess.run(["kenlm/build/bin/build_binary", modelFile, modelBin], check=True)
        decoder = build_ctcdecoder(
            labels = create_labels(processor),
            kenlm_model = modelBin,
            hotwords = create_ref_words(db_path),
            hotword_weight = 20.0
        )
    else:
        decoder = None
    return decoder
