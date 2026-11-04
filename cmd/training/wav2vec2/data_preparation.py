import os
import sys
import torch
import torchaudio
import numpy as np
import soundfile as sf
from io import BytesIO
from transformers import Wav2Vec2Processor
from sqlite_utility import *
from common import ensureMinimumTensorSize
sys.path.insert(0, os.path.join(os.environ['GOPROJ'], 'utility', 'text_normalize'))
from text_normalize import normalizeText
import time

"""
NOTE: def dataPruner has a bug.  In a recent run it pruned out only 214 verses when it was
given an input of 0.1, which should have eliminated 10%.
As of 10/28/75, this model is not being used.  So, I have not bothered to fix it.
"""

# This file does all of the data preparation prior to training, and its content is also used for ASR
# Each step performs a process and stores the result in a sqlite database in the TMP directory.
# def prepareDataset() creates a table called samples that contains each word sample in the audio
# the data is a tensor that has been padded to a minimum size of too small, and compressed.
# def identifyBatches identifies the samples that go together in a batch and creates a record
# in sqlite of the samples that go in each batch.
# def prepareBatches() creates the actual tensors, and stores them compressed in a sqlite table.


def dataPreparation(scriptsDB, scriptsDBPath, audioDir, sampleType, processor, targetMemoryMB, minAudioSec):
    (minTensorLength, minTensorMB) = calcMinTensor(minAudioSec, torch.float32)
    print("minAudioSec:", minAudioSec, "minTensorLength:", minTensorLength, "minTensorMB:", minTensorMB)
    scriptsDBName = os.path.basename(scriptsDBPath)
    samplesDBPath = os.path.join(os.getenv('FCBH_DATASET_TMP'), scriptsDBName)
    if os.path.exists(samplesDBPath):
        os.remove(samplesDBPath)
    samplesDB = SqliteUtility(samplesDBPath)
    if sampleType == "verse":
        numSamples = prepareVerseDataset(scriptsDB, samplesDB, audioDir, processor)
        maxBatchSize = 128
    elif sampleType == "word":
        numSamples = prepareWordDataset(scriptsDB, samplesDB, audioDir, processor)
        maxBatchSize = 512
    elif sampleType == "phrase":
          numSamples = preparePhraseDataset(scriptsDB, samplesDB, audioDir, processor)
          maxBatchSize = 248
    else:
        print("Unknown sampleType: " + sampleType, file=sys.stderr, flush=True)
        sys.exit(1)
    # 22 sec
    print("numSamples", numSamples)
    numBatches = identifyBatches(samplesDB, maxBatchSize, targetMemoryMB, minTensorMB)
    print("numBatches", numBatches)
    numTensors = prepareBatches(samplesDB, processor, minTensorLength)
    print("numTensors", numTensors)
    return samplesDB


def calcMinTensor(minAudioSec, dtype):
    minTensorLength = minAudioSec * 16000
    bytesPerElement = torch.finfo(dtype).bits // 8
    minTensorMB = (minTensorLength * bytesPerElement) / (1024 * 1024)
    return (int(minTensorLength), minTensorMB)


"""
The arti program, also called Artificial Polyglot is being used to proof
the correctness of audio files, but there is a fundamental problem with this that
the training might learn the errors, and therefore not be able to identify them as errors.
To mitigate this problem, this step is using data found during forced alignment to remove
words that are likely to have errors.
The forced alignment is returning a pair of timestamps and a probability of correctness for each
character.  I summarize these probabilities of correctness to an average for each word,
and each script line.
"""

