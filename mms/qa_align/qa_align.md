You don't need to write a decoder. Existing forced alignment code works directly with your model's logits — you just need to feed it the right inputs.
Best option: torchaudio.functional.forced_align
Available in torchaudio 2.1+, works directly with CTC logits:
pythonimport torchaudio
import torch

# Get logits from your adapter model
with torch.no_grad():
logits = model(**inputs).logits  # [1, T, V]

log_probs = torch.nn.functional.log_softmax(logits, dim=-1)

# Convert reference text to token indices using your adapter vocab
vocab = processor.tokenizer.get_vocab()
tokens = [vocab[c] for c in reference_text]  # reference already NFC
tokens_tensor = torch.tensor([tokens])

# forced_align returns frame-level token alignments
frame_alignment, scores = torchaudio.functional.forced_align(
log_probs,
tokens_tensor,
blank=processor.tokenizer.pad_token_id  # CTC blank index
)
Then convert frame alignments to time offsets:
python# Each frame is typically 20ms (model-dependent)
frame_duration = 0.02  # seconds

word_boundaries = torchaudio.functional.merge_tokens(
frame_alignment[0], scores[0]
)
for token in word_boundaries:
start = token.start * frame_duration
end = token.end * frame_duration

## Alternative: ctc-segmentation
A Python library designed specifically for this use case, language-agnostic, works with any CTC model:
python# pip install ctc-segmentation
import ctc_segmentation

# Works with numpy log_probs directly
The key insight: neither library cares about the script or language — they just see token indices and log-probs. Your native-script vocab is transparent to them.
torchaudio.functional.forced_align is probably the cleaner fit given you're already in the PyTorch/HuggingFace ecosystem.