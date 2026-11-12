import os
import sys
import torch
from io import BytesIO
from data_preparation import encodeSample, calcMinTensor
from common import ensureMinimumTensorSize
sys.path.insert(0, os.path.join(os.environ['GOPROJ'], 'utility', 'text_normalize'))
from text_normalize import normalizeText

# The validation set is the samples that pruning removed from training, so the model never trains
# on them.  Each sample is in the "fa_score" group (pruned_data), the "silence" group
# (pruned_silence), or both.  "all" is every validation sample counted once.
# Headings (verse_str '0') are left out: they are pruned because they are headings, not because
# of their fa_score, and they are often not read.
# Word samples have no validation set; training then runs every epoch.

VALIDATION_GROUPS = ("fa_score", "silence", "all")
VALIDATION_MAX_BATCH = 64

PHRASE_QUERY = """
    SELECT s.book_id || ' ' || s.chapter_num || ':' || s.verse_str || '.' || p.phrase_num AS ref,
        s.audio_file, p.begin_ts, p.end_ts, p.text,
        p.phrase_id IN (SELECT phrase_id FROM pruned_data) AS fa_pruned,
        p.phrase_id IN (SELECT phrase_id FROM pruned_silence) AS silence_pruned
    FROM phrases p
    JOIN scripts s ON s.script_id = p.script_id
    WHERE s.verse_str != '0'
    AND (p.phrase_id IN (SELECT phrase_id FROM pruned_data)
        OR p.phrase_id IN (SELECT phrase_id FROM pruned_silence))
    ORDER BY p.end_ts - p.begin_ts
    """

VERSE_QUERY = """
    SELECT s.book_id || ' ' || s.chapter_num || ':' || s.verse_str AS ref,
        s.audio_file, s.script_begin_ts, s.script_end_ts, GROUP_CONCAT(w.word, ' ' ORDER BY w.word_id) AS text,
        s.script_id IN (SELECT script_id FROM pruned_data) AS fa_pruned,
        s.script_id IN (SELECT script_id FROM pruned_silence) AS silence_pruned
    FROM scripts s
    JOIN words w ON w.script_id = s.script_id
    WHERE w.ttype = 'W' AND s.verse_str != '0'
    AND (s.script_id IN (SELECT script_id FROM pruned_data)
        OR s.script_id IN (SELECT script_id FROM pruned_silence))
    GROUP BY s.script_id
    ORDER BY s.script_end_ts - s.script_begin_ts
    """


class ValidationSample:
    def __init__(self, reference, inputValues, attentionMask, text, faPruned, silencePruned):
        self.reference = reference
        self.inputValues = inputValues          # 1-D float tensor, already normalized by the processor
        self.attentionMask = attentionMask      # 0 for the padding added to reach minAudioSec, as in training
        self.text = text                        # normalized reference text
        self.faPruned = faPruned
        self.silencePruned = silencePruned


def prepareValidationSet(scriptsDB, audioDir, sampleType, processor, minAudioSec):
    """Returns the pruned samples as a list of ValidationSample, or None when sampleType has no
    validation set.  It must run before scriptsDB is closed."""
    if sampleType == "phrase":
        query = PHRASE_QUERY
    elif sampleType == "verse":
        query = VERSE_QUERY
    else:
        return None
    (minTensorLength, _) = calcMinTensor(minAudioSec, torch.float32)
    vocab = processor.tokenizer.get_vocab()
    unkId = processor.tokenizer.unk_token_id
    samples = []
    for (reference, audioFile, beginTS, endTS, text, faPruned, silencePruned) in scriptsDB.select(query, ()):
        if endTS <= beginTS:
            continue
        # encodeSample is the training path, so validation audio is prepared exactly as training audio
        (inputValuesBlob, _, _, _, _) = encodeSample(reference, audioFile, beginTS, endTS, text,
                                                      audioDir, processor, vocab, unkId)
        inputValues = torch.load(BytesIO(inputValuesBlob))
        inputValues, attentionMask = ensureMinimumTensorSize(inputValues, minTensorLength, 0)
        samples.append(ValidationSample(reference, inputValues, attentionMask, cleanText(normalizeText(text)),
                                        bool(faPruned), bool(silencePruned)))
    counts = {g: sum(1 for s in samples if inGroup(s, g)) for g in VALIDATION_GROUPS}
    print("validation samples", counts)
    return samples


def runValidation(model, processor, samples, batchSizeMB, useBF16):
    """Greedy ASR on every validation sample, as mms_asr.py does without a decoder.
    Returns {group: {"cer", "num_samples", "empty_fraction"}} for each of VALIDATION_GROUPS."""
    model.eval()
    transcripts = []
    with torch.no_grad():
        for batch in validationBatches(samples, batchSizeMB):
            features = [{"input_values": s.inputValues, "attention_mask": s.attentionMask} for s in batch]
            padded = processor.pad(features, padding=True, return_tensors="pt")
            inputs = {k: v.to(model.device) for k, v in padded.items()}
            with torch.autocast(device_type=model.device.type, dtype=torch.bfloat16, enabled=useBF16):
                logits = model(**inputs).logits
            ids = torch.argmax(logits, dim=-1).cpu()
            # Frames that come from padding are cut off, so a batch decodes as single samples do.
            frameLens = model._get_feat_extract_output_lengths(padded["attention_mask"].sum(-1))
            for i, sample in enumerate(batch):
                text = processor.decode(ids[i, :int(frameLens[i])])
                transcripts.append((sample, cleanText(text)))
    if torch.cuda.is_available():
        torch.cuda.empty_cache()
    elif torch.backends.mps.is_available():
        torch.mps.empty_cache()
    model.train()
    return summarize(transcripts)


def validationBatches(samples, batchSizeMB):
    """Samples are sorted by length, so each batch is a run of similar lengths padded to its last."""
    ordered = sorted(samples, key=lambda s: s.inputValues.numel())
    batch = []
    for s in ordered:
        paddedMB = s.inputValues.element_size() * s.inputValues.numel() * (len(batch) + 1) / (1024 * 1024)
        if batch and (len(batch) >= VALIDATION_MAX_BATCH or paddedMB > batchSizeMB):
            yield batch
            batch = []
        batch.append(s)
    if batch:
        yield batch


def summarize(transcripts):
    result = {}
    for group in VALIDATION_GROUPS:
        edits = 0
        refLen = 0
        empty = 0
        count = 0
        for sample, asrText in transcripts:
            if not inGroup(sample, group):
                continue
            count += 1
            edits += levenshtein(sample.text, asrText)
            refLen += len(sample.text)
            if asrText == "":
                empty += 1
        result[group] = {
            "cer": round(edits / refLen, 4) if refLen > 0 else None,
            "num_samples": count,
            "empty_fraction": round(empty / count, 4) if count > 0 else None,
        }
    return result


def inGroup(sample, group):
    if group == "fa_score":
        return sample.faPruned
    if group == "silence":
        return sample.silencePruned
    return True


def cleanText(text):
    return " ".join(text.split())


def levenshtein(ref, hyp):
    """Char edit distance, the S+D+I that utility/diff counts for CER."""
    if len(ref) < len(hyp):
        ref, hyp = hyp, ref
    previous = list(range(len(hyp) + 1))
    for i, r in enumerate(ref, 1):
        current = [i]
        for j, h in enumerate(hyp, 1):
            current.append(min(previous[j - 1] + (r != h), previous[j] + 1, current[j - 1] + 1))
        previous = current
    return previous[-1]