def prepareVerseDataset(scriptsDB, samplesDB, audioDir, processor):
    samplesDB.execute('DROP TABLE IF EXISTS samples', ())
    samples = 'CREATE TABLE samples (idx INTEGER PRIMARY KEY, input_values BLOB, labels BLOB, text TEXT, reference TEXT, memory_mb FLOAT)'
    samplesDB.execute(samples,())
    query = """
        SELECT s.book_id || ' ' || s.chapter_num || ':' || s.verse_str as ref,
            s.audio_file, s.script_begin_ts, s.script_end_ts, GROUP_CONCAT(w.word, ' ' ORDER BY w.word_id) AS text
        FROM scripts s
        JOIN words w ON w.script_id = s.script_id
        WHERE w.ttype = 'W' AND s.script_id NOT IN (SELECT script_id FROM pruned_data)
        AND s.script_id NOT IN (SELECT script_id FROM pruned_silence)
        GROUP BY s.book_id, s.chapter_num, s.verse_str, s.audio_file, s.script_begin_ts, s.script_end_ts
        ORDER BY s.script_end_ts - s.script_begin_ts
        """
    index = -1
    vocab = processor.tokenizer.get_vocab()
    unkId = processor.tokenizer.unk_token_id
    numUnk = 0
    missingChars = set()
    data = scriptsDB.select(query,())
    for (reference, audioFile, beginTS, endTS, text) in data:
        if endTS == 0.0:
            continue
        index += 1
        (inputValuesBlob, labelsBlob, memoryMB, unkCount, missing) = encodeSample(
            reference, audioFile, beginTS, endTS, text, audioDir, processor, vocab, unkId)
        numUnk += unkCount
        missingChars.update(missing)
        insert = 'INSERT INTO samples (idx, input_values, labels, text, reference, memory_mb) VALUES (?,?,?,?,?,?)'
        samplesDB.execute(insert, (index, inputValuesBlob, labelsBlob, text, reference, memoryMB))
    if numUnk > 0:
        # Labels containing [UNK] teach the model to output [UNK]; this should never happen
        # because the vocabulary is built from the same text with the same normalizeText.
        chars = ' '.join(f"{ch!r} U+{ord(ch):04X}" for ch in sorted(missingChars))
        print(f"WARNING: {numUnk} [UNK] tokens in training labels. Characters not in vocab: {chars}",
              file=sys.stderr, flush=True)
    return index + 1


def preparePhraseDataset(scriptsDB, samplesDB, audioDir, processor):
    samplesDB.execute('DROP TABLE IF EXISTS samples', ())
    samples = 'CREATE TABLE samples (idx INTEGER PRIMARY KEY, input_values BLOB, labels BLOB, text TEXT, reference TEXT, memory_mb FLOAT)'
    samplesDB.execute(samples,())
    query = """
        SELECT s.script_id, s.book_id || ' ' || s.chapter_num || ':' || s.verse_str as ref,
            s.audio_file, w.word_begin_ts, w.word_end_ts, w.ttype, w.word,
            (s.script_id IN (SELECT script_id FROM pruned_data)
             OR s.script_id IN (SELECT script_id FROM pruned_silence)) AS pruned
        FROM scripts s
        JOIN words w ON w.script_id = s.script_id
        WHERE w.ttype IN ('W', 'P')
        ORDER BY w.script_id, w.word_seq
        """
    phrases = mergeSingleWords(groupPhrases(scriptsDB.select(query,())))
    padded = padWordTimestamps(phrases)
    data = [p for p in padded if not p[1]]     # now drop phrases of pruned verses
    data.sort(key=lambda row: row[5] - row[4]) # sort by size

    index = -1
    vocab = processor.tokenizer.get_vocab()
    unkId = processor.tokenizer.unk_token_id
    numUnk = 0
    missingChars = set()
    for (scriptId, pruned, reference, audioFile, beginTS, endTS, text) in data:
        if endTS <= beginTS:
            continue
        index += 1
        (inputValuesBlob, labelsBlob, memoryMB, unkCount, missing) = encodeSample(
            reference, audioFile, beginTS, endTS, text, audioDir, processor, vocab, unkId)
        numUnk += unkCount
        missingChars.update(missing)
        insert = 'INSERT INTO samples (idx, input_values, labels, text, reference, memory_mb) VALUES (?,?,?,?,?,?)'
        samplesDB.execute(insert, (index, inputValuesBlob, labelsBlob, text, reference, memoryMB))
    if numUnk > 0:
        # Labels containing [UNK] teach the model to output [UNK]; this should never happen
        # because the vocabulary is built from the same text with the same normalizeText.
        chars = ' '.join(f"{ch!r} U+{ord(ch):04X}" for ch in sorted(missingChars))
        print(f"WARNING: {numUnk} [UNK] tokens in training labels. Characters not in vocab: {chars}",
              file=sys.stderr, flush=True)
    return index + 1


