import os
import sys
import json
from datasets import Dataset, Audio
from transformers import Wav2Vec2ForCTC
from transformers import AutoProcessor
import torch
import torchaudio
from transformers.models.wav2vec2.modeling_wav2vec2 import WAV2VEC2_ADAPTER_SAFE_FILE
from safetensors.torch import load_file
sys.path.insert(0, os.path.abspath(os.path.join(os.environ['GOPROJ'], 'logger')))
from error_handler import setup_error_handler


## Documentation used to write this program
## https://huggingface.co/docs/transformers/main/en/model_doc/mms
## This program is NOT reentrant because of torch.cuda.empty_cache()

setup_error_handler()

def isSupportedLanguage(modelId:str, lang:str):
    processor = AutoProcessor.from_pretrained(modelId)
    dict = processor.tokenizer.vocab.keys()
    for l in dict:
        if l == lang:
            return True
    return False

def ensureMinimumTensorSize(batch, minTensorLength, padValue):
    tensor = batch.input_values
    batch_size = tensor.shape[0]
    originalLen = tensor.shape[-1]
    if originalLen < minTensorLength:
        paddingNeeded = minTensorLength - originalLen
        tensor = torch.nn.functional.pad(tensor, (0, paddingNeeded), mode='constant', value=padValue)
        batch['input_values'] = tensor
        mask = torch.zeros((batch_size, minTensorLength), dtype=torch.long)
        mask[:, :originalLen] = 1
        batch['attention_mask'] = mask
    return batch

if len(sys.argv) < 2:
    print("Usage: qa_align.py {iso639-3} [adapter]", file=sys.stderr)
    sys.exit(1)
lang = sys.argv[1]
adapter = len(sys.argv) > 2 and sys.argv[2].lower() == "adapter"
if torch.cuda.is_available():
    device = 'cuda'
elif torch.backends.mps.is_available():
    device = 'mps'
else:
    device = 'cpu'
modelId = "facebook/mms-1b-all"
if adapter:
    outputDir = os.path.join(os.getenv('FCBH_DATASET_DB'), 'mms_adapters', lang)
    processorDir = os.path.join(outputDir, f"processor_{lang}")
    processor = AutoProcessor.from_pretrained(processorDir)
    model = Wav2Vec2ForCTC.from_pretrained(modelId, local_files_only=True)
    vocabSize = len(processor.tokenizer)
    hiddenSize = model.lm_head.weight.shape[1]
    model.lm_head = torch.nn.Linear(hiddenSize, vocabSize)
    adapterFile = WAV2VEC2_ADAPTER_SAFE_FILE.format(lang)
    adapterFile = os.path.join(outputDir, adapterFile)
    adapter_weights = load_file(adapterFile)
    model.load_state_dict(adapter_weights, strict=False)
else:
    if not isSupportedLanguage(modelId, lang):
        print(lang, "is not supported by", modelId, file=sys.stderr)
    processor = AutoProcessor.from_pretrained(modelId, target_lang=lang)
    model = Wav2Vec2ForCTC.from_pretrained(modelId, target_lang=lang, ignore_mismatched_sizes=True)

model = model.to(device)
vocab = processor.tokenizer.get_vocab()
id2char = {v: k for k, v in vocab.items()}
for line in sys.stdin:
    torch.cuda.empty_cache()
    data = json.loads(line)
    audioFile = data['audio']
    reference_text = data['text']
    fromDict = Dataset.from_dict({"audio": [audioFile]})
    streamData = fromDict.cast_column("audio", Audio(sampling_rate=16000))
    sample = next(iter(streamData))["audio"]["array"]
    inputs = processor(sample, sampling_rate=16_000, return_tensors="pt")
    inputs = ensureMinimumTensorSize(inputs, 3200, 0)
    inputs = {name: tensor.to(device) for name, tensor in inputs.items()}
    with torch.no_grad():
        logits = model(**inputs).logits # [1, T, V]
    log_probs = torch.nn.functional.log_softmax(logits, dim=-1)
    # Convert reference text to token indices using your adapter vocab
    normalized = reference_text.replace(' ', '|').lower()
    tokens = [vocab[c] for c in normalized if c in vocab]
    tokens_tensor = torch.tensor([tokens]).to(device)
    # forced_align returns frame-level token alignments
    frame_alignment, scores = torchaudio.functional.forced_align(
        log_probs,
        tokens_tensor,
        blank=processor.tokenizer.pad_token_id  # CTC blank index
    )
    # Each frame is typically 20ms (model-dependent)
    frame_duration = 0.02  # seconds
    token_boundaries = torchaudio.functional.merge_tokens(
        frame_alignment[0], scores[0]
    )
    # Collect timestamps and fa_score
    char_timestamps = []
    for token in token_boundaries:
        char = id2char[token.token]
        start = token.start * frame_duration
        end = token.end * frame_duration
        char_timestamps.append({
            'char': char,
            'start': start,
            'end': end,
            'score': token.score
        })
    sys.stdout.write(json.dumps(char_timestamps))
    sys.stdout.write("\n")
    sys.stdout.flush()

## Testing
## cd Documents/go2/dataset/mms
## conda activate mms_asr
## python mms_asr.py eng
## /Users/gary/FCBH2024/download/ENGWEB/ENGWEBN2DA-mp3-64/B02___01_Mark________ENGWEBN2DA.wav



