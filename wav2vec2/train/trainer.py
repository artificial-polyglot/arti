import os
os.environ["PYTORCH_ENABLE_MPS_FALLBACK"] = "1"
import time
import json
import torch
import torch.nn as nn
from torch.utils.data import DataLoader
from transformers import get_linear_schedule_with_warmup
import logging
# from train_adapter
import sys
import transformers
from transformers import Wav2Vec2FeatureExtractor
from transformers import Wav2Vec2Processor
from transformers import Wav2Vec2ForCTC
from tokenizer import createTokenizer
from safetensors.torch import save_file as safe_save_file
from transformers.models.wav2vec2.modeling_wav2vec2 import WAV2VEC2_ADAPTER_SAFE_FILE
from sqlite_utility import *
from data_preparation import *
from dataset import *
from debug import *
from model import getWav2Vec2ForCTCModel
sys.path.insert(0, os.path.abspath(os.path.join(os.environ['GOPROJ'], 'logger')))
from error_handler import setup_error_handler

#
# https://docs.pytorch.org/tutorials/beginner/introyt/trainingyt.html
#

setup_error_handler()
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def train_wav2vec2(model, dataset, num_epochs=3, lr=5e-5,
                     warmup_steps=100, max_grad_norm=1.0, log_steps=50, use_bf16=False):
    """
    Train Wav2Vec2 Model with PyTorch
    Args:
        model: Wav2Vec2 model
        dataset: Your custom dataset that returns padded batches as dicts
        num_epochs: Number of training epochs
        lr: Learning rate
        warmup_steps: Warmup steps for lr scheduler
        max_grad_norm: Max gradient norm for clipping
        log_steps: Steps between logging
        use_bf16: Run the forward pass under bf16 autocast (CUDA only, no GradScaler needed)
    """
    dataloader = DataLoader(dataset, batch_size=None, shuffle=True)

    optimizer = torch.optim.AdamW(model.parameters(), lr=lr)
    total_steps = len(dataloader) * num_epochs
    scheduler = get_linear_schedule_with_warmup(
        optimizer, num_warmup_steps=warmup_steps, num_training_steps=total_steps
    )

    model.train()
    global_step = 0
    total_loss = 0
    epochLosses = []
    for epoch in range(num_epochs):
        logger.info(f"Starting epoch {epoch + 1}/{num_epochs}")
        epoch_loss = 0

        for step, batch in enumerate(dataloader):
            batch = {k: v.to(model.device) for k, v in batch.items()}
            # bf16 autocast covers the forward pass only. The model computes its CTC loss
            # in float32 regardless, and weights, gradients and optimizer state stay float32.
            with torch.autocast(device_type=model.device.type, dtype=torch.bfloat16, enabled=use_bf16):
                outputs = model(**batch)
            loss = outputs.loss
            if not torch.isfinite(loss):
                logger.warning(f"Skipping step {epoch + 1}/{step}: loss is {loss.item()}")
                optimizer.zero_grad()
                continue
            loss.backward()
            gradNorm = torch.nn.utils.clip_grad_norm_(model.parameters(), max_grad_norm)
            if not torch.isfinite(gradNorm):
                logger.warning(f"Skipping step {epoch + 1}/{step}: gradient norm is {gradNorm.item()}")
                optimizer.zero_grad()
                continue
            optimizer.step()
            scheduler.step()
            optimizer.zero_grad()
            epoch_loss += loss.item()
            total_loss += loss.item()
            global_step += 1
            if global_step % log_steps == 0:
                step_label = f"Step {epoch + 1}/{step}"
                avg_loss = total_loss / global_step
                current_lr = scheduler.get_last_lr()[0]
                logger.info(
                    f"{step_label}: Loss = {loss.item():.4f}, "
                    f"Avg Loss = {avg_loss:.4f}, LR = {current_lr:.2e}"
                )
                #memoryStatistics(logger, step_label)
                #modelMemoryStatistics(logger, model, step_label)
                if torch.cuda.is_available():
                    torch.cuda.empty_cache()
                    logger.warning("Cleared CUDA cache due to fragmentation")
                elif torch.backends.mps.is_available():
                    torch.mps.empty_cache()
                    logger.warning("Cleared MPS cache due to fragmentation")
        avg_epoch_loss = epoch_loss / len(dataloader)
        epochLosses.append(round(avg_epoch_loss, 4))
        logger.info(f"Epoch {epoch + 1} completed. Average loss: {avg_epoch_loss:.4f}")

    logger.info("Training completed!")
    return model, epochLosses


# Community conversions can change; pin the tested revision.
PINNED_REVISIONS = {
    "aadel4/omniASR-W2V-300M": "5dabd8d0f1c5c3aa6fb3e0a7dfd613f49104a364",
    "aadel4/omniASR-W2V-1B":   "dad3de98e71b105a2906ac8a80ad67e191ff0550",
    "facebook/mms-300m":       "4ee317ce793c53dbc041fc4376c7558292dd38dc",
    "facebook/mms-1b":         "0d2f7adb9903d98894d70ae11f7fbdfc8cb71a69",
}