def groupPhrases(rows):
    """rows: (scriptId, reference, audioFile, beginTS, endTS, ttype, word, pruned), in reading order.
    Every punctuation token ends a phrase, and so does the end of the script.  Mid-word punctuation is
    part of its word, so it does not break a phrase.  Returns one row per phrase, in order:
    (scriptId, pruned, reference.phraseNum, audioFile, beginTS, endTS, text), the shape padWordTimestamps uses."""
    phrases = []
    current = []
    phraseNum = 0
    for i, (scriptId, reference, audioFile, beginTS, endTS, ttype, word, pruned) in enumerate(rows):
        if ttype == 'W' and endTS > beginTS:     # drop zero-length words (e.g. footnote references)
            current.append((beginTS, endTS, word))
        lastInScript = i + 1 == len(rows) or rows[i + 1][0] != scriptId
        if current and (ttype == 'P' or lastInScript):
            phraseNum += 1
            text = ' '.join(w[2] for w in current)
            phrases.append((scriptId, pruned, f"{reference}.{phraseNum}", audioFile, current[0][0], current[-1][1], text))
            current = []
        if lastInScript:
            phraseNum = 0
    return phrases


def mergeSingleWords(phrases):
    """phrases: the rows groupPhrases returns.  Joins each single-word phrase to the phrase before or after it
    in the same script, whichever is separated from it by the shorter pause, and renumbers the references.
    A script that contains only one word in total stays a single-word phrase."""
    scripts = {}
    for phrase in phrases:
        scripts.setdefault(phrase[0], []).append(phrase)
    result = []
    for verse in scripts.values():
        i = 0
        while i < len(verse) and len(verse) > 1:
            if ' ' in verse[i][6]:
                i += 1
                continue
            toPrev = i == len(verse) - 1 or (i > 0 and verse[i][4] - verse[i - 1][5] <= verse[i + 1][4] - verse[i][5])
            first = i - 1 if toPrev else i
            a, b = verse[first], verse[first + 1]
            verse[first:first + 2] = [a[:4] + (a[4], b[5], a[6] + ' ' + b[6])]
            i = first + 1
        for num, phrase in enumerate(verse, start=1):
            reference = phrase[2].rsplit('.', 1)[0]
            result.append(phrase[:2] + (f"{reference}.{num}",) + phrase[3:])
    return result


def prepareWordDataset(scriptsDB, samplesDB, audioDir, processor):
    samplesDB.execute('DROP TABLE IF EXISTS samples', ())
    samples = """CREATE TABLE samples (idx INTEGER PRIMARY KEY, script_id INTEGER, word_id INTEGER,
            input_values BLOB, labels BLOB, text TEXT, reference TEXT, fa_score FLOAT, memory_mb FLOAT)"""
    samplesDB.execute(samples,())
    query = """
       SELECT s.script_id, w.word_id, s.book_id || ' ' || s.chapter_num || ':' || s.verse_str || '.' || w.word_seq as ref,
               s.audio_file, w.word_begin_ts, w.word_end_ts, w.fa_score, w.word AS text,
               (w.word_id IN (SELECT word_id FROM pruned_silence)
                OR w.word_id IN (SELECT word_id FROM pruned_data)) AS pruned
        FROM scripts s
        JOIN words w ON w.script_id = s.script_id
        WHERE w.ttype = 'W'
        ORDER BY w.word_id
        """
    rows = scriptsDB.select(query,())
    rows = [r for r in rows if r[5] > r[4]]     # drop zero-length words (e.g. footnote references)
    padded = padWordTimestamps([r[:8] for r in rows])
    data = [p for p, r in zip(padded, rows) if not r[8]]   # now drop pruned words
    data.sort(key=lambda row: row[5] - row[4]) # sort by size

    index = -1
    vocab = processor.tokenizer.get_vocab()
    unkId = processor.tokenizer.unk_token_id
    numUnk = 0
    missingChars = set()
    for (scriptId, wordId, reference, audioFile, beginTS, endTS, faScore, text) in data:
        if endTS <= beginTS:
            continue
        index += 1
        (inputValuesBlob, labelsBlob, memoryMB, unkCount, missing) = encodeSample(
            reference, audioFile, beginTS, endTS, text, audioDir, processor, vocab, unkId)
        numUnk += unkCount
        missingChars.update(missing)

        insert = """INSERT INTO samples (idx, script_id, word_id, input_values, labels, text, reference,
            fa_score, memory_mb) VALUES (?,?,?,?,?,?,?,?,?)"""
        samplesDB.execute(insert, (index, scriptId, wordId, inputValuesBlob, labelsBlob, text, reference, faScore, memoryMB))

    if numUnk > 0:
        # Labels containing [UNK] teach the model to output [UNK]; this should never happen
        # because the vocabulary is built from the same text with the same normalizeText.
        chars = ' '.join(f"{ch!r} U+{ord(ch):04X}" for ch in sorted(missingChars))
        print(f"WARNING: {numUnk} [UNK] tokens in training labels. Characters not in vocab: {chars}",
              file=sys.stderr, flush=True)
    return index + 1


