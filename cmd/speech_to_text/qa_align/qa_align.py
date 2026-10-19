import os
os.environ["PYTORCH_ENABLE_MPS_FALLBACK"] = "1"
import sys
import glob
import json
from transformers import Wav2Vec2ForCTC
from transformers import AutoProcessor
from transformers import Wav2Vec2Config
import torch
import torchaudio
import soundfile as sf
#from transformers.models.wav2vec2.modeling_wav2vec2 import WAV2VEC2_ADAPTER_SAFE_FILE
from safetensors.torch import load_file
sys.path.insert(0, os.path.abspath(os.path.join(os.environ['GOPROJ'], 'logger')))
from error_handler import setup_error_handler
sys.path.insert(0, os.path.join(os.environ['GOPROJ'], 'utility', 'text_normalize'))
from text_normalize import normalizeText


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


if len(sys.argv) < 4:
    print("Usage: qa_align.py {iso639-3} {mms|mms_adapters|<model base>} {model directory}", file=sys.stderr)
    sys.exit(1)
lang = sys.argv[1]
model_type = sys.argv[2]
model_dir = sys.argv[3]      # where Go downloaded the trained model; not used for "mms"

def findOne(pattern):
    matches = glob.glob(os.path.join(model_dir, pattern))
    if len(matches) != 1:
        raise FileNotFoundError(f"Expected exactly one {pattern} in {model_dir}, found {len(matches)}: {matches}")
    return matches[0]

if model_type.lower() == "mms":                     # plain mms-1b-all, no training
    modelId = "facebook/mms-1b-all"
    if not isSupportedLanguage(modelId, lang):
        print(lang, "is not supported by", modelId, file=sys.stderr)
    processor = AutoProcessor.from_pretrained(modelId, target_lang=lang)
    model = Wav2Vec2ForCTC.from_pretrained(modelId, target_lang=lang, ignore_mismatched_sizes=True)
else:
    if not os.path.isdir(model_dir):
        raise FileNotFoundError(f"Model directory not found: {model_dir}")
    if glob.glob(os.path.join(model_dir, "adapter.*.safetensors")):
        # MMS adapter (verse- or word-trained) on top of mms-1b-all
        adapterFile = findOne("adapter.*.safetensors")
        processorDir = findOne("processor_*")
        print("Loading adapter", adapterFile, file=sys.stderr, flush=True)
        processor = AutoProcessor.from_pretrained(processorDir, local_files_only=True)
        model = Wav2Vec2ForCTC.from_pretrained("facebook/mms-1b-all", local_files_only=True)
        hiddenSize = model.lm_head.weight.shape[1]
        model.lm_head = torch.nn.Linear(hiddenSize, len(processor.tokenizer))
        result = model.load_state_dict(load_file(adapterFile), strict=False)
        if result.unexpected_keys:
            raise ValueError(f"Adapter file has weights the model doesn't: {result.unexpected_keys[:5]}")
    else:
        # fully fine-tuned model saved with save_pretrained (mms-300m, mms-1b, Omnilingual, raw)
        print("Loading model", model_dir, file=sys.stderr, flush=True)
        processor = AutoProcessor.from_pretrained(model_dir, local_files_only=True)
        config = Wav2Vec2Config.from_pretrained(model_dir, local_files_only=True)
        model = Wav2Vec2ForCTC.from_pretrained(model_dir, config=config, local_files_only=True)
model.eval()

if torch.cuda.is_available():
    device = 'cuda'
elif torch.backends.mps.is_available():
    device = 'mps'
else:
    device = 'cpu'

model = model.to(device)
vocab = processor.tokenizer.get_vocab()
id2char = {v: k for k, v in vocab.items()}
for line in sys.stdin:
    data = json.loads(line)
    audioFile = data['audio']
    reference_text = data['text']
    sample, sampleRate = sf.read(audioFile, dtype="float32")
    if sampleRate != 16000:
        raise ValueError(f"{audioFile}: expected 16000 Hz audio, got {sampleRate}")
    if sample.ndim > 1:          # just in case a clip is stereo
        sample = sample.mean(axis=1)
    inputs = processor(sample, sampling_rate=16_000, return_tensors="pt")
    if model.config.feat_extract_norm == "group": # Needed for wav2vec2ctc models
        inputs.pop("attention_mask", None)
    inputs = ensureMinimumTensorSize(inputs, 3200, 0)
    inputs = {name: tensor.to(device) for name, tensor in inputs.items()}
    with torch.no_grad():
        logits = model(**inputs).logits # [1, T, V]
    log_probs = torch.nn.functional.log_softmax(logits, dim=-1)
    # Create transcript
    pred_ids = torch.argmax(logits, dim=-1)          # [1, T]
    transcript = processor.batch_decode(pred_ids)[0] # collapses repeats, drops blank, '|' -> space
    # Convert reference text to token indices using your adapter vocab
    normalized = normalizeText(reference_text)
    tokens = [vocab[c] for c in normalized if c in vocab]
    char_timestamps = []
    if len(tokens) == 0:
        # Empty (or entirely out-of-vocab) reference text - forced_align chokes
        # on a zero-length target tensor, so skip alignment for this verse
        # rather than crashing the whole run.
        pass
    else:
        tokens_tensor = torch.tensor([tokens]).to(device)
        try:
            # forced_align returns frame-level token alignments
            frame_alignment, scores = torchaudio.functional.forced_align(
                log_probs,
                tokens_tensor,
                blank=processor.tokenizer.pad_token_id  # CTC blank index
            )
            # Each frame is typically 20ms (model-dependent)
            # frame_duration = 0.02  # seconds
            frame_duration = (len(sample) / 16000) / log_probs.shape[1] # recent change Claude 8/24/26
            token_boundaries = torchaudio.functional.merge_tokens(
                frame_alignment[0], scores[0]
            )
            # Collect timestamps and fa_score
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
        except RuntimeError as e:
            # CTC forced_align can't run when the reference text has more
            # characters than the audio segment has frames (short verse,
            # long text) - skip alignment for this verse rather than
            # crashing the whole run.
            print(f"forced_align failed, skipping verse: {e}", file=sys.stderr)
    result = {
        "transcript": transcript,
        "alignment": char_timestamps,
    }
    sys.stdout.write(json.dumps(result))
    sys.stdout.write("\n")
    sys.stdout.flush()

## Testing
## cd Documents/go2/dataset/mms
## conda activate mms_asr
## python mms_asr.py eng
## /Users/gary/FCBH2024/download/ENGWEB/ENGWEBN2DA-mp3-64/B02___01_Mark________ENGWEBN2DA.wav