if len(sys.argv) < 14:
    usage = """Usage: python train_adapter.py {iso639-3} {databasePath} {audioDirectory} {modelBase}
                    {sampleType} {batchMB} {numEpochs} {learningRage} {warmupPct} {gradNormMax} {minAudioSec}
                    {useBF16} {loggingSteps}"""
    print(usage, file=sys.stderr)
    sys.exit(1)
targetLang = sys.argv[1]
databasePath = sys.argv[2]
audioDirectory = sys.argv[3].strip("'")
modelBase = sys.argv[4]
sampleType = sys.argv[5]
batchSizeMB = int(sys.argv[6])
numEpochs = int(sys.argv[7])
learningRate = float(sys.argv[8])
warmupPct = float(sys.argv[9])
gradNormMax = float(sys.argv[10])
minAudioSec = float(sys.argv[11])
useBF16Arg = sys.argv[12].lower()
loggingSteps = int(sys.argv[13])

print("ModelBase", modelBase, "SampleType", sampleType, "BatchSizeMB", batchSizeMB, "NumEpochs", numEpochs, "learningRate", learningRate,
    "warmupPct", warmupPct, "gradNormMax", gradNormMax, "minAudioSec", minAudioSec, "useBF16", useBF16Arg)

startTime = time.strftime("%Y-%m-%d %H:%M:%S")
torch.manual_seed(42)

database = SqliteUtility(databasePath)
tokenizer = createTokenizer(database, targetLang)

featureExtractor = Wav2Vec2FeatureExtractor(
    feature_size=1,
    sampling_rate=16000,
    padding_value=0.0,
    do_normalize=True,
    return_attention_mask=True
)

processor = Wav2Vec2Processor(
    feature_extractor=featureExtractor,
    tokenizer=tokenizer
)

sampleDB = dataPreparation(database, databasePath, audioDirectory, sampleType, processor, batchSizeMB, minAudioSec)
database.close()

ADAPTER_BASE = "facebook/mms-1b-all"
isAdapter = modelBase == ADAPTER_BASE

if modelBase == "wav2vec2ctc":
    model = getWav2Vec2ForCTCModel(processor)
else:
    model = Wav2Vec2ForCTC.from_pretrained(
        modelBase,
        revision=PINNED_REVISIONS.get(modelBase),
        vocab_size=len(processor.tokenizer),
        pad_token_id=processor.tokenizer.pad_token_id,
        ignore_mismatched_sizes=isAdapter,   # mms-1b-all's own output layer is replaced by ours
        ctc_loss_reduction="mean",    # the model's default is "sum"
        ctc_zero_infinity=True,
#        mask_time_prob=0.05,          # default 0.075
#        mask_time_length=4,           # default 10 frames (200 ms)
#        mask_time_min_masks=0,        # default 2
    )
    if sampleType == "word":
        model.config.mask_time_prob = 0.05
        model.config.mask_time_length = 4
        model.config.mask_time_min_masks = 0
    if isAdapter:
        model.init_adapter_layers()
        model.freeze_base_model()
        for param in model._get_adapters().values():
            param.requires_grad = True
    else:
        model.freeze_feature_encoder()

if torch.cuda.is_available():
    device = torch.device("cuda")
elif torch.backends.mps.is_available():
    device = torch.device("mps")
else:
    device = torch.device("cpu")
print("device", device)
model.to(device)

useBF16 = (useBF16Arg == "true" and device.type == "cuda" and torch.cuda.is_bf16_supported())

dataset = MyDataset(sampleDB)
warmupSteps = int(len(dataset) * numEpochs * warmupPct / 100.0)
print("warmupSteps", warmupSteps)
trainedModel, epochLosses = train_wav2vec2(
        model,
        dataset,
        num_epochs = numEpochs,
        lr = learningRate,
        warmup_steps = warmupSteps,
        max_grad_norm = gradNormMax,
        log_steps = loggingSteps,
        use_bf16 = useBF16
)
numSamples = countSamples(sampleDB)
numTensors = len(dataset)
sampleDB.close()
logger.info("Training completed!")
modelName = modelBase.replace("/", "-")
outputDir = os.path.join(os.getenv('FCBH_DATASET_DB'), modelName, targetLang)
os.makedirs(outputDir, exist_ok=True)
info = {
    "model_base": modelBase,
    "base_revision": PINNED_REVISIONS.get(modelBase) or "main",
    "sample_type": sampleType,
    "bf16": useBF16,
#    "image_version": os.environ.get("ARTI_IMAGE_VERSION", "unknown"),
    "torch": torch.__version__,
    "transformers": transformers.__version__,
    "num_samples": numSamples,                  # after pruning
    "num_batches": numTensors,
    "epoch_losses": epochLosses,                # average loss per epoch
    "started": startTime, "finished": time.strftime("%Y-%m-%d %H:%M:%S"),
}
with open(os.path.join(outputDir, "training_info.json"), "w") as f:
    json.dump(info, f, indent=2)
if isAdapter:
    adapterFile = os.path.join(outputDir, WAV2VEC2_ADAPTER_SAFE_FILE.format(targetLang))
    safe_save_file(model._get_adapters(), adapterFile, metadata={"format": "pt"})
    processor.save_pretrained(os.path.join(outputDir, "processor_" + targetLang))
else:
    model.save_pretrained(outputDir)
    processor.save_pretrained(outputDir)  # Saves both feature extractor and tokenizer
logger.info(f"Model and processor saved to {outputDir}")