def encodeSample(reference, audioFile, beginTS, endTS, text, audioDir, processor, vocab, unkId):
    """Load one span of audio and encode it and its text as the blobs stored in samples."""
    audioFile = audioFile.replace(".mp3", ".wav")
    audioPath = os.path.join(audioDir, audioFile)

    sample_rate = sf.info(audioPath).samplerate
    if sample_rate != 16000:
        print("Audio sample rate must be 16000", file=sys.stderr, flush=True)
        sys.exit(1)

    speech, sample_rate = torchaudio.load(
        audioPath,
        frame_offset = int(beginTS * 16000),
        num_frames = int((endTS - beginTS) * 16000)
    )
    speech = speech.squeeze().numpy()

    inputValues = processor(
            speech,
            sampling_rate=16000,
            return_tensors=None,
            padding=False
        ).input_values
    inputValues = np.array(inputValues)
    inputValuesTensor = torch.tensor(inputValues, dtype=torch.float).squeeze(0)
    memoryMB = inputValuesTensor.element_size() * inputValuesTensor.numel() / (1024 * 1024)
    if memoryMB == 0:
        print(reference, audioFile, "Has Zero Length input_values", file=sys.stderr, flush=True)
        sys.exit(1)

    normalized = normalizeText(text)
    labels = processor(text=normalized).input_ids
    missing = set(ch for ch in normalized if ch != ' ' and ch not in vocab)
    labelsTensor = torch.tensor(labels, dtype=torch.long)

    buffer = BytesIO()
    torch.save(inputValuesTensor, buffer)
    inputValuesBlob = buffer.getvalue()

    buffer = BytesIO()
    torch.save(labelsTensor, buffer)
    labelsBlob = buffer.getvalue()
    return (inputValuesBlob, labelsBlob, memoryMB, labels.count(unkId), missing)


WORD_PAD = 0.20  # seconds of context added on each side, limited to half the gap to the neighbouring word

def padWordTimestamps(rows):
    """rows: (scriptId, wordId, reference, audioFile, beginTS, endTS, faScore, text), ordered by
    word_id and including every word, so each row's neighbours are its real neighbours in the audio."""
    result = []
    for i, row in enumerate(rows):
        audioFile, begin, end = row[3], row[4], row[5]
        prev = rows[i - 1] if i > 0 and rows[i - 1][3] == audioFile else None
        nxt = rows[i + 1] if i + 1 < len(rows) and rows[i + 1][3] == audioFile else None
        newBegin = begin - WORD_PAD
        if prev is not None:
            newBegin = max(newBegin, (prev[5] + begin) / 2)
        newEnd = end + WORD_PAD
        if nxt is not None:
            newEnd = min(newEnd, (end + nxt[4]) / 2)
        result.append(row[:4] + (max(newBegin, 0.0), newEnd) + row[6:])
    return result

