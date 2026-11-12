import os
import shutil
from safetensors.torch import save_file as safe_save_file
from transformers.models.wav2vec2.modeling_wav2vec2 import WAV2VEC2_ADAPTER_SAFE_FILE

# Each epoch's model is saved under FCBH_DATASET_TMP, not in the model directory, because
# controller.go uploads the whole model directory to R2.  Only the best and the latest
# epochs are kept.  At the end, the chosen epoch is copied into the model directory.


def saveModel(model, processor, directory, isAdapter, targetLang):
    os.makedirs(directory, exist_ok=True)
    if isAdapter:
        adapterFile = os.path.join(directory, WAV2VEC2_ADAPTER_SAFE_FILE.format(targetLang))
        safe_save_file(model._get_adapters(), adapterFile, metadata={"format": "pt"})
        processor.save_pretrained(os.path.join(directory, "processor_" + targetLang))
    else:
        model.save_pretrained(directory)
        processor.save_pretrained(directory)  # Saves both feature extractor and tokenizer


class CheckpointManager:
    def __init__(self, modelName, targetLang, processor, isAdapter):
        self.root = os.path.join(os.getenv('FCBH_DATASET_TMP'), 'checkpoints', modelName, targetLang)
        shutil.rmtree(self.root, ignore_errors=True)   # never mix in epochs of an earlier run
        os.makedirs(self.root, exist_ok=True)
        self.targetLang = targetLang
        self.processor = processor
        self.isAdapter = isAdapter

    def epochDir(self, epoch):
        return os.path.join(self.root, f"epoch_{epoch:03d}")

    def save(self, model, epoch, keepEpochs):
        """Saves this epoch, then deletes every saved epoch not in keepEpochs."""
        saveModel(model, self.processor, self.epochDir(epoch), self.isAdapter, self.targetLang)
        keep = {os.path.basename(self.epochDir(e)) for e in keepEpochs if e is not None}
        for name in os.listdir(self.root):
            if name not in keep:
                shutil.rmtree(os.path.join(self.root, name), ignore_errors=True)

    def publish(self, epoch, outputDir):
        """Copies the saved epoch into the model directory that is uploaded to R2."""
        os.makedirs(outputDir, exist_ok=True)
        shutil.copytree(self.epochDir(epoch), outputDir, dirs_exist_ok=True)

    def cleanup(self):
        shutil.rmtree(self.root, ignore_errors=True)