def identifyBatches(database, maxBatchSize, targetMemoryMB, minTensorMB):
    count = database.selectOne('SELECT count(*) FROM samples WHERE memory_mb > ?', (targetMemoryMB,))
    if len(count) > 0 and count[0] > 0:
        print(count[0], "samples were dropped, because they were too large.", file=sys.stderr)
    database.execute('DROP TABLE IF EXISTS batches', ())
    batches = """CREATE TABLE batches (idx INTEGER PRIMARY KEY, num_samples INTEGER, memory_mb FLOAT,
                padded_mb FLOAT, indexes BLOB)"""
    database.execute(batches,())
    query = 'SELECT idx, text, reference, memory_mb FROM samples'
    samples = database.select(query, ())
    insert = 'INSERT INTO batches (idx, num_samples, memory_mb, padded_mb, indexes) VALUES (?,?,?,?,?)'
    batchNum = 0
    batch = []
    unpaddedSize = 0.0
    paddedSize = 0.0
    for (index, text, reference, memoryMB) in samples:
        memoryMB = max(memoryMB, minTensorMB)
        if len(batch) >= maxBatchSize or (memoryMB * (len(batch) + 1)) >= targetMemoryMB:
            database.execute(insert, (batchNum, len(batch), unpaddedSize, paddedSize,
                ','.join(map(str, batch))))
            batchNum += 1
            batch = []
            unpaddedSize = 0.0
            paddedSize = 0.0
        batch.append(index)
        unpaddedSize += memoryMB
        paddedSize = len(batch) * memoryMB
    if len(batch) > 0:
        database.execute(insert, (batchNum, len(batch), unpaddedSize, paddedSize,
            ','.join(map(str, batch))))
    return batchNum + 1


def prepareBatches(database, processor, minTensorLength):
    database.execute('DROP TABLE IF EXISTS tensors', ())
    padding: Union[bool, str] = True
    table = """CREATE TABLE tensors (idx INTEGER PRIMARY KEY, num_samples INTEGER, memory_mb FLOAT,
                input_values BLOB, attention_mask BLOB, labels BLOB)"""
    database.execute(table, ())
    batches = database.select('SELECT idx, num_samples, memory_mb, padded_mb, indexes FROM batches', ())
    for (index, numSamples, memoryMB, paddedMB, indexes) in batches:
        #print("\nIndex", index, numSamples, memoryMB, paddedMB, indexes)
        samples = indexes.split(',')
        inputFeatures = []
        labelFeatures = []
        for s in samples:
            sampleQuery = """SELECT idx, input_values, labels, text, reference, memory_mb
                            FROM samples WHERE idx = ?"""
            (sampleIdx, inputValues, labels, text, reference, memoryMB) = database.selectOne(sampleQuery, (int(s),))
            buffer = BytesIO(inputValues)
            inputTensor = torch.load(buffer)

            originalLen = len(inputTensor)
            inputTensor, attentionMask = ensureMinimumTensorSize(inputTensor, minTensorLength, 0)
            inputFeatures.append({
                "input_values": inputTensor,
                "attention_mask": attentionMask
            })
            buffer = BytesIO(labels)
            labelsTensor = torch.load(buffer)
            labelFeatures.append({"input_ids": labelsTensor})
        batch = processor.pad(
            inputFeatures,
            padding = padding,
            return_tensors = "pt",
        )
        labelsBatch = processor.pad(
            labels = labelFeatures,
            padding = padding,
            return_tensors = "pt",
        )
        # replace padding with -100 to ignore loss correctly
        labels = labelsBatch["input_ids"].masked_fill(labelsBatch.attention_mask.ne(1), -100)
        insert = """INSERT INTO tensors (idx, num_samples, memory_mb, input_values,
                            attention_mask, labels) VALUES (?,?,?,?,?,?)"""
        inputValuesBuffer = BytesIO()
        torch.save(batch['input_values'], inputValuesBuffer)
        attentionMaskBuffer = BytesIO()
        torch.save(batch['attention_mask'], attentionMaskBuffer)
        labelsBuffer = BytesIO()
        torch.save(labels, labelsBuffer)
        database.execute(insert, (index, numSamples, paddedMB, inputValuesBuffer.getvalue(),
                attentionMaskBuffer.getvalue(), labelsBuffer.getvalue()))
    return len(batches)


def displaySamples(database):
    query = 'SELECT idx, input_values, labels, text, reference, memory_mb FROM samples'
    samples = database.select(query, ())
    print("length", len(samples))
    for (index, inputValues, labels, text, reference, memoryMB) in samples:
        buffer = BytesIO(inputValues)
        audioTensor = torch.load(buffer)
        buffer = BytesIO(labels)
        labelsTensor = torch.load(buffer)
        print("\nIndex", index)
        print("audio", audioTensor.shape, type(audioTensor))
        print("labels", labelsTensor.shape, type(labelsTensor))
        print("text", text)
        print("reference", reference)
        print("memory_mb", memoryMB)
    print("numSamples", len(samples))

def countSamples(database):
    return database.selectOne('SELECT count(*) FROM samples', ())

def displayBatches(database):
    batches = database.select('SELECT idx, num_samples, memory_mb, padded_mb, indexes FROM batches', ())
    for (index, numSamples, memoryMB, paddedMB, indexes) in batches:
        print("\nIndex", index, numSamples, memoryMB, paddedMB, indexes)
    print("numBatches", len(batches))


def displayTensors(database):
    batches = database.select('SELECT idx, num_samples, memory_mb, input_values, attention_mask, labels FROM tensors', ())
    for (index, numSamples, memoryMB, inputValues, attentionMask, labels) in batches:
        inputValuesBuffer = BytesIO(inputValues)
        audioTensor = torch.load(inputValuesBuffer)
        maskBuffer = BytesIO(attentionMask)
        maskTensor = torch.load(maskBuffer)
        labelsBuffer = BytesIO(labels)
        labelsTensor = torch.load(labelsBuffer)
        print("\nIndex", index)
        print("audio", audioTensor.shape, type(audioTensor), audioTensor.nbytes / (1024**2))
        print("mask", maskTensor.shape, type(maskTensor), maskTensor.nbytes / (1024**2))
        print("labels", labelsTensor.shape, type(labelsTensor), labelsTensor.nbytes / (1024**2))
        print("memory_mb", memoryMB)


def checkBlob(idx, name, blob, dtype):
    buffer = BytesIO(blob)
    tensor = torch.load(buffer)
    if torch.isnan(tensor).any():
        print(f"NaN found in {name} {idx}")
    if torch.isinf(tensor).any():
        print(f"Inf found in {name} {idx}")
    if tensor.numel() == 0:
        print(f"Empty tensor {nme} in sample {idx}")


def checkSamples(samplesDB):
    samples = samplesDB.select('SELECT idx, input_values, labels, text TEXT, reference, memory_mb FROM samples',())
    for (idx, inputValues, labels, text, reference, memoryMB) in samples:
        checkBlob(idx, 'input_values', inputValues, np.float32)
        checkBlob(idx, 'labels', labels, np.int64)


if __name__ == "__main__":
    from tokenizer import createTokenizer
    from transformers import Wav2Vec2Processor, Wav2Vec2FeatureExtractor, Wav2Vec2CTCTokenizer
    dbPath = os.getenv("FCBH_DATASET_DB") + "/GaryNTest/N2KEUWB4.db"
    database = SqliteUtility(dbPath)
    audioPath = os.getenv("FCBH_DATASET_FILES") + "/N2KEUWB4/N2KEUWBT"
    tokenizer = createTokenizer(database, "keu")
    featureExtractor = Wav2Vec2FeatureExtractor(
        feature_size=1, sampling_rate=16000, padding_value=0.0,
        do_normalize=True, return_attention_mask=True
    )
    processor = Wav2Vec2Processor(feature_extractor=featureExtractor, tokenizer=tokenizer)
    samplesDB = dataPreparation(database, dbPath, audioPath, processor, 1024, 16, 1.0, 0.1)
    database.close()
    displaySamples(samplesDB)
    displayBatches(samplesDB)
    displayTensors(samplesDB)
    checkSamples(samplesDB),
    samplesDB.close()